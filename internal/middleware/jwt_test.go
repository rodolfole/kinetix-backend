package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"kinetix-api/internal/auth"
)

func TestJWTAuth_TokenTypes(t *testing.T) {
	cfg := auth.Config{SecretKey: "test-secret", TokenExpiry: time.Hour, RefreshExpiry: 24 * time.Hour}
	access, err := auth.GenerateToken(cfg, "user-1", "organizer")
	if err != nil {
		t.Fatal(err)
	}
	refresh, err := auth.GenerateRefreshToken(cfg, "user-1", "organizer")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		header     string
		wantStatus int
		wantCalled bool
	}{
		{"access token passes", "Bearer " + access, http.StatusOK, true},
		{"refresh token rejected", "Bearer " + refresh, http.StatusUnauthorized, false},
		{"missing header rejected", "", http.StatusUnauthorized, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			h := JWTAuth(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				if c, ok := r.Context().Value(ClaimsKey).(*auth.Claims); !ok || c.UserID != "user-1" {
					t.Errorf("claims not in context: %#v", r.Context().Value(ClaimsKey))
				}
				w.WriteHeader(http.StatusOK)
			}))
			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			if rr.Code != tt.wantStatus || called != tt.wantCalled {
				t.Fatalf("status=%d called=%v, want %d/%v (body=%s)", rr.Code, called, tt.wantStatus, tt.wantCalled, rr.Body.String())
			}
		})
	}
}
