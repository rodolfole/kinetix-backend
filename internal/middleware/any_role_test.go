package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"kinetix-api/internal/auth"
)

func TestRequireAnyRole(t *testing.T) {
	tests := []struct {
		name       string
		claims     *auth.Claims
		wantStatus int
		wantCalled bool
	}{
		{"organizer allowed", &auth.Claims{Role: "organizer"}, http.StatusOK, true},
		{"admin allowed", &auth.Claims{Role: "admin"}, http.StatusOK, true},
		{"participant forbidden", &auth.Claims{Role: "participant"}, http.StatusForbidden, false},
		{"missing claims unauthorized", nil, http.StatusUnauthorized, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			h := RequireAnyRole("organizer", "admin")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				w.WriteHeader(http.StatusOK)
			}))
			req := httptest.NewRequest(http.MethodPost, "/api/v1/events", nil)
			if tt.claims != nil {
				req = req.WithContext(context.WithValue(req.Context(), ClaimsKey, tt.claims))
			}
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			if rr.Code != tt.wantStatus || called != tt.wantCalled {
				t.Fatalf("status=%d called=%v, want %d/%v (body=%s)", rr.Code, called, tt.wantStatus, tt.wantCalled, rr.Body.String())
			}
		})
	}
}
