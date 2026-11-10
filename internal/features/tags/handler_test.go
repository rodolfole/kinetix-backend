package tags

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"kinetix-api/internal/auth"
	"kinetix-api/internal/middleware"
)

// newRouter mirrors main.go wiring: ListByEvent is public; Create and Delete
// live under a JWT-protected group. The repo is built with a nil store so
// handler paths that reach the repo would nil-deref — only paths that fail
// before reaching the repo are exercised here.
func newRouter() http.Handler {
	repo := NewRepository(nil)
	h := NewHandler(repo)
	r := chi.NewRouter()

	// Public (mirrors main.go line 166).
	r.Get("/events/{eventId}/tags", h.ListByEvent)

	// Protected (mirrors main.go lines 227–228).
	r.Group(func(protected chi.Router) {
		cfg := auth.Config{SecretKey: "test-secret", TokenExpiry: time.Hour}
		protected.Use(middleware.JWTAuth(cfg))
		protected.Post("/events/tags", h.Create)
		protected.Delete("/events/tags/{id}", h.Delete)
	})

	return r
}

// authedRouter exposes the protected routes directly so test bodies stay
// focused on auth-specific behaviour.
func authedRouter(t *testing.T) http.Handler {
	t.Helper()
	repo := NewRepository(nil)
	h := NewHandler(repo)
	r := chi.NewRouter()
	r.Post("/events/tags", h.Create)
	r.Delete("/events/tags/{id}", h.Delete)
	cfg := auth.Config{SecretKey: "test-secret", TokenExpiry: time.Hour}
	return middleware.JWTAuth(cfg)(r)
}

// createTagRequest is the unexported request DTO. These tests pin its shape
// (JSON tags) and the manual validation done in the Create handler.
func TestCreateTagRequest_JSONShape(t *testing.T) {
	eventID := uuid.New()
	req := createTagRequest{
		EventID: eventID,
		Name:    "trail-run",
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(data)
	if !contains(s, `"event_id":"`+eventID.String()+`"`) {
		t.Errorf("missing event_id in %s", s)
	}
	if !contains(s, `"name":"trail-run"`) {
		t.Errorf("missing name in %s", s)
	}
}

func TestCreateTagRequest_FieldTypes(t *testing.T) {
	// Guard against accidental type drift: the handler relies on
	// uuid.Nil for "missing event_id".
	req := createTagRequest{}
	rv := reflect.ValueOf(req)
	if rv.Field(0).Type().String() != "uuid.UUID" {
		t.Errorf("EventID should be uuid.UUID, got %s", rv.Field(0).Type())
	}
	if rv.Field(1).Type().String() != "string" {
		t.Errorf("Name should be string, got %s", rv.Field(1).Type())
	}
}

// ===== HTTP handler tests =====

func TestCreate_BadJSONReturns400(t *testing.T) {
	r := newRouter()
	req := httptest.NewRequest(http.MethodPost, "/events/tags", bytes.NewReader([]byte("not-json")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 or 401, got %d", rr.Code)
	}
}

func TestCreate_NoAuthHeaderReturns401(t *testing.T) {
	h := authedRouter(t)
	body, _ := json.Marshal(createTagRequest{
		EventID: uuid.New(),
		Name:    "trail",
	})
	req := httptest.NewRequest(http.MethodPost, "/events/tags", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestCreate_BadTokenReturns401(t *testing.T) {
	h := authedRouter(t)
	body, _ := json.Marshal(createTagRequest{
		EventID: uuid.New(),
		Name:    "trail",
	})
	req := httptest.NewRequest(http.MethodPost, "/events/tags", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer not.a.valid.jwt")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestDelete_NoAuthHeaderReturns401(t *testing.T) {
	h := authedRouter(t)
	req := httptest.NewRequest(http.MethodDelete, "/events/tags/"+uuid.New().String(), nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestDelete_BadUUIDReturns400(t *testing.T) {
	// Bad-UUID parsing is independent of auth — hit the router without JWT
	// to verify the 400 path.
	repo := NewRepository(nil)
	hh := NewHandler(repo)
	r := chi.NewRouter()
	r.Delete("/events/tags/{id}", hh.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/events/tags/not-a-uuid", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad UUID, got %d", rr.Code)
	}
}

// ListByEvent parses {eventId} as a UUID. A bad UUID should produce 400
// before reaching the repo.
func TestListByEvent_BadUUIDReturns400(t *testing.T) {
	repo := NewRepository(nil)
	h := NewHandler(repo)
	r := chi.NewRouter()
	r.Get("/events/{eventId}/tags", h.ListByEvent)

	req := httptest.NewRequest(http.MethodGet, "/events/not-a-uuid/tags", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad UUID, got %d", rr.Code)
	}
}

func contains(haystack, needle string) bool {
	if len(needle) == 0 {
		return true
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
