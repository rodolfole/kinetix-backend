package orders

import (
	"bytes"
	"context"
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

// Shared validator instance for DTO validation tests.
var validate = validator.New()

// newRouter wires the handler Routes onto a fresh chi router. The service is
// nil-repo so any test that reaches service.* would nil-deref; only test paths
// that fail before reaching the service.

// adminCtx attaches JWT claims with the admin role, as middleware would.
func adminCtx(r *http.Request) *http.Request {
	return r.WithContext(context.WithValue(context.Background(), middleware.ClaimsKey, &auth.Claims{UserID: "u-admin", Role: "admin"}))
}

func newRouter() (http.Handler, *Handler) {
	svc := NewService(nil, nil, nil, validate)
	h := NewHandler(svc, "test-webhook-secret", nil)
	r := chi.NewRouter()
	h.Routes(r)
	return r, h
}

// authedRouter wraps the routes with JWT auth middleware so we can verify
// auth-required behaviour without exercising the service.
func authedRouter(t *testing.T) http.Handler {
	t.Helper()
	h, _ := newRouter()
	cfg := auth.Config{SecretKey: "test-secret", TokenExpiry: time.Hour}
	return middleware.JWTAuth(cfg)(h)
}

// validCreateOrder returns a CreateOrderRequest that passes validation.
func validCreateOrder() CreateOrderRequest {
	return CreateOrderRequest{
		RegistrationID: uuid.New(),
		ParticipantID:  uuid.New(),
		EventID:        uuid.New(),
		Amount:         100.50,
		Currency:       "MXN",
	}
}

func TestCreateOrderRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateOrderRequest
		wantErr bool
	}{
		{
			name:    "valid",
			req:     validCreateOrder(),
			wantErr: false,
		},
		{
			name:    "missing currency",
			req:     func() CreateOrderRequest { r := validCreateOrder(); r.Currency = ""; return r }(),
			wantErr: true,
		},
		{
			name:    "currency wrong length",
			req:     func() CreateOrderRequest { r := validCreateOrder(); r.Currency = "MX"; return r }(),
			wantErr: true,
		},
		{
			name:    "negative amount",
			req:     func() CreateOrderRequest { r := validCreateOrder(); r.Amount = -1; return r }(),
			wantErr: true,
		},
		{
			name:    "zero amount",
			req:     func() CreateOrderRequest { r := validCreateOrder(); r.Amount = 0; return r }(),
			wantErr: true,
		},
		{
			name: "missing event id",
			req: func() CreateOrderRequest {
				r := validCreateOrder()
				r.EventID = uuid.Nil
				return r
			}(),
			wantErr: true,
		},
		{
			name: "missing participant id",
			req: func() CreateOrderRequest {
				r := validCreateOrder()
				r.ParticipantID = uuid.Nil
				return r
			}(),
			wantErr: true,
		},
		{
			name: "missing registration id",
			req: func() CreateOrderRequest {
				r := validCreateOrder()
				r.RegistrationID = uuid.Nil
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
				t.Fatalf("unexpected validation error: %v", err)
			}
		})
	}
}

func TestUpdateOrderPaymentInfoRequest_Validate(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name    string
		req     UpdateOrderPaymentInfoRequest
		wantErr bool
	}{
		{
			name: "valid",
			req: UpdateOrderPaymentInfoRequest{
				PaymentMethod:        "credit_card",
				MercadoPagoPaymentID: "12345",
				TransactionDate:      now,
				Status:               "paid",
			},
			wantErr: false,
		},
		{
			name: "all empty strings allowed",
			req: UpdateOrderPaymentInfoRequest{
				PaymentMethod:        "",
				MercadoPagoPaymentID: "",
				Status:               "",
			},
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

func TestCreate_BadJSONReturns400(t *testing.T) {
	r, _ := newRouter()
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestGetByID_BadUUIDReturns400(t *testing.T) {
	r, _ := newRouter()
	req := httptest.NewRequest(http.MethodGet, "/not-a-uuid", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad UUID, got %d", rr.Code)
	}
}

func TestList_NoQueryParamsReturns400(t *testing.T) {
	r, _ := newRouter()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 when no event_id/participant_id/status provided, got %d", rr.Code)
	}
}

func TestUpdatePaymentInfo_BadUUIDReturns400(t *testing.T) {
	r, _ := newRouter()
	body, _ := json.Marshal(UpdateOrderPaymentInfoRequest{Status: "paid"})
	req := adminCtx(httptest.NewRequest(http.MethodPut, "/not-a-uuid/payment", bytes.NewReader(body)))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad UUID, got %d", rr.Code)
	}
}

func TestUpdatePaymentInfo_BadJSONReturns400(t *testing.T) {
	r, _ := newRouter()
	id := uuid.New().String()
	req := adminCtx(httptest.NewRequest(http.MethodPut, "/"+id+"/payment", bytes.NewReader([]byte("{not-json"))))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad JSON, got %d", rr.Code)
	}
}

func TestUpdateStatus_BadUUIDReturns400(t *testing.T) {
	r, _ := newRouter()
	body, _ := json.Marshal(map[string]string{"status": "paid"})
	req := adminCtx(httptest.NewRequest(http.MethodPut, "/not-a-uuid/status", bytes.NewReader(body)))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad UUID, got %d", rr.Code)
	}
}

func TestUpdateStatus_BadJSONReturns400(t *testing.T) {
	r, _ := newRouter()
	id := uuid.New().String()
	req := adminCtx(httptest.NewRequest(http.MethodPut, "/"+id+"/status", bytes.NewReader([]byte("garbage"))))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad JSON, got %d", rr.Code)
	}
}

func TestCreate_NoAuthHeaderReturns401(t *testing.T) {
	h := authedRouter(t)
	body, _ := json.Marshal(validCreateOrder())
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
	body, _ := json.Marshal(validCreateOrder())
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer not.a.valid.jwt")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 with bad token, got %d", rr.Code)
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
