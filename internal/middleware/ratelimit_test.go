package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiter_AllowsUpToLimit(t *testing.T) {
	rl := NewRateLimiter(3, time.Minute)
	key := "1.2.3.4"

	for i := 0; i < 3; i++ {
		if !rl.Allow(key) {
			t.Errorf("request %d should be allowed (limit=3)", i+1)
		}
	}
}

func TestRateLimiter_BlocksAfterLimit(t *testing.T) {
	rl := NewRateLimiter(3, time.Minute)
	key := "1.2.3.4"

	for i := 0; i < 3; i++ {
		rl.Allow(key)
	}
	if rl.Allow(key) {
		t.Error("4th request should be blocked")
	}
}

func TestRateLimiter_PerKeyIsolation(t *testing.T) {
	rl := NewRateLimiter(2, time.Minute)

	// Key A fills up
	rl.Allow("a")
	rl.Allow("a")
	if rl.Allow("a") {
		t.Error("key 'a' should be blocked at 3rd request")
	}

	// Key B is unaffected
	if !rl.Allow("b") {
		t.Error("key 'b' should be allowed (independent counter)")
	}
	if !rl.Allow("b") {
		t.Error("key 'b' 2nd request should be allowed")
	}
}

func TestRateLimiter_WindowExpiry(t *testing.T) {
	rl := NewRateLimiter(2, 50*time.Millisecond)
	key := "x"

	rl.Allow(key)
	rl.Allow(key)
	if rl.Allow(key) {
		t.Error("3rd request inside window should be blocked")
	}

	time.Sleep(80 * time.Millisecond)

	// After window expires, requests should be allowed again
	if !rl.Allow(key) {
		t.Error("request after window expiry should be allowed")
	}
}

func TestIPRateLimit_Middleware_Allows(t *testing.T) {
	mw := IPRateLimit(5, time.Minute)
	hits := 0
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("request %d expected 200, got %d", i+1, rr.Code)
		}
	}
	if hits != 5 {
		t.Errorf("expected 5 hits, got %d", hits)
	}
}

func TestIPRateLimit_Middleware_Blocks429(t *testing.T) {
	mw := IPRateLimit(2, time.Minute)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First 2 OK
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.RemoteAddr = "10.0.0.2:5555"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("warmup %d expected 200, got %d", i+1, rr.Code)
		}
	}

	// 3rd should be blocked with 429
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.RemoteAddr = "10.0.0.2:5555"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", rr.Code)
	}
}

func TestAuthRateLimit_DefaultLimit(t *testing.T) {
	mw := AuthRateLimit()
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Default is 10/min per IP
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
		req.RemoteAddr = "10.0.0.3:1111"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("request %d expected 200, got %d", i+1, rr.Code)
		}
	}

	// 11th should be blocked
	req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	req.RemoteAddr = "10.0.0.3:1111"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("11th request expected 429, got %d", rr.Code)
	}
}
