package organizers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"kinetix-api/internal/auth"
	"kinetix-api/internal/middleware"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// validate is a package-level validator matching the existing test style.
var validate = validator.New()

func validateStruct(v any) error {
	return validate.Struct(v)
}

func testJWTConfig() auth.Config {
	return auth.Config{
		SecretKey:   "test-secret-key",
		TokenExpiry: time.Hour,
	}
}

// validCreateOrganizer returns a request that passes DTO validation.
func validCreateOrganizer() CreateOrganizerRequest {
	return CreateOrganizerRequest{
		BusinessName:   "Acme Corp",
		BrandName:      "Acme",
		RFC:            "ABC010101ABC",
		BillingZipCode: "44100",
		BillingState:   "Jalisco",
		BillingCity:    "Guadalajara",
		ContactName:    "John Doe",
		ContactEmail:   "john@example.com",
		ContactPhone:   "+525555555555",
		LogoUrl:        "https://example.com/logo.png",
	}
}

func TestCreateOrganizerRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(r *CreateOrganizerRequest)
		wantErr bool
	}{
		{name: "valid", mutate: func(r *CreateOrganizerRequest) {}, wantErr: false},
		{name: "missing business name", mutate: func(r *CreateOrganizerRequest) { r.BusinessName = "" }, wantErr: true},
		{name: "business name too short", mutate: func(r *CreateOrganizerRequest) { r.BusinessName = "a" }, wantErr: true},
		{name: "missing RFC", mutate: func(r *CreateOrganizerRequest) { r.RFC = "" }, wantErr: true},
		{name: "RFC length 12 valid", mutate: func(r *CreateOrganizerRequest) { r.RFC = "ABC010101AB1" }, wantErr: false},
		{name: "RFC length 13 valid", mutate: func(r *CreateOrganizerRequest) { r.RFC = "ABC010101ABC" }, wantErr: false},
		{name: "RFC bad length", mutate: func(r *CreateOrganizerRequest) { r.RFC = "ABC" }, wantErr: true},
		{name: "missing zip", mutate: func(r *CreateOrganizerRequest) { r.BillingZipCode = "" }, wantErr: true},
		{name: "missing state", mutate: func(r *CreateOrganizerRequest) { r.BillingState = "" }, wantErr: true},
		{name: "missing city", mutate: func(r *CreateOrganizerRequest) { r.BillingCity = "" }, wantErr: true},
		{name: "missing contact name", mutate: func(r *CreateOrganizerRequest) { r.ContactName = "" }, wantErr: true},
		{name: "missing contact email", mutate: func(r *CreateOrganizerRequest) { r.ContactEmail = "" }, wantErr: true},
		{name: "bad contact email", mutate: func(r *CreateOrganizerRequest) { r.ContactEmail = "not-an-email" }, wantErr: true},
		{name: "missing phone", mutate: func(r *CreateOrganizerRequest) { r.ContactPhone = "" }, wantErr: true},
		{name: "phone too short", mutate: func(r *CreateOrganizerRequest) { r.ContactPhone = "123" }, wantErr: true},
		{name: "phone too long", mutate: func(r *CreateOrganizerRequest) {
			r.ContactPhone = strings.Repeat("1", 21)
		}, wantErr: true},
		{name: "bad logo url", mutate: func(r *CreateOrganizerRequest) { r.LogoUrl = "not a url" }, wantErr: true},
		{name: "empty logo url ok (omitempty)", mutate: func(r *CreateOrganizerRequest) { r.LogoUrl = "" }, wantErr: false},
		{name: "empty brand name ok (omitempty)", mutate: func(r *CreateOrganizerRequest) { r.BrandName = "" }, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validCreateOrganizer()
			tt.mutate(&req)
			err := validateStruct(req)
			if tt.wantErr && err == nil {
				t.Fatal("expected validation error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestUpdateOrganizerRequest_Validate(t *testing.T) {
	base := UpdateOrganizerRequest{
		BusinessName:   "Acme Corp",
		BrandName:      "Acme",
		BillingZipCode: "44100",
		BillingState:   "Jalisco",
		BillingCity:    "Guadalajara",
		ContactName:    "Jane Doe",
		ContactEmail:   "jane@example.com",
		ContactPhone:   "+525555555555",
		LogoUrl:        "https://example.com/logo.png",
	}

	tests := []struct {
		name    string
		mutate  func(r *UpdateOrganizerRequest)
		wantErr bool
	}{
		{name: "valid", mutate: func(r *UpdateOrganizerRequest) {}, wantErr: false},
		{name: "missing business name", mutate: func(r *UpdateOrganizerRequest) { r.BusinessName = "" }, wantErr: true},
		{name: "missing zip", mutate: func(r *UpdateOrganizerRequest) { r.BillingZipCode = "" }, wantErr: true},
		{name: "missing state", mutate: func(r *UpdateOrganizerRequest) { r.BillingState = "" }, wantErr: true},
		{name: "missing city", mutate: func(r *UpdateOrganizerRequest) { r.BillingCity = "" }, wantErr: true},
		{name: "missing contact name", mutate: func(r *UpdateOrganizerRequest) { r.ContactName = "" }, wantErr: true},
		{name: "missing contact email", mutate: func(r *UpdateOrganizerRequest) { r.ContactEmail = "" }, wantErr: true},
		{name: "bad contact email", mutate: func(r *UpdateOrganizerRequest) { r.ContactEmail = "no-at-sign" }, wantErr: true},
		{name: "missing phone", mutate: func(r *UpdateOrganizerRequest) { r.ContactPhone = "" }, wantErr: true},
		{name: "phone too short", mutate: func(r *UpdateOrganizerRequest) { r.ContactPhone = "12345" }, wantErr: true},
		{name: "bad logo url", mutate: func(r *UpdateOrganizerRequest) { r.LogoUrl = "not a url" }, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := base
			tt.mutate(&req)
			err := validateStruct(req)
			if tt.wantErr && err == nil {
				t.Fatal("expected validation error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// newHandlerWithNilService returns a Handler that has no backing service.
// Routes that early-return (UUID parsing, body validation) can still be
// exercised safely; routes that would call the service must not be invoked.
func newHandlerWithNilService() *Handler {
	return NewHandler(nil, validate)
}

// ============ HTTP behavior tests ============

// newRouterWithJWT mounts a single route behind JWTAuth middleware.
func newRouterWithJWT(handler http.HandlerFunc, pattern string) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.JWTAuth(testJWTConfig()))
	r.Method("GET", pattern, handler)
	return r
}

func TestHandler_Auth_NoToken_Returns401(t *testing.T) {
	h := newHandlerWithNilService()
	r := newRouterWithJWT(h.GetOrganizer, "/{id}")

	req := httptest.NewRequest(http.MethodGet, "/"+uuid.New().String(), nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", rr.Code)
	}
}

func TestHandler_Auth_BadHeaderFormat_Returns401(t *testing.T) {
	h := newHandlerWithNilService()
	r := newRouterWithJWT(h.GetOrganizer, "/{id}")

	req := httptest.NewRequest(http.MethodGet, "/"+uuid.New().String(), nil)
	req.Header.Set("Authorization", "Token abc.def.ghi")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for non-Bearer header, got %d", rr.Code)
	}
}

func TestHandler_Auth_InvalidToken_Returns401(t *testing.T) {
	h := newHandlerWithNilService()
	r := newRouterWithJWT(h.GetOrganizer, "/{id}")

	req := httptest.NewRequest(http.MethodGet, "/"+uuid.New().String(), nil)
	req.Header.Set("Authorization", "Bearer not.a.real.jwt")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for invalid token, got %d", rr.Code)
	}
}

// ============ Invalid UUID / bad body tests ============

func TestHandler_GetOrganizer_InvalidUUID_Returns400(t *testing.T) {
	h := newHandlerWithNilService()

	r := chi.NewRouter()
	r.Get("/{id}", h.GetOrganizer)

	req := httptest.NewRequest(http.MethodGet, "/not-a-uuid", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid UUID, got %d", rr.Code)
	}
}

func TestHandler_UpdateOrganizer_InvalidUUID_Returns400(t *testing.T) {
	h := newHandlerWithNilService()

	r := chi.NewRouter()
	r.Put("/{id}", h.UpdateOrganizer)

	body := strings.NewReader(`{"business_name":"Acme"}`)
	req := httptest.NewRequest(http.MethodPut, "/not-a-uuid", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid UUID, got %d", rr.Code)
	}
}

func TestHandler_DeleteOrganizer_InvalidUUID_Returns400(t *testing.T) {
	h := newHandlerWithNilService()

	r := chi.NewRouter()
	r.Delete("/{id}", h.DeleteOrganizer)

	req := httptest.NewRequest(http.MethodDelete, "/bad", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid UUID, got %d", rr.Code)
	}
}

func TestHandler_CreateOrganizer_InvalidJSON_Returns400(t *testing.T) {
	h := newHandlerWithNilService()

	r := chi.NewRouter()
	r.Post("/", h.CreateOrganizer)

	body := strings.NewReader(`{not valid json`)
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d", rr.Code)
	}
}
