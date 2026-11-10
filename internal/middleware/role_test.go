package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"kinetix-api/internal/auth"
)

func TestRequireRole_RejectsNonMatching(t *testing.T) {
	role := "admin"
	mw := RequireRole(role)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Claims with a non-matching role go in via context.
	claims := &auth.Claims{Role: "organizer"}
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req = req.WithContext(context.WithValue(req.Context(), ClaimsKey, claims))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d (body=%s)", rr.Code, rr.Body.String())
	}
}

func TestRequireRole_AllowsMatching(t *testing.T) {
	mw := RequireRole("admin")
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	claims := &auth.Claims{Role: "admin"}
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req = req.WithContext(context.WithValue(req.Context(), ClaimsKey, claims))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d (body=%s)", rr.Code, rr.Body.String())
	}
}

func TestRequireRole_RejectsMissingClaims(t *testing.T) {
	mw := RequireRole("admin")
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// No claims in context.
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}
