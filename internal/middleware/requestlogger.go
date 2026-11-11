package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

type loggingResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *loggingResponseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// RequestLogger logs one JSON line per request with the chi request id.
func RequestLogger(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lw := &loggingResponseWriter{ResponseWriter: w, status: http.StatusOK}
		handler.ServeHTTP(lw, r)

		attrs := []slog.Attr{
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", lw.status),
			slog.String("request_id", idFromContext(r)),
			slog.Int64("duration_ms", time.Since(start).Milliseconds()),
		}
		if claims, ok := ClaimsFrom(r.Context()); ok {
			attrs = append(attrs, slog.String("user_id", claims.UserID))
		}
		slog.LogAttrs(r.Context(), slog.LevelInfo, "http request", attrs...)
	})
}

// RequestIDResolver reads the request id. Injected in cmd/api to avoid the
// chi import here; nil falls back to the response header.
var RequestIDResolver func(r *http.Request) string

// idFromContext reads the request id (chi's RequestID middleware sets a
// context value). Injected in cmd/api via RequestIDResolver to avoid the chi
// import here; falls back to the response header.
func idFromContext(r *http.Request) string {
	if RequestIDResolver != nil {
		if id := RequestIDResolver(r); id != "" {
			return id
		}
	}
	return r.Header.Get("X-Request-Id")
}
