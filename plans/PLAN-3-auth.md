# PLAN-3: Auth hardening and test coverage

- Status: Draft
- Created: 2026-09-25 | Updated: 2026-11-06
- Spec: docs/specs/SPEC-auth.md
- ADRs: ADR-4, ADR-11, ADR-6

## Goal
The auth module has service-level tests, an integration test for the rate limiter, and middleware responses that follow RFC 6750 and 429 conventions.

## Scope
### In
- Rate-limit integration test, sentinel-error unit tests, `WWW-Authenticate` and `Retry-After` headers, JWT claim documentation.
### Out
- Per-user rate limits (would need session state we do not store).
- Token revocation / blocklist (log-out stays client-side).
- Token-type and secret fixes: PLAN-15 (B1, B2).

## Design
- Keep HS256 with the secret from `config.Load()`; keep `AuthRateLimit()` (`IPRateLimit(10, time.Minute)`) as the only limiter.
- Handler uses `switch err` on sentinel errors; every new service error must be a sentinel handled there.
- The limiter keys on `r.RemoteAddr`; behind a proxy, chi `RealIP` rewrites it first (ADR-11).

## Tasks
- [ ] T1 Add integration test for the rate limiter (11th login from one IP returns 429 with `{"error":"too many requests, please try again later"}`) — test first: `TestAuthRateLimit_Trips` — files: `internal/integration_test/integration_test.go`
- [ ] T2 Add unit tests for service sentinels (`ErrInvalidCredentials` unknown email and bad password, `ErrUserInactive`, `ErrEmailTaken`) — test first: `internal/auth/service_test.go` — files: `internal/auth/service_test.go`
- [ ] T3 Emit `WWW-Authenticate: Bearer error="invalid_token"` on 401 from `JWTAuth` — test first: `internal/middleware/middleware_test.go` (missing header, bad prefix, bad signature) — files: `internal/middleware/middleware.go`, `internal/middleware/middleware_test.go`
- [ ] T4 Emit `Retry-After` (window in seconds) on 429 from `IPRateLimit` — test first: `internal/middleware/ratelimit_test.go` — files: `internal/middleware/ratelimit.go`, `internal/middleware/ratelimit_test.go`
- [ ] T5 Document the JWT claim contract on `Claims` (`user_id`, `role`; registered `iss=kinetix-api`, `iat`, `nbf`, `exp`) — test first: n/a (comment only), `go vet` — files: `internal/auth/auth.go`

## Verification
```bash
go test -race -count=1 ./internal/auth/... ./internal/middleware/...
go test -race -count=1 ./internal/integration_test/... -run RateLimit   # needs the compose stack (and build tag after PLAN-15 B8)
```
Acceptance: 11th auth request gets 429, sentinels covered, 401 carries `WWW-Authenticate`.

## Risks and rollback
- New 11th-request test can stall on the token cache: call `JSONRequest` directly with a bad password.
- `WWW-Authenticate` is additive; `Retry-After` exposes the window size (already public in source).

## Open questions
- Separate rate-limit buckets for login and register (today they share 10/min)?
- Server-side revocation for log-out (table or Redis)?
