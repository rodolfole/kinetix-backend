package orders

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"kinetix-api/internal/auth"
	"kinetix-api/internal/middleware"
)

func participantClaims() *auth.Claims {
	p := uuid.New()
	return &auth.Claims{UserID: "u-part", Role: "participant", ParticipantID: &p}
}

func adminClaims() *auth.Claims {
	return &auth.Claims{UserID: "u-admin", Role: "admin"}
}

func claimsCtx(claims *auth.Claims) context.Context {
	return context.WithValue(context.Background(), middleware.ClaimsKey, claims)
}

// routesWithClaims builds the order routes as JWTAuth would attach claims.
func routesWithClaims() http.Handler {
	_, h := newRouter()
	r := chi.NewRouter()
	h.Routes(r)
	return r
}

func TestOrderMutationsRequireAdmin(t *testing.T) {
	id := uuid.New().String()

	tests := []struct {
		name       string
		claims     *auth.Claims
		method     string
		path       string
		wantStatus int
	}{
		{"payment requires admin", participantClaims(), http.MethodPut, "/" + id + "/payment", http.StatusForbidden},
		{"status requires admin", participantClaims(), http.MethodPut, "/" + id + "/status", http.StatusForbidden},
		{"payment without claims rejected", nil, http.MethodPut, "/" + id + "/payment", http.StatusUnauthorized},
		{"status without claims rejected", nil, http.MethodPut, "/" + id + "/status", http.StatusUnauthorized},
		{"admin passes middleware to decode", adminClaims(), http.MethodPut, "/" + id + "/payment", http.StatusBadRequest},
		{"participant list is untouched", participantClaims(), http.MethodGet, "/", http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			if tt.claims != nil {
				req = req.WithContext(claimsCtx(tt.claims))
			}
			rr := httptest.NewRecorder()
			routesWithClaims().ServeHTTP(rr, req)
			if rr.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body=%s)", rr.Code, tt.wantStatus, rr.Body.String())
			}
		})
	}
}
