# ADR-4: JWT authentication

- Status: Accepted
- Date: 2026-09-15
- Deciders: Kinetix team
- Related: ADR-2, ADR-11, docs/API_CONTRACTS.md, docs/PATTERNS.md

## Context
The frontend (Astro SSR) calls the API on behalf of three roles: `admin`, `organizer`, `participant` (`users.role` CHECK, migration 012). The API is stateless and runs as a single binary. Auth code lives in `internal/auth` (`auth.go`, `service.go`, `handler.go`) and `internal/middleware/middleware.go`.

## Decision
- Passwords are hashed with bcrypt (`bcrypt.DefaultCost`) in `internal/auth/service.go`.
- Sessions are JWT HS256 (`golang-jwt/jwt/v5`). Claims: `user_id`, `role`, plus registered claims with issuer `kinetix-api` (`internal/auth/auth.go`).
- Access token expiry is 24h and refresh expiry 7d, set in `cmd/api/main.go`. The signing key comes from `JWT_SECRET` via `internal/config`.
- `JWTAuth` reads `Authorization: Bearer <token>`, validates it and stores `*auth.Claims` in the request context; `RequireRole(role)` returns 403 when the role differs (`internal/middleware/middleware.go`).
- Login, registration (organizer, participant) and `POST /auth/refresh` are public and rate limited (ADR-11).

## Consequences
### Positive
- Stateless verification: no session store, any replica can validate a token.
- Role is in the token, so `RequireRole` costs no query.

### Negative / risks
- The refresh token is produced by the same `GenerateToken` as the access token (`internal/auth/service.go`, Login/Register/RefreshToken) with the same claims and expiry, so it is interchangeable with the access token; `RefreshToken` validates it with `ValidateToken` and checks no `typ`. docs/API_CONTRACTS.md states the target (`typ: "refresh"`).
- `TokenExpiry`/`RefreshExpiry` are hardcoded in `main.go`; `JWT_TOKEN_EXPIRY` is read but unused.
- `config.Load` defaults `JWT_SECRET` to a known string (`internal/config/config.go`), so the `panic` for an empty secret is unreachable and a missing variable silently uses a public key. `docker-compose.yml` sets `dev-secret-change-in-production`.
- No revocation: a stolen token is valid until expiry; role changes apply only at next login.
- Only `/admin/*` enforces a role; other protected routes accept any valid token.
- `JWTAuth` errors use `json.Write` with ad hoc maps instead of the error envelope (ADR-6).

## Alternatives considered
- Server-side sessions (cookie + store): rejected, needs shared storage for multiple replicas.
- Opaque tokens with introspection: rejected, adds a lookup per request.
- External IdP (Auth0, Keycloak): rejected for now, extra service and cost; revisit if SSO is required.
