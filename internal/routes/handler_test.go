package routes

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"kinetix-api/internal/auth"
	"kinetix-api/internal/middleware"
)

var validate = validator.New()

// newRouter wires handler Routes onto a fresh chi router. The service has a
// nil repo so any path that reaches the service would nil-deref — tests only
// exercise paths that fail before reaching the service.
func newRouter() http.Handler {
	svc := NewService(nil, validate)
	h := NewHandler(svc, validate)
	r := chi.NewRouter()
	h.Routes(r)
	return r
}

// authedRouter wraps the routes with JWT auth so we can verify 401 behaviour.
func authedRouter(t *testing.T) http.Handler {
	t.Helper()
	cfg := auth.Config{SecretKey: "test-secret", TokenExpiry: time.Hour}
	return middleware.JWTAuth(cfg)(newRouter())
}

// UploadRouteRequest is exposed as a DTO. Pin its JSON shape so future
// refactors don't silently break the multipart-side mapping in handler.go.
func TestUploadRouteRequest_JSONShape(t *testing.T) {
	eventID := uuid.New()
	distanceID := uuid.New()

	withBoth := UploadRouteRequest{
		EventID:    eventID,
		DistanceID: &distanceID,
		Name:       "Morning Loop",
	}
	data, err := json.Marshal(withBoth)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(data)
	if !contains(s, `"event_id":"`+eventID.String()+`"`) {
		t.Errorf("missing event_id in %s", s)
	}
	if !contains(s, `"distance_id":"`+distanceID.String()+`"`) {
		t.Errorf("missing distance_id in %s", s)
	}
	if !contains(s, `"name":"Morning Loop"`) {
		t.Errorf("missing name in %s", s)
	}

	// distance_id has omitempty → absent from JSON when nil.
	noDistance := UploadRouteRequest{EventID: eventID, Name: "Loop"}
	data2, _ := json.Marshal(noDistance)
	if contains(string(data2), `"distance_id"`) {
		t.Errorf("expected distance_id to be omitted, got %s", data2)
	}
}

// ===== HTTP handler tests =====

func TestGetByID_BadUUIDReturns400(t *testing.T) {
	r := newRouter()
	req := httptest.NewRequest(http.MethodGet, "/not-a-uuid", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad UUID, got %d", rr.Code)
	}
}

func TestDelete_BadUUIDReturns400(t *testing.T) {
	r := newRouter()
	req := httptest.NewRequest(http.MethodDelete, "/not-a-uuid", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad UUID, got %d", rr.Code)
	}
}

// multipartWithField builds a multipart body containing a single text field
// (e.g. event_id). The caller chooses whether to also add a "file" part.
func multipartWithField(t *testing.T, field, value string) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	if err := w.WriteField(field, value); err != nil {
		t.Fatalf("write field: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	return body, w.FormDataContentType()
}

func TestUpload_MissingFileReturns400(t *testing.T) {
	r := newRouter()
	body, contentType := multipartWithField(t, "event_id", uuid.New().String())

	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", contentType)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 when file part missing, got %d (body=%s)", rr.Code, rr.Body.String())
	}
}

func TestUpload_MissingEventIDReturns400(t *testing.T) {
	r := newRouter()
	// Build a multipart with a file but no event_id.
	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	fileW, err := mw.CreateFormFile("file", "route.gpx")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := io.WriteString(fileW, "<gpx></gpx>"); err != nil {
		t.Fatalf("write file: %v", err)
	}
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 when event_id missing, got %d", rr.Code)
	}
}

func TestUpload_BadEventIDReturns400(t *testing.T) {
	r := newRouter()
	body, contentType := multipartWithField(t, "event_id", "not-a-uuid")

	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", contentType)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad event_id, got %d", rr.Code)
	}
}

func TestUpload_BadDistanceIDReturns400(t *testing.T) {
	r := newRouter()
	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	if err := mw.WriteField("event_id", uuid.New().String()); err != nil {
		t.Fatalf("write event_id: %v", err)
	}
	if err := mw.WriteField("distance_id", "not-a-uuid"); err != nil {
		t.Fatalf("write distance_id: %v", err)
	}
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad distance_id, got %d", rr.Code)
	}
}

// ===== Auth tests =====

func TestUpload_NoAuthHeaderReturns401(t *testing.T) {
	h := authedRouter(t)
	body, contentType := multipartWithField(t, "event_id", uuid.New().String())
	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", contentType)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestUpload_BadTokenReturns401(t *testing.T) {
	h := authedRouter(t)
	body, contentType := multipartWithField(t, "event_id", uuid.New().String())
	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", "Bearer not.a.valid.jwt")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestGetByID_NoAuthHeaderReturns401(t *testing.T) {
	h := authedRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/"+uuid.New().String(), nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestDelete_NoAuthHeaderReturns401(t *testing.T) {
	h := authedRouter(t)
	req := httptest.NewRequest(http.MethodDelete, "/"+uuid.New().String(), nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func contains(haystack, needle string) bool {
	return len(needle) == 0 || (len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
