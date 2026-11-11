package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"kinetix-api/internal/auth"
)

func TestClaimsFrom(t *testing.T) {
	claims := &auth.Claims{UserID: "u1", Role: "organizer"}

	if _, ok := ClaimsFrom(context.Background()); ok {
		t.Fatal("no claims in context must return false")
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(context.WithValue(context.Background(), ClaimsKey, claims))
	got, ok := ClaimsFrom(req.Context())
	if !ok || got != claims {
		t.Fatalf("got %v ok=%v, want the stored claims", got, ok)
	}

	if _, ok := ClaimsFrom(context.WithValue(context.Background(), ClaimsKey, "garbage")); ok {
		t.Fatal("wrong type in context must return false")
	}
}
