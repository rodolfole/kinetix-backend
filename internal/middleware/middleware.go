package middleware

import (
	"context"
	"net/http"
	"strings"

	"kinetix-api/internal/auth"
	"kinetix-api/internal/json"
)

type contextKey string

const (
	ClaimsKey contextKey = "claims"
)

// JWTAuth creates JWT authentication middleware
func JWTAuth(cfg auth.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				json.Write(w, http.StatusUnauthorized, map[string]string{
					"error": "authorization header required",
				})
				return
			}

			if !strings.HasPrefix(authHeader, "Bearer ") {
				json.Write(w, http.StatusUnauthorized, map[string]string{
					"error": "invalid authorization header format",
				})
				return
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := auth.ValidateToken(cfg, token)
			if err != nil {
				json.Write(w, http.StatusUnauthorized, map[string]string{
					"error": "invalid or expired token",
				})
				return
			}

			// Add claims to context
			ctx := context.WithValue(r.Context(), ClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// CORS creates CORS middleware
func CORS() func(http.Handler) http.Handler {
	allowedOrigins := []string{
		"http://localhost:4322", // Astro dev server
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			isAllowed := false
			for _, allowed := range allowedOrigins {
				if origin == allowed {
					isAllowed = true
					break
				}
			}

			if isAllowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "86400") // 24 hours

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Logger creates request logging middleware
func Logger() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Log request (in production, use proper logging)
			// log.Printf("%s %s", r.Method, r.URL.Path)
			next.ServeHTTP(w, r)
		})
	}
}
