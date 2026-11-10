package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORS(t *testing.T) {
	allowed := []string{"http://localhost:4325", "https://kinetix.events"}
	tests := []struct {
		name        string
		method      string
		origin      string
		wantStatus  int
		wantOrigin  string
		wantCreds   string
		wantNextRun bool
	}{
		{"allowed origin echoed", http.MethodGet, "http://localhost:4325", http.StatusOK, "http://localhost:4325", "true", true},
		{"second allowed origin", http.MethodGet, "https://kinetix.events", http.StatusOK, "https://kinetix.events", "true", true},
		{"unknown origin not echoed", http.MethodGet, "https://evil.example", http.StatusOK, "", "", true},
		{"preflight answered", http.MethodOptions, "http://localhost:4325", http.StatusNoContent, "http://localhost:4325", "true", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ran := false
			h := CORS(allowed)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ran = true
				w.WriteHeader(http.StatusOK)
			}))
			req := httptest.NewRequest(tt.method, "/api/v1/events", nil)
			req.Header.Set("Origin", tt.origin)
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus || ran != tt.wantNextRun {
				t.Fatalf("status=%d next=%v, want %d/%v", rr.Code, ran, tt.wantStatus, tt.wantNextRun)
			}
			if got := rr.Header().Get("Access-Control-Allow-Origin"); got != tt.wantOrigin {
				t.Errorf("Allow-Origin=%q, want %q", got, tt.wantOrigin)
			}
			if got := rr.Header().Get("Access-Control-Allow-Credentials"); got != tt.wantCreds {
				t.Errorf("Allow-Credentials=%q, want %q", got, tt.wantCreds)
			}
			// Vary: Origin is set on every response so caches never
			// serve one origin's response to another.
			if got := rr.Header().Get("Vary"); got != "Origin" {
				t.Errorf("Vary=%q, want Origin", got)
			}
		})
	}
}
