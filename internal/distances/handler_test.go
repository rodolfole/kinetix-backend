package distances

import (
	"bytes"
	"encoding/json"
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

// newRouter wires handler Routes onto a fresh chi router. The service is built
// with a nil repo so any test path that reaches the service would nil-deref —
// only paths that fail validation/auth before reaching the service are tested.
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

func validCreateDistance() CreateDistanceRequest {
	return CreateDistanceRequest{
		EventID:        uuid.New(),
		DistanceTypeID: uuid.New(),
		Capacity:       100,
		Surface:        "road",
	}
}

func TestCreateDistanceRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateDistanceRequest
		wantErr bool
	}{
		{name: "valid", req: validCreateDistance(), wantErr: false},
		{
			name:    "missing event id",
			req:     func() CreateDistanceRequest { r := validCreateDistance(); r.EventID = uuid.Nil; return r }(),
			wantErr: true,
		},
		{
			name:    "missing distance type id",
			req:     func() CreateDistanceRequest { r := validCreateDistance(); r.DistanceTypeID = uuid.Nil; return r }(),
			wantErr: true,
		},
		{
			name:    "zero capacity",
			req:     func() CreateDistanceRequest { r := validCreateDistance(); r.Capacity = 0; return r }(),
			wantErr: true,
		},
		{
			name:    "negative capacity",
			req:     func() CreateDistanceRequest { r := validCreateDistance(); r.Capacity = -5; return r }(),
			wantErr: true,
		},
		{
			name:    "missing surface",
			req:     func() CreateDistanceRequest { r := validCreateDistance(); r.Surface = ""; return r }(),
			wantErr: true,
		},
		{
			name:    "surface too long",
			req:     func() CreateDistanceRequest { r := validCreateDistance(); r.Surface = string(make([]byte, 51)); return r }(),
			wantErr: true, // validator cap is max=50, so 51 bytes exceeds it
		},
		{
			name: "surface exactly 50",
			req:  func() CreateDistanceRequest { r := validCreateDistance(); r.Surface = string(make([]byte, 50)); return r }(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.Struct(tt.req)
			if tt.wantErr && err == nil {
				t.Fatal("expected validation error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected validation error: %v", err)
			}
		})
	}
}

func TestUpdateDistanceRequest_Validate(t *testing.T) {
	valid := func() UpdateDistanceRequest {
		return UpdateDistanceRequest{
			DistanceTypeID: uuid.New(),
			Capacity:       50,
			Surface:        "trail",
		}
	}
	tests := []struct {
		name    string
		req     UpdateDistanceRequest
		wantErr bool
	}{
		{name: "valid", req: valid(), wantErr: false},
		{
			name:    "missing distance type id",
			req:     func() UpdateDistanceRequest { r := valid(); r.DistanceTypeID = uuid.Nil; return r }(),
			wantErr: true,
		},
		{
			name:    "zero capacity",
			req:     func() UpdateDistanceRequest { r := valid(); r.Capacity = 0; return r }(),
			wantErr: true,
		},
		{
			name:    "missing surface",
			req:     func() UpdateDistanceRequest { r := valid(); r.Surface = ""; return r }(),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.Struct(tt.req)
			if tt.wantErr && err == nil {
				t.Fatal("expected validation error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected validation error: %v", err)
			}
		})
	}
}

// ===== HTTP handler tests =====

func TestCreate_BadJSONReturns400(t *testing.T) {
	r := newRouter()
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte("not-json")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestGetByID_BadUUIDReturns400(t *testing.T) {
	r := newRouter()
	req := httptest.NewRequest(http.MethodGet, "/not-a-uuid", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad UUID, got %d", rr.Code)
	}
}

func TestUpdate_BadUUIDReturns400(t *testing.T) {
	r := newRouter()
	body, _ := json.Marshal(validUpdateDistance())
	req := httptest.NewRequest(http.MethodPut, "/not-a-uuid", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad UUID, got %d", rr.Code)
	}
}

func TestUpdate_BadJSONReturns400(t *testing.T) {
	r := newRouter()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPut, "/"+id, bytes.NewReader([]byte("{broken")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad JSON, got %d", rr.Code)
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

func TestListByEvent_BadUUIDReturns400(t *testing.T) {
	r := newRouter()
	req := httptest.NewRequest(http.MethodGet, "/event/not-a-uuid", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad UUID, got %d", rr.Code)
	}
}

// ===== Auth tests =====

func TestCreate_NoAuthHeaderReturns401(t *testing.T) {
	h := authedRouter(t)
	body, _ := json.Marshal(validCreateDistance())
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestCreate_BadTokenReturns401(t *testing.T) {
	h := authedRouter(t)
	body, _ := json.Marshal(validCreateDistance())
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
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

func TestUpdate_NoAuthHeaderReturns401(t *testing.T) {
	h := authedRouter(t)
	id := uuid.New().String()
	body, _ := json.Marshal(validUpdateDistance())
	req := httptest.NewRequest(http.MethodPut, "/"+id, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
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

func validUpdateDistance() UpdateDistanceRequest {
	return UpdateDistanceRequest{
		DistanceTypeID: uuid.New(),
		Capacity:       50,
		Surface:        "trail",
	}
}
