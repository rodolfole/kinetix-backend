package events

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

// testJWTConfig returns a fixed JWT config for middleware tests.
func testJWTConfig() auth.Config {
	return auth.Config{
		SecretKey:   "test-secret-key",
		TokenExpiry: time.Hour,
	}
}

// validCreateEvent returns a request that passes DTO validation.
func validCreateEvent() CreateEventRequest {
	return CreateEventRequest{
		OrganizerID:  uuid.New(),
		Name:         "Marathon 2026",
		Slug:         "marathon-2026",
		Description:  "Annual marathon",
		EventDate:    time.Date(2026, 6, 1, 8, 0, 0, 0, time.UTC),
		Deadline:     time.Date(2026, 5, 1, 23, 59, 59, 0, time.UTC),
		State:        "Jalisco",
		Municipality: "Guadalajara",
		Address:      "123 Main Street, City",
		LogoURL:      "https://example.com/logo.png",
		Itinerary: []ItineraryItem{
			{Key: "start", Time: "08:00", Icon: "flag"},
		},
	}
}

func TestCreateEventRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(r *CreateEventRequest)
		wantErr bool
	}{
		{name: "valid", mutate: func(r *CreateEventRequest) {}, wantErr: false},
		{name: "missing organizer id", mutate: func(r *CreateEventRequest) { r.OrganizerID = uuid.Nil }, wantErr: true},
		{name: "missing name", mutate: func(r *CreateEventRequest) { r.Name = "" }, wantErr: true},
		{name: "name too short", mutate: func(r *CreateEventRequest) { r.Name = "a" }, wantErr: true},
		{name: "missing slug", mutate: func(r *CreateEventRequest) { r.Slug = "" }, wantErr: true},
		{name: "missing address", mutate: func(r *CreateEventRequest) { r.Address = "" }, wantErr: true},
		{name: "address too short", mutate: func(r *CreateEventRequest) { r.Address = "abc" }, wantErr: true},
		{name: "missing event_date", mutate: func(r *CreateEventRequest) { r.EventDate = time.Time{} }, wantErr: true},
		{name: "missing deadline", mutate: func(r *CreateEventRequest) { r.Deadline = time.Time{} }, wantErr: true},
		{name: "missing itinerary", mutate: func(r *CreateEventRequest) { r.Itinerary = nil }, wantErr: true},
		{name: "empty itinerary", mutate: func(r *CreateEventRequest) { r.Itinerary = []ItineraryItem{} }, wantErr: true},
		{name: "bad logo url", mutate: func(r *CreateEventRequest) { r.LogoURL = "not-a-url" }, wantErr: true},
		{name: "valid logo url", mutate: func(r *CreateEventRequest) { r.LogoURL = "https://cdn.example.com/x.png" }, wantErr: false},
		{name: "itinerary item missing key", mutate: func(r *CreateEventRequest) {
			r.Itinerary = []ItineraryItem{{Key: "", Time: "08:00", Icon: "flag"}}
		}, wantErr: true},
		{name: "itinerary item missing time", mutate: func(r *CreateEventRequest) {
			r.Itinerary = []ItineraryItem{{Key: "start", Time: "", Icon: "flag"}}
		}, wantErr: true},
		{name: "itinerary item missing icon", mutate: func(r *CreateEventRequest) {
			r.Itinerary = []ItineraryItem{{Key: "start", Time: "08:00", Icon: ""}}
		}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validCreateEvent()
			tt.mutate(&req)
			err := validateStruct(req)
			if tt.wantErr && err == nil {
				t.Fatal("expected validation error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

func TestUpdateEventRequest_Validate(t *testing.T) {
	base := UpdateEventRequest{
		Name:         "Marathon 2026",
		Slug:         "marathon-2026",
		EventDate:    time.Date(2026, 6, 1, 8, 0, 0, 0, time.UTC),
		Deadline:     time.Date(2026, 5, 1, 23, 59, 59, 0, time.UTC),
		Address:      "123 Main Street, City",
		Itinerary:    []ItineraryItem{{Key: "start", Time: "08:00", Icon: "flag"}},
		Status:       "draft",
	}

	tests := []struct {
		name    string
		mutate  func(r *UpdateEventRequest)
		wantErr bool
	}{
		{name: "valid draft", mutate: func(r *UpdateEventRequest) {}, wantErr: false},
		{name: "valid empty status (omitempty)", mutate: func(r *UpdateEventRequest) { r.Status = "" }, wantErr: false},
		{name: "status published", mutate: func(r *UpdateEventRequest) { r.Status = "published" }, wantErr: false},
		{name: "status cancelled", mutate: func(r *UpdateEventRequest) { r.Status = "cancelled" }, wantErr: false},
		{name: "status completed", mutate: func(r *UpdateEventRequest) { r.Status = "completed" }, wantErr: false},
		{name: "status invalid", mutate: func(r *UpdateEventRequest) { r.Status = "archived" }, wantErr: true},
		{name: "missing name", mutate: func(r *UpdateEventRequest) { r.Name = "" }, wantErr: true},
		{name: "missing slug", mutate: func(r *UpdateEventRequest) { r.Slug = "" }, wantErr: true},
		{name: "missing address", mutate: func(r *UpdateEventRequest) { r.Address = "" }, wantErr: true},
		{name: "missing itinerary", mutate: func(r *UpdateEventRequest) { r.Itinerary = nil }, wantErr: true},
		{name: "bad logo url", mutate: func(r *UpdateEventRequest) { r.LogoURL = "not a url" }, wantErr: true},
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
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

func TestCreateAwardRequest_Validate(t *testing.T) {
	base := CreateAwardRequest{
		DistanceID:    uuid.New(),
		AgeCategoryID: uuid.New(),
		Gender:        "male",
		Place:         1,
		PrizeAmount:   1000,
	}

	tests := []struct {
		name    string
		mutate  func(r *CreateAwardRequest)
		wantErr bool
	}{
		{name: "valid male", mutate: func(r *CreateAwardRequest) {}, wantErr: false},
		{name: "valid female", mutate: func(r *CreateAwardRequest) { r.Gender = "female" }, wantErr: false},
		{name: "valid general", mutate: func(r *CreateAwardRequest) { r.Gender = "general" }, wantErr: false},
		{name: "invalid gender", mutate: func(r *CreateAwardRequest) { r.Gender = "other" }, wantErr: true},
		{name: "missing distance", mutate: func(r *CreateAwardRequest) { r.DistanceID = uuid.Nil }, wantErr: true},
		{name: "missing age category", mutate: func(r *CreateAwardRequest) { r.AgeCategoryID = uuid.Nil }, wantErr: true},
		{name: "missing gender", mutate: func(r *CreateAwardRequest) { r.Gender = "" }, wantErr: true},
		{name: "negative prize", mutate: func(r *CreateAwardRequest) { r.PrizeAmount = -1 }, wantErr: true},
		{name: "place too low", mutate: func(r *CreateAwardRequest) { r.Place = -1 }, wantErr: true},
		{name: "place too high", mutate: func(r *CreateAwardRequest) { r.Place = 21 }, wantErr: true},
		{name: "place at top", mutate: func(r *CreateAwardRequest) { r.Place = 20 }, wantErr: false},
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
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

func TestUpdateAwardRequest_Validate(t *testing.T) {
	base := UpdateAwardRequest{
		AgeCategoryID: uuid.New(),
		Gender:        "female",
		Place:         2,
		PrizeAmount:   500,
	}

	tests := []struct {
		name    string
		mutate  func(r *UpdateAwardRequest)
		wantErr bool
	}{
		{name: "valid", mutate: func(r *UpdateAwardRequest) {}, wantErr: false},
		{name: "invalid gender", mutate: func(r *UpdateAwardRequest) { r.Gender = "x" }, wantErr: true},
		{name: "missing age category", mutate: func(r *UpdateAwardRequest) { r.AgeCategoryID = uuid.Nil }, wantErr: true},
		{name: "negative prize", mutate: func(r *UpdateAwardRequest) { r.PrizeAmount = -10 }, wantErr: true},
		{name: "place -1 invalid", mutate: func(r *UpdateAwardRequest) { r.Place = -1 }, wantErr: true},
		{name: "place 21 invalid", mutate: func(r *UpdateAwardRequest) { r.Place = 21 }, wantErr: true},
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
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

func TestCreatePricingStageRequest_Validate(t *testing.T) {
	base := CreatePricingStageRequest{
		DistanceID: uuid.New(),
		Name:       "Early bird",
		Price:      500,
		StartDate:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:    time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	tests := []struct {
		name    string
		mutate  func(r *CreatePricingStageRequest)
		wantErr bool
	}{
		{name: "valid", mutate: func(r *CreatePricingStageRequest) {}, wantErr: false},
		{name: "missing distance", mutate: func(r *CreatePricingStageRequest) { r.DistanceID = uuid.Nil }, wantErr: true},
		{name: "missing start_date", mutate: func(r *CreatePricingStageRequest) { r.StartDate = time.Time{} }, wantErr: true},
		{name: "missing end_date", mutate: func(r *CreatePricingStageRequest) { r.EndDate = time.Time{} }, wantErr: true},
		{name: "negative price", mutate: func(r *CreatePricingStageRequest) { r.Price = -1 }, wantErr: true},
		{name: "name too long", mutate: func(r *CreatePricingStageRequest) {
			r.Name = strings.Repeat("a", 101)
		}, wantErr: true},
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
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

func TestCreateSponsorRequest_Validate(t *testing.T) {
	base := CreateSponsorRequest{
		EventID:    uuid.New(),
		Name:       "Acme",
		Tier:       "gold",
		LogoURL:    "https://acme.example/logo.png",
		WebsiteURL: "https://acme.example",
	}

	tests := []struct {
		name    string
		mutate  func(r *CreateSponsorRequest)
		wantErr bool
	}{
		{name: "valid", mutate: func(r *CreateSponsorRequest) {}, wantErr: false},
		{name: "missing event id", mutate: func(r *CreateSponsorRequest) { r.EventID = uuid.Nil }, wantErr: true},
		{name: "missing name", mutate: func(r *CreateSponsorRequest) { r.Name = "" }, wantErr: true},
		{name: "missing tier", mutate: func(r *CreateSponsorRequest) { r.Tier = "" }, wantErr: true},
		{name: "bad logo url", mutate: func(r *CreateSponsorRequest) { r.LogoURL = "not a url" }, wantErr: true},
		{name: "bad website url", mutate: func(r *CreateSponsorRequest) { r.WebsiteURL = "no scheme" }, wantErr: true},
		{name: "empty urls ok (omitempty)", mutate: func(r *CreateSponsorRequest) {
			r.LogoURL = ""
			r.WebsiteURL = ""
		}, wantErr: false},
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
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

func TestCreateRegistrationLocationRequest_Validate(t *testing.T) {
	base := CreateRegistrationLocationRequest{
		EventID:      uuid.New(),
		LocationName: "Plaza Principal",
		Address:      "Av. Reforma 100, Centro",
		Schedule:     "9:00-18:00",
	}

	tests := []struct {
		name    string
		mutate  func(r *CreateRegistrationLocationRequest)
		wantErr bool
	}{
		{name: "valid", mutate: func(r *CreateRegistrationLocationRequest) {}, wantErr: false},
		{name: "missing event id", mutate: func(r *CreateRegistrationLocationRequest) { r.EventID = uuid.Nil }, wantErr: true},
		{name: "missing location name", mutate: func(r *CreateRegistrationLocationRequest) { r.LocationName = "" }, wantErr: true},
		{name: "missing address", mutate: func(r *CreateRegistrationLocationRequest) { r.Address = "" }, wantErr: true},
		{name: "address too short", mutate: func(r *CreateRegistrationLocationRequest) { r.Address = "abc" }, wantErr: true},
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
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

func TestEventFieldConfigInput_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     EventFieldConfigInput
		wantErr bool
	}{
		{name: "valid", cfg: EventFieldConfigInput{SectionName: "sponsors", IsEnabled: true, IsRequired: false}, wantErr: false},
		{name: "empty section", cfg: EventFieldConfigInput{SectionName: ""}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateStruct(tt.cfg)
			if tt.wantErr && err == nil {
				t.Fatal("expected validation error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestUpdateEventFieldConfigRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     UpdateEventFieldConfigRequest
		wantErr bool
	}{
		{
			name: "valid single field",
			req: UpdateEventFieldConfigRequest{
				Fields: []EventFieldConfigInput{{SectionName: "sponsors"}},
			},
			wantErr: false,
		},
		{
			name: "missing fields",
			req:  UpdateEventFieldConfigRequest{Fields: nil},
			wantErr: true,
		},
		{
			// validator's "required" only rejects the zero value (nil slice);
			// an explicitly-constructed empty (non-nil) slice passes.
			name: "empty fields list passes (non-nil)",
			req:  UpdateEventFieldConfigRequest{Fields: []EventFieldConfigInput{}},
			wantErr: false,
		},
		{
			name: "field missing section name",
			req: UpdateEventFieldConfigRequest{
				Fields: []EventFieldConfigInput{{SectionName: ""}},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateStruct(tt.req)
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
// Routes that early-return (e.g. UUID parsing, body validation) can still be
// exercised safely against this handler; routes that would call the service
// must not be invoked.
func newHandlerWithNilService() *Handler {
	return NewHandler(nil, validate)
}

// ============ HTTP behavior tests ============

// newRouterWithJWT mounts a single route behind JWTAuth middleware and returns
// the router. Tests use this to verify that protected routes return 401 when
// the caller is missing or sends a malformed token.
func newRouterWithJWT(handler http.HandlerFunc, pattern string) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.JWTAuth(testJWTConfig()))
	r.Method("GET", pattern, handler)
	return r
}

func TestHandler_Auth_NoToken_Returns401(t *testing.T) {
	h := newHandlerWithNilService()
	r := newRouterWithJWT(h.GetEvent, "/{id}")

	req := httptest.NewRequest(http.MethodGet, "/"+uuid.New().String(), nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", rr.Code)
	}
}

func TestHandler_Auth_BadHeaderFormat_Returns401(t *testing.T) {
	h := newHandlerWithNilService()
	r := newRouterWithJWT(h.GetEvent, "/{id}")

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
	r := newRouterWithJWT(h.GetEvent, "/{id}")

	req := httptest.NewRequest(http.MethodGet, "/"+uuid.New().String(), nil)
	req.Header.Set("Authorization", "Bearer not.a.real.jwt")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for invalid token, got %d", rr.Code)
	}
}

// ============ Invalid UUID / bad body / status patch tests ============
//
// The handlers below all short-circuit BEFORE reaching the service, so we can
// safely use a Handler whose service is nil. They cover the early-return
// "not found / bad input" surface area called out in the test spec.

func TestHandler_GetEvent_InvalidUUID_Returns400(t *testing.T) {
	h := newHandlerWithNilService()

	r := chi.NewRouter()
	r.Get("/{id}", h.GetEvent)

	req := httptest.NewRequest(http.MethodGet, "/not-a-uuid", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid UUID, got %d", rr.Code)
	}
}

func TestHandler_GetAward_InvalidUUID_Returns400(t *testing.T) {
	h := newHandlerWithNilService()

	r := chi.NewRouter()
	r.Get("/awards/{id}", h.GetAward)

	req := httptest.NewRequest(http.MethodGet, "/awards/abc", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid award UUID, got %d", rr.Code)
	}
}

func TestHandler_GetSponsor_InvalidUUID_Returns400(t *testing.T) {
	h := newHandlerWithNilService()

	r := chi.NewRouter()
	r.Get("/sponsors/{id}", h.GetSponsor)

	req := httptest.NewRequest(http.MethodGet, "/sponsors/zzz", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid sponsor UUID, got %d", rr.Code)
	}
}

func TestHandler_GetPricingStage_InvalidUUID_Returns400(t *testing.T) {
	h := newHandlerWithNilService()

	r := chi.NewRouter()
	r.Get("/pricing/{id}", h.GetPricingStage)

	req := httptest.NewRequest(http.MethodGet, "/pricing/123", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid pricing stage UUID, got %d", rr.Code)
	}
}

func TestHandler_GetRegistrationLocation_InvalidUUID_Returns400(t *testing.T) {
	h := newHandlerWithNilService()

	r := chi.NewRouter()
	r.Get("/locations/{id}", h.GetRegistrationLocation)

	req := httptest.NewRequest(http.MethodGet, "/locations/bad", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid location UUID, got %d", rr.Code)
	}
}

func TestHandler_ListAwards_InvalidDistanceID_Returns400(t *testing.T) {
	h := newHandlerWithNilService()

	r := chi.NewRouter()
	r.Get("/distance/{distance_id}/awards", h.ListAwards)

	req := httptest.NewRequest(http.MethodGet, "/distance/not-uuid/awards", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid distance ID, got %d", rr.Code)
	}
}

func TestHandler_UpdateEventStatus_InvalidUUID_Returns400(t *testing.T) {
	h := newHandlerWithNilService()

	r := chi.NewRouter()
	r.Patch("/{id}/status", h.UpdateEventStatus)

	body := strings.NewReader(`{"status":"published"}`)
	req := httptest.NewRequest(http.MethodPatch, "/bad-id/status", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid UUID, got %d", rr.Code)
	}
}

func TestHandler_UpdateEventStatus_EmptyStatus_Returns400(t *testing.T) {
	h := newHandlerWithNilService()

	r := chi.NewRouter()
	r.Patch("/{id}/status", h.UpdateEventStatus)

	body := strings.NewReader(`{"status":""}`)
	req := httptest.NewRequest(http.MethodPatch, "/"+uuid.New().String()+"/status", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty status, got %d", rr.Code)
	}
}

func TestHandler_UpdateEventStatus_EmptyBody_Returns400(t *testing.T) {
	h := newHandlerWithNilService()

	r := chi.NewRouter()
	r.Patch("/{id}/status", h.UpdateEventStatus)

	req := httptest.NewRequest(http.MethodPatch, "/"+uuid.New().String()+"/status", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty body, got %d", rr.Code)
	}
}

func TestHandler_CreateEvent_InvalidJSON_Returns400(t *testing.T) {
	h := newHandlerWithNilService()

	r := chi.NewRouter()
	r.Post("/", h.CreateEvent)

	body := strings.NewReader(`{"name": this is not json`)
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d", rr.Code)
	}
}

func TestHandler_UpdateEvent_InvalidUUID_Returns400(t *testing.T) {
	h := newHandlerWithNilService()

	r := chi.NewRouter()
	r.Put("/{id}", h.UpdateEvent)

	body := strings.NewReader(`{}`)
	req := httptest.NewRequest(http.MethodPut, "/not-a-uuid", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid UUID, got %d", rr.Code)
	}
}

func TestHandler_DeleteEvent_InvalidUUID_Returns400(t *testing.T) {
	h := newHandlerWithNilService()

	r := chi.NewRouter()
	r.Delete("/{id}", h.DeleteEvent)

	req := httptest.NewRequest(http.MethodDelete, "/bad", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid UUID, got %d", rr.Code)
	}
}
