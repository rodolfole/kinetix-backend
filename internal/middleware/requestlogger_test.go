package middleware

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"kinetix-api/internal/auth"
)

func TestRequestLoggerEmitsJSON(t *testing.T) {
	buf := &bytes.Buffer{}
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(old) })

	h := RequestLogger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	claims := &auth.Claims{UserID: "u-1", Role: "organizer"}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/events", nil).WithContext(
		context.WithValue(context.Background(), ClaimsKey, claims),
	)
	rw := httptest.NewRecorder()
	h.ServeHTTP(rw, req)

	line := buf.String()
	for _, want := range []string{`"method":"GET"`, `"path":"/api/v1/events"`, `"status":418`, `"user_id":"u-1"`} {
		if !strings.Contains(line, want) {
			t.Errorf("log line missing %s: %s", want, line)
		}
	}
}

func TestRequestIDResolver(t *testing.T) {
	RequestIDResolver = func(r *http.Request) string { return "rid-42" }
	t.Cleanup(func() { RequestIDResolver = nil })

	if got := idFromContext(httptest.NewRequest(http.MethodGet, "/", nil)); got != "rid-42" {
		t.Fatalf("request id = %q, want rid-42", got)
	}
}
