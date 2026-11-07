# ADR-11: In-memory rate limit on auth endpoints

- Status: Accepted
- Date: 2026-09-15
- Deciders: Kinetix team
- Related: ADR-2, ADR-4, docs/ARCHITECTURE.md

## Context
Login and registration are public and sensitive to credential stuffing and bcrypt CPU abuse. The deployment is a single API container (ADR-9), so a shared store is not available yet. `internal/middleware/ratelimit.go` provides the limiter, mounted on `/api/v1/auth` in `cmd/api/main.go`.

## Decision
- Limit `/api/v1/auth/*` to 10 requests per minute per client (`AuthRateLimit()` = `IPRateLimit(10, time.Minute)`).
- State is an in-process `map[string][]time.Time` guarded by a mutex; a sliding window keeps timestamps per key. A goroutine prunes stale keys every 5 minutes.
- Over the limit the API returns `429 {"error":"too many requests, please try again later"}`.
- The limiter is middleware only; no other route group is rate limited.

## Consequences
### Positive
- No infrastructure: works with one process; covered by `internal/middleware/ratelimit_test.go`.
- Cheap and local to the auth routes.

### Negative / risks
- The key is `r.RemoteAddr` (`internal/middleware/ratelimit.go`), which includes the client port, so each new TCP connection gets a fresh bucket and the limit is weak. Behind a proxy it would key by proxy. `chimiddleware.RealIP` rewrites `RemoteAddr` from `X-Forwarded-For`/`X-Real-IP`, which clients can spoof when the API is exposed directly.
- State is in memory: it resets on restart and is not shared across replicas, so the effective limit is `10 x replicas`.
- The cleanup goroutine has no stop and the ticker is never closed; each `NewRateLimiter` call leaks one.
- Per-IP only: no per-account lockout for a single targeted email.
- `429` does not set `Retry-After` and does not use the error envelope `code` (ADR-6).
- Public `/orders/webhook`, uploads and catalog reads have no limit.

## Alternatives considered
- Redis-backed limiter: rejected for now, adds a service; adopt when running more than one replica.
- Proxy-level limit (nginx, API gateway): complementary, but not available in the current stack.
- `golang.org/x/time/rate` token bucket: rejected, the present sliding window is simple and tested.
