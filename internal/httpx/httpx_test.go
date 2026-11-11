package httpx

import (
	"strings"

	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type body struct {
	Email string `json:"email" validate:"required,email"`
	Age   int    `json:"age" validate:"gte=0"`
}

func router() http.Handler {
	r := chi.NewRouter()
	r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, ok := UUIDParam(w, r, "id")
		if !ok {
			return
		}
		_ = id
		w.WriteHeader(http.StatusNoContent)
	})
	return r
}

func TestWriteErrorMapping(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"not found", fmt.Errorf("x: %w", ErrNotFound), 404, "NOT_FOUND"},
		{"conflict", fmt.Errorf("x: %w", ErrConflict), 409, "CONFLICT"},
		{"forbidden", fmt.Errorf("x: %w", ErrForbidden), 403, "FORBIDDEN"},
		{"unauth", fmt.Errorf("x: %w", ErrUnathenticated), 401, "UNAUTHORIZED"},
		{"invalid input", fmt.Errorf("x: %w", ErrInvalidInput), 400, "VALIDATION_FAILED"},
		{"upstream", fmt.Errorf("x: %w", ErrUpstream), 502, "UPSTREAM_ERROR"},
		{"unknown", errors.New("db boom"), 500, "INTERNAL"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			WriteError(rr, httptest.NewRequest(http.MethodGet, "/", nil), tt.err)
			if rr.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body=%s)", rr.Code, tt.wantStatus, rr.Body.String())
			}
			b := rr.Body.String()
			if tt.wantCode == "INTERNAL" {
				if got := `"error":"internal error"`; !contains(b, got) || contains(b, "db boom") {
					t.Fatalf("internal errors must not leak: %s", b)
				}
				return
			}
			if !contains(b, fmt.Sprintf(`"error":"x: %v"`, tt.err)[:9]) && !contains(b, `"error":"`) {
				t.Fatalf("missing error field: %s", b)
			}
			if tt.wantCode != "" && !contains(b, tt.wantCode) {
				t.Fatalf("missing code %s: %s", tt.wantCode, b)
			}
		})
	}
}

func stringsReader(s string) *strings.Reader { return strings.NewReader(s) }

func routeCtx(key, value string) *chi.Context {
	rc := chi.NewRouteContext()
	rc.URLParams.Add(key, value)
	return rc
}

func contains(hay, needle string) bool {
	return len(needle) > 0 && len(hay) >= len(needle) && (func() bool {
		for i := 0; i+len(needle) <= len(hay); i++ {
			if hay[i:i+len(needle)] == needle {
				return true
			}
		}
		return false
	})()
}

func TestUUIDParam(t *testing.T) {
	id := uuid.New()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x/"+id.String(), nil).WithContext(
		context.WithValue(context.Background(), chi.RouteCtxKey, routeCtx("id", id.String())),
	)
	got, ok := UUIDParam(rr, req, "id")
	if !ok || got != id {
		t.Fatalf("got %v ok=%v", got, ok)
	}

	rr2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/x/not-a-uuid", nil).WithContext(
		context.WithValue(context.Background(), chi.RouteCtxKey, routeCtx("id", "not-a-uuid")),
	)
	if _, ok := UUIDParam(rr2, req2, "id"); ok {
		t.Fatal("malformed id must return false")
	}
	if rr2.Code != http.StatusBadRequest || !contains(rr2.Body.String(), "INVALID_ID") {
		t.Fatalf("status=%d body=%s, want 400 INVALID_ID", rr2.Code, rr2.Body.String())
	}
}

func TestReadRejectsUnknownFields(t *testing.T) {
	rr := httptest.NewRecorder()
	var b body
	if Read(rr, httptest.NewRequest(http.MethodPost, "/", stringsReader(`{"email":"a@b.co","age":3,"extra":1}`)), &b) {
		t.Fatal("unknown fields must be rejected")
	}
	if rr.Code != http.StatusBadRequest || !contains(rr.Body.String(), "VALIDATION_FAILED") {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestWriteValidationFields(t *testing.T) {
	v := validator.New()
	rr := httptest.NewRecorder()
	err := v.Struct(body{Email: "nope", Age: -1})
	WriteValidation(rr, err)
	if rr.Code != http.StatusBadRequest || !contains(rr.Body.String(), `"fields"`) {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if !contains(rr.Body.String(), `"email"`) || !contains(rr.Body.String(), `"age"`) {
		t.Fatalf("field messages missing: %s", rr.Body.String())
	}
}
