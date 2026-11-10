package registrations

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

// newRouter wires the handler Routes onto a fresh chi router. The service is
// constructed with nil repo/store so tests that reach service methods would
// nil-deref; only test paths that fail before reaching the service.
func newRouter() http.Handler {
	svc := NewService(nil, nil, validate)
	h := NewHandler(svc)
	r := chi.NewRouter()
	h.Routes(r)
	return r
}

// authedRouter wraps the routes with JWT auth middleware so we can verify
// auth-required behaviour without exercising the service.
func authedRouter(t *testing.T) http.Handler {
	t.Helper()
	cfg := auth.Config{SecretKey: "test-secret", TokenExpiry: time.Hour}
	return middleware.JWTAuth(cfg)(newRouter())
}

func validCreateRegistration() CreateRegistrationRequest {
	return CreateRegistrationRequest{
		EventID:       uuid.New(),
		ParticipantID: uuid.New(),
		DistanceID:    uuid.New(),
		Category:      "10K",
	}
}

func TestCreateRegistrationRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateRegistrationRequest
		wantErr bool
	}{
		{name: "valid", req: validCreateRegistration(), wantErr: false},
		{
			name:    "missing event id",
			req:     func() CreateRegistrationRequest { r := validCreateRegistration(); r.EventID = uuid.Nil; return r }(),
			wantErr: true,
		},
		{
			name:    "missing participant id",
			req:     func() CreateRegistrationRequest { r := validCreateRegistration(); r.ParticipantID = uuid.Nil; return r }(),
			wantErr: true,
		},
		{
			name:    "missing distance id",
			req:     func() CreateRegistrationRequest { r := validCreateRegistration(); r.DistanceID = uuid.Nil; return r }(),
			wantErr: true,
		},
		{
			name:    "missing category",
			req:     func() CreateRegistrationRequest { r := validCreateRegistration(); r.Category = ""; return r }(),
			wantErr: true,
		},
		{
			name: "category too long",
			req: func() CreateRegistrationRequest {
				r := validCreateRegistration()
				r.Category = strings.Repeat("a", 21)
				return r
			}(),
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
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestCreateParticipantData_Validate(t *testing.T) {
	birth := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		req     CreateParticipantData
		wantErr bool
	}{
		{
			name: "valid",
			req: CreateParticipantData{
				FirstName: "John", LastName: "Doe", Gender: "male",
				BirthDate: birth, Email: "user@example.com",
				Phone: "+5255555555", Country: "MX",
				Size: strPtr("M"), Address: "123 Main St",
			},
			wantErr: false,
		},
		{name: "missing first_name", req: func() CreateParticipantData {
			r := validParticipant(); r.FirstName = ""; return r
		}(), wantErr: true},
		{name: "missing last_name", req: func() CreateParticipantData {
			r := validParticipant(); r.LastName = ""; return r
		}(), wantErr: true},
		{name: "missing gender", req: func() CreateParticipantData {
			r := validParticipant(); r.Gender = ""; return r
		}(), wantErr: true},
		{name: "missing email", req: func() CreateParticipantData {
			r := validParticipant(); r.Email = ""; return r
		}(), wantErr: true},
		{name: "bad email", req: func() CreateParticipantData {
			r := validParticipant(); r.Email = "no-at-sign"; return r
		}(), wantErr: true},
		{name: "missing phone", req: func() CreateParticipantData {
			r := validParticipant(); r.Phone = ""; return r
		}(), wantErr: true},
		{name: "phone too short", req: func() CreateParticipantData {
			r := validParticipant(); r.Phone = "12345"; return r
		}(), wantErr: true},
		{name: "missing country", req: func() CreateParticipantData {
			r := validParticipant(); r.Country = ""; return r
		}(), wantErr: true},
		{name: "country wrong length", req: func() CreateParticipantData {
			r := validParticipant(); r.Country = "MEX"; return r
		}(), wantErr: true},
		{name: "missing address", req: func() CreateParticipantData {
			r := validParticipant(); r.Address = ""; return r
		}(), wantErr: true},
		{name: "bad size", req: func() CreateParticipantData {
			r := validParticipant(); r.Size = strPtr("XXXL"); return r
		}(), wantErr: true},
		{name: "good size XL", req: func() CreateParticipantData {
			r := validParticipant(); r.Size = strPtr("XL"); return r
		}(), wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.Struct(tt.req)
			if tt.wantErr && err == nil {
				t.Fatal("expected validation error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestCreateWithParticipantRegistration_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateWithParticipantRegistration
		wantErr bool
	}{
		{
			name: "valid",
			req: CreateWithParticipantRegistration{
				DistanceID: uuid.New(),
				Category:   "10K",
				Time:       strPtr("00:30:00"),
				Team:       strPtr("Runners"),
				Size:       strPtr("M"),
			},
			wantErr: false,
		},
		{
			name:    "missing distance id",
			req:     CreateWithParticipantRegistration{Category: "10K"},
			wantErr: true,
		},
		{
			name:    "missing category",
			req:     CreateWithParticipantRegistration{DistanceID: uuid.New()},
			wantErr: true,
		},
		{
			name: "category too long",
			req: CreateWithParticipantRegistration{
				DistanceID: uuid.New(),
				Category:   strings.Repeat("a", 21),
			},
			wantErr: true,
		},
		{
			name: "bad size",
			req: CreateWithParticipantRegistration{
				DistanceID: uuid.New(),
				Category:   "10K",
				Size:       strPtr("XXXL"),
			},
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
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestCreateWithParticipantRequest_Validate(t *testing.T) {
	birth := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		req     CreateWithParticipantRequest
		wantErr bool
	}{
		{
			name: "valid with new participant",
			req: CreateWithParticipantRequest{
				EventID: uuid.New(),
				Participant: validParticipant(),
				Registration: CreateWithParticipantRegistration{
					DistanceID: uuid.New(),
					Category:   "10K",
				},
			},
			wantErr: false,
		},
		{
			name: "missing event id",
			req: CreateWithParticipantRequest{
				Participant: validParticipant(),
				Registration: CreateWithParticipantRegistration{
					DistanceID: uuid.New(),
					Category:   "10K",
				},
			},
			wantErr: true,
		},
		{
			name: "missing registration",
			req: CreateWithParticipantRequest{
				EventID:     uuid.New(),
				Participant: validParticipant(),
			},
			wantErr: true,
		},
		{
			name: "registration missing fields",
			req: CreateWithParticipantRequest{
				EventID: uuid.New(),
				Participant: CreateParticipantData{
					FirstName: "John", LastName: "Doe", Gender: "male",
					BirthDate: birth, Email: "user@example.com",
					Phone: "+5255555555", Country: "MX",
					Size: strPtr("M"), Address: "123 Main St",
				},
				// Registration struct exists but lacks required fields
			},
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
				t.Fatalf("unexpected error: %v", err)
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

func TestCreateWithParticipant_BadJSONReturns400(t *testing.T) {
	r := newRouter()
	req := httptest.NewRequest(http.MethodPost, "/with-participant", bytes.NewReader([]byte("{broken")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestListByEvent_MissingEventIDReturns400(t *testing.T) {
	r := newRouter()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 when event_id missing, got %d", rr.Code)
	}
}

func TestListByEvent_BadUUIDReturns400(t *testing.T) {
	r := newRouter()
	req := httptest.NewRequest(http.MethodGet, "/?event_id=not-a-uuid", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad UUID, got %d", rr.Code)
	}
}

func TestListByParticipant_BadUUIDReturns400(t *testing.T) {
	r := newRouter()
	req := httptest.NewRequest(http.MethodGet, "/participant/not-a-uuid", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad UUID, got %d", rr.Code)
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
	body, _ := json.Marshal(validCreateRegistration())
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
	body, _ := json.Marshal(validCreateRegistration())
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

func TestDelete_NoAuthHeaderReturns401(t *testing.T) {
	h := authedRouter(t)
	req := httptest.NewRequest(http.MethodDelete, "/"+uuid.New().String(), nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

// ===== helpers =====

func strPtr(s string) *string { return &s }

func validParticipant() CreateParticipantData {
	birth := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	return CreateParticipantData{
		FirstName: "John",
		LastName:  "Doe",
		Gender:    "male",
		BirthDate: birth,
		Email:     "user@example.com",
		Phone:     "+5255555555",
		Country:   "MX",
		Size:      strPtr("M"),
		Address:   "123 Main St",
	}
}