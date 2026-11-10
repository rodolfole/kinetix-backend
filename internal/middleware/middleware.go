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
			claims, err := auth.ValidateAccessToken(cfg, token)
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

// RequireRole returns a middleware that allows the request only if the JWT
// claim's Role matches the given role. Must be chained AFTER JWTAuth so
// claims are in the context. Returns 403 (not 302) for non-matching roles
// because the caller is authenticated but unauthorized for the resource.
func RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := r.Context().Value(ClaimsKey).(*auth.Claims)
			if !ok || claims == nil {
				json.WriteError(w, http.StatusUnauthorized, "missing auth claims")
				return
			}
			if claims.Role != role {
				json.WriteError(w, http.StatusForbidden, "insufficient role")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAnyRole allows the request when the JWT role is one of roles.
// Same contract as RequireRole: 401 without claims, 403 for other roles.
func RequireAnyRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := r.Context().Value(ClaimsKey).(*auth.Claims)
			if !ok || claims == nil {
				json.WriteError(w, http.StatusUnauthorized, "missing auth claims")
				return
			}
			if _, ok := allowed[claims.Role]; !ok {
				json.WriteError(w, http.StatusForbidden, "insufficient role")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// CORS creates CORS middleware that echoes only the configured origins
// (config.ServerConfig.CORSAllowedOrigins).
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
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

			// The origin participates in cache keys: do not return
			// a cached response across origins.
			w.Header().Set("Vary", "Origin")
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
