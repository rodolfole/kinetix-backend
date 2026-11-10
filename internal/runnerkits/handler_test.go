package runnerkits

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"kinetix-api/internal/auth"
	"kinetix-api/internal/middleware"
)

var validate = validator.New()

// newRouter wires handler Routes onto a fresh chi router. Service has nil
// repo/store — only validation/auth paths that fail before reaching the
// service are exercised here.
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

func validCreateKitItem() CreateRunnerKitItemRequest {
	return CreateRunnerKitItemRequest{
		EventID: uuid.New(),
		Name:    "Bib",
		Icon:    "shirt",
	}
}

func TestCreateKitItemRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateRunnerKitItemRequest
		wantErr bool
	}{
		{name: "valid", req: validCreateKitItem(), wantErr: false},
		{name: "valid no icon", req: CreateRunnerKitItemRequest{EventID: uuid.New(), Name: "Medal"}, wantErr: false},
		{
			name:    "missing event id",
			req:     func() CreateRunnerKitItemRequest { r := validCreateKitItem(); r.EventID = uuid.Nil; return r }(),
			wantErr: true,
		},
		{
			name:    "missing name",
			req:     func() CreateRunnerKitItemRequest { r := validCreateKitItem(); r.Name = ""; return r }(),
			wantErr: true,
		},
		{
			name: "name too long",
			req: func() CreateRunnerKitItemRequest {
				r := validCreateKitItem()
				r.Name = strings.Repeat("a", 101)
				return r
			}(),
			wantErr: true,
		},
		{
			name: "name exactly 100",
			req: func() CreateRunnerKitItemRequest {
				r := validCreateKitItem()
				r.Name = strings.Repeat("a", 100)
				return r
			}(),
			wantErr: false,
		},
		{
			name: "icon too long",
			req: func() CreateRunnerKitItemRequest {
				r := validCreateKitItem()
				r.Icon = strings.Repeat("x", 51)
				return r
			}(),
			wantErr: true,
		},
		{
			name: "icon exactly 50",
			req: func() CreateRunnerKitItemRequest {
				r := validCreateKitItem()
				r.Icon = strings.Repeat("x", 50)
				return r
			}(),
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

func validUpdateKitItem() UpdateRunnerKitItemRequest {
	return UpdateRunnerKitItemRequest{
		Name: "Updated Bib",
		Icon: "tshirt",
	}
}

func TestUpdateKitItemRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     UpdateRunnerKitItemRequest
		wantErr bool
	}{
		{name: "valid", req: validUpdateKitItem(), wantErr: false},
		{
			name:    "missing name",
			req:     UpdateRunnerKitItemRequest{Name: ""},
			wantErr: true,
		},
		{
			name:    "name too long",
			req:     UpdateRunnerKitItemRequest{Name: strings.Repeat("a", 101)},
			wantErr: true,
		},
		{
			name:    "icon too long",
			req:     UpdateRunnerKitItemRequest{Name: "ok", Icon: strings.Repeat("x", 51)},
			wantErr: true,
		},
		{
			name:    "empty icon allowed (omitempty)",
			req:     UpdateRunnerKitItemRequest{Name: "ok", Icon: ""},
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

func TestListByEvent_BadUUIDReturns400(t *testing.T) {
	r := newRouter()
	req := httptest.NewRequest(http.MethodGet, "/event/not-a-uuid", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad UUID, got %d", rr.Code)
	}
}

func TestUpdate_BadUUIDReturns400(t *testing.T) {
	r := newRouter()
	body, _ := json.Marshal(validUpdateKitItem())
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

// ===== Auth tests =====

func TestCreate_NoAuthHeaderReturns401(t *testing.T) {
	h := authedRouter(t)
	body, _ := json.Marshal(validCreateKitItem())
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
	body, _ := json.Marshal(validCreateKitItem())
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
	body, _ := json.Marshal(validUpdateKitItem())
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
