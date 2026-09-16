package middleware

import (
	"net/http"
	"sync"
	"time"

	"kinetix-api/internal/json"
)

// RateLimiter holds state for a simple token-bucket rate limiter
type RateLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

// NewRateLimiter creates a new rate limiter
// limit: max requests per window
// window: time window duration
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
	// Start cleanup goroutine
	go rl.cleanup()
	return rl
}

// Allow checks if a request from the given key should be allowed
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.window)

	// Filter out old requests
	var validRequests []time.Time
	for _, t := range rl.requests[key] {
		if t.After(windowStart) {
			validRequests = append(validRequests, t)
		}
	}

	if len(validRequests) >= rl.limit {
		rl.requests[key] = validRequests
		return false
	}

	rl.requests[key] = append(validRequests, now)
	return true
}

// cleanup removes old entries periodically
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		windowStart := now.Add(-rl.window)
		for key, times := range rl.requests {
			var valid []time.Time
			for _, t := range times {
				if t.After(windowStart) {
					valid = append(valid, t)
				}
			}
			if len(valid) == 0 {
				delete(rl.requests, key)
			} else {
				rl.requests[key] = valid
			}
		}
		rl.mu.Unlock()
	}
}

// IPRateLimit creates middleware that limits requests by IP
func IPRateLimit(limit int, window time.Duration) func(http.Handler) http.Handler {
	limiter := NewRateLimiter(limit, window)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			if ip == "" {
				next.ServeHTTP(w, r)
				return
			}

			if !limiter.Allow(ip) {
				json.Write(w, http.StatusTooManyRequests, map[string]string{
					"error": "too many requests, please try again later",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// AuthRateLimit creates middleware that limits auth endpoint requests by IP
// Default: 10 attempts per minute per IP
func AuthRateLimit() func(http.Handler) http.Handler {
	return IPRateLimit(10, time.Minute)
}