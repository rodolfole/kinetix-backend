package participants

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	sqlcdb "kinetix-api/internal/adapters/postgresql/sqlc"
	"kinetix-api/internal/auth"
	kinetix_middleware "kinetix-api/internal/middleware"
	"kinetix-api/internal/store"
)

// Shared validator instance mirroring the patterns in auth/handler_test.go
// and orders/handler_test.go.
var validate = validator.New()

// newRouter wires the participant Routes onto a fresh chi router. The service
// is constructed with a nil repo so any test that reaches service.* would
// nil-deref; only test paths that fail before reaching the service are covered.
func newRouter() http.Handler {
	svc := NewService(nil, validate)
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
	return kinetix_middleware.JWTAuth(cfg)(newRouter())
}

// notFoundRouter mirrors newRouter but uses a Repository whose Store is wired
// to a fake DBTX that always errors out. This lets us exercise the
// "service returned an error" -> 404 path the handler takes on a missing
// participant, without requiring a live database.
//
// The fake Store is built via reflection on the *store.Store struct so the
// production code is left untouched. If the Store layout ever changes (field
// rename/type change) the test skips gracefully instead of panicking.
func notFoundRouter(t *testing.T) http.Handler {
	t.Helper()

	fakeQ := sqlcdb.New(&fakeDBTX{err: errors.New("participant not found")})
	s := &store.Store{}
	v := reflect.ValueOf(s).Elem()
	fld := v.FieldByName("Queries")
	if !fld.IsValid() || !fld.CanSet() {
		t.Skip("internal/store layout changed; cannot inject fake Queries for not-found test")
	}
	fld.Set(reflect.ValueOf(fakeQ))

	repo := &Repository{store: s}
	svc := NewService(repo, validate)
	h := NewHandler(svc)
	r := chi.NewRouter()
	h.Routes(r)
	return r
}

// fakeDBTX is an sqlc.DBTX whose QueryRow returns a Row whose Scan fails,
// simulating the underlying query missing the requested participant.
type fakeDBTX struct {
	err error
}

func (f *fakeDBTX) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, errors.New("exec not supported in fakeDBTX")
}
func (f *fakeDBTX) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return nil, errors.New("query not supported in fakeDBTX")
}
func (f *fakeDBTX) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return &fakeRow{err: f.err}
}

type fakeRow struct{ err error }

func (r *fakeRow) Scan(dest ...any) error { return r.err }

// validCreateParticipant returns a CreateParticipantRequest that passes validation.
func validCreateParticipant() CreateParticipantRequest {
	return CreateParticipantRequest{
		FirstName: "John",
		LastName:  "Doe",
		Gender:    "male",
		BirthDate: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
		Email:     "user@example.com",
		Phone:     "+5255555555",
		Country:   "MX",
	}
}

// validUpdateParticipant returns an UpdateParticipantRequest that passes validation.
// The struct has identical fields to CreateParticipantRequest but is a distinct
// type, so a fresh factory is needed.
func validUpdateParticipant() UpdateParticipantRequest {
	return UpdateParticipantRequest{
		FirstName: "John",
		LastName:  "Doe",
		Gender:    "male",
		BirthDate: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
		Email:     "user@example.com",
		Phone:     "+5255555555",
		Country:   "MX",
	}
}

// strPtr returns a pointer to the given string; used for optional fields
// like Size that take *string.
func strPtr(s string) *string { return &s }

// ===== DTO validator tests =====

func TestCreateParticipantRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateParticipantRequest
		wantErr bool
	}{
		{
			name:    "valid",
			req:     validCreateParticipant(),
			wantErr: false,
		},
		{
			name:    "missing email",
			req:     func() CreateParticipantRequest { r := validCreateParticipant(); r.Email = ""; return r }(),
			wantErr: true,
		},
		{
			name:    "bad email",
			req:     func() CreateParticipantRequest { r := validCreateParticipant(); r.Email = "no-at-sign"; return r }(),
			wantErr: true,
		},
		{
			name:    "missing phone",
			req:     func() CreateParticipantRequest { r := validCreateParticipant(); r.Phone = ""; return r }(),
			wantErr: true,
		},
		{
			name:    "phone too short",
			req:     func() CreateParticipantRequest { r := validCreateParticipant(); r.Phone = "12345"; return r }(),
			wantErr: true,
		},
		{
			name:    "missing first name",
			req:     func() CreateParticipantRequest { r := validCreateParticipant(); r.FirstName = ""; return r }(),
			wantErr: true,
		},
		{
			name:    "missing last name",
			req:     func() CreateParticipantRequest { r := validCreateParticipant(); r.LastName = ""; return r }(),
			wantErr: true,
		},
		{
			name:    "missing gender",
			req:     func() CreateParticipantRequest { r := validCreateParticipant(); r.Gender = ""; return r }(),
			wantErr: true,
		},
		{
			name:    "missing country",
			req:     func() CreateParticipantRequest { r := validCreateParticipant(); r.Country = ""; return r }(),
			wantErr: true,
		},
		{
			name:    "country wrong length",
			req:     func() CreateParticipantRequest { r := validCreateParticipant(); r.Country = "MEX"; return r }(),
			wantErr: true,
		},
		{
			name:    "bad size",
			req:     func() CreateParticipantRequest { r := validCreateParticipant(); r.Size = strPtr("XXXL"); return r }(),
			wantErr: true,
		},
		{
			name:    "good size M",
			req:     func() CreateParticipantRequest { r := validCreateParticipant(); r.Size = strPtr("M"); return r }(),
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

func TestUpdateParticipantRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     UpdateParticipantRequest
		wantErr bool
	}{
		{
			name:    "valid",
			req:     validUpdateParticipant(),
			wantErr: false,
		},
		{
			name:    "missing email",
			req:     func() UpdateParticipantRequest { r := validUpdateParticipant(); r.Email = ""; return r }(),
			wantErr: true,
		},
		{
			name:    "bad email",
			req:     func() UpdateParticipantRequest { r := validUpdateParticipant(); r.Email = "not-an-email"; return r }(),
			wantErr: true,
		},
		{
			name:    "missing phone",
			req:     func() UpdateParticipantRequest { r := validUpdateParticipant(); r.Phone = ""; return r }(),
			wantErr: true,
		},
		{
			name:    "phone too short",
			req:     func() UpdateParticipantRequest { r := validUpdateParticipant(); r.Phone = "123"; return r }(),
			wantErr: true,
		},
		{
			name:    "missing first name",
			req:     func() UpdateParticipantRequest { r := validUpdateParticipant(); r.FirstName = ""; return r }(),
			wantErr: true,
		},
		{
			name:    "missing last name",
			req:     func() UpdateParticipantRequest { r := validUpdateParticipant(); r.LastName = ""; return r }(),
			wantErr: true,
		},
		{
			name:    "missing gender",
			req:     func() UpdateParticipantRequest { r := validUpdateParticipant(); r.Gender = ""; return r }(),
			wantErr: true,
		},
		{
			name:    "bad size",
			req:     func() UpdateParticipantRequest { r := validUpdateParticipant(); r.Size = strPtr("XXXL"); return r }(),
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

// ===== HTTP handler tests (paths that fail before reaching the service) =====

func TestCreate_BadJSONReturns400(t *testing.T) {
	r := newRouter()
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte("not-json")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad JSON, got %d", rr.Code)
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
	body, _ := json.Marshal(validCreateParticipant())
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

// ===== Not-found handler test (handler reaches the service) =====

func TestGetByID_ParticipantNotFound_Returns404(t *testing.T) {
	r := notFoundRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/"+uuid.New().String(), nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing participant, got %d", rr.Code)
	}
}

// ===== Auth tests =====

func TestCreate_NoAuthHeaderReturns401(t *testing.T) {
	h := authedRouter(t)
	body, _ := json.Marshal(validCreateParticipant())
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 with no auth, got %d", rr.Code)
	}
}

func TestCreate_BadTokenReturns401(t *testing.T) {
	h := authedRouter(t)
	body, _ := json.Marshal(validCreateParticipant())
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer not.a.valid.jwt")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 with bad token, got %d", rr.Code)
	}
}

func TestList_NoAuthHeaderReturns401(t *testing.T) {
	h := authedRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 with no auth, got %d", rr.Code)
	}
}

func TestGetByID_NoAuthHeaderReturns401(t *testing.T) {
	h := authedRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/"+uuid.New().String(), nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 with no auth, got %d", rr.Code)
	}
}

func TestUpdate_NoAuthHeaderReturns401(t *testing.T) {
	h := authedRouter(t)
	body, _ := json.Marshal(validCreateParticipant())
	req := httptest.NewRequest(http.MethodPut, "/"+uuid.New().String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 with no auth, got %d", rr.Code)
	}
}

func TestDelete_NoAuthHeaderReturns401(t *testing.T) {
	h := authedRouter(t)
	req := httptest.NewRequest(http.MethodDelete, "/"+uuid.New().String(), nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 with no auth, got %d", rr.Code)
	}
}
