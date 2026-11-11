# ADR-2: chi router with stdlib handlers

- Status: Accepted
- Date: 2026-09-15
- Deciders: Kinetix team
- Related: ADR-1, ADR-4, ADR-11, docs/ARCHITECTURE.md

## Context
The API needs routing with path params (`/events/{id}`), route groups with per-group middleware (JWT, role, rate limit) and graceful shutdown, without tying handlers to a framework. `cmd/api/main.go` mounts everything on a `chi.NewRouter()` and serves it with a plain `http.Server`.

## Decision
- Use `go-chi/chi/v5` as router; handlers are `func(w http.ResponseWriter, r *http.Request)`.
- Global middleware, in order: `RequestID`, `RealIP`, `Logger`, `Recoverer`, `Timeout(60s)`, project `CORS` (`cmd/api/main.go`).
- Authorization is applied per group: `/api/v1/auth` gets `AuthRateLimit`; the protected group uses `JWTAuth`; `/admin/*` adds `RequireRole("admin")`.
- Each slice exposes `Routes(r chi.Router)`; `main.go` decides what is public and protected.
- Path params are read with `chi.URLParam`.
- The server shuts down gracefully on SIGINT/SIGTERM with a 5s timeout (`cmd/api/main.go`).

## Consequences
### Positive
- Handlers and middleware are plain `net/http`: testable with `httptest`, no framework types leak in.
- Group-level middleware keeps access rules visible in one file.

### Negative / risks
- Protected routes are registered one by one in `main.go` to avoid chi mounting a JWT subrouter over public `GET /events` (comment in `cmd/api/main.go`); `events.Handler.Routes()` and `admin.Handler.Routes()` are dead code.
- Most protected routes only check that a JWT exists, not role or ownership; only `/admin/*` uses `RequireRole`.
- `Dockerfile` sets `GIN_MODE=release`, a leftover from a Gin-based draft; it has no effect.
- `/health` and `/` are hand-written handlers that do not use `internal/json`.

## Alternatives considered
- Gin/Echo: rejected, custom context types lock handlers to the framework.
- stdlib `http.ServeMux` (Go 1.22+): workable, but route groups and middleware chaining are more verbose.
- gorilla/mux: rejected, no longer the maintained default and offers no advantage over chi.
