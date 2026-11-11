# Spec: Auth Module

- Status: Implemented
- Plan: plans/PLAN-3-auth.md

## Objective

Identity surface for organizers and participants: register, log in, and refresh.
Issues HS256 JWTs carrying `user_id` and `role` claims; passwords are bcrypt-hashed.
Mounted at `/api/v1/auth/*` behind an IP-based rate limit (10 req/min) to blunt
credential stuffing on the login/register endpoints.

Two flavours of account are created in-band:
- **Organizer** — full business profile (RFC, billing address, brand name).
- **Participant** — personal profile (name, birth date, phone).

In both cases a row is written to `users` (with `role` and the FK to
`organizers` or `participants`) and a `users_password_hashes` bcrypt digest is
stored. The session is then a JWT pair.

## Tech Stack

- Go 1.22+, Chi v5 router
- `github.com/golang-jwt/jwt/v5` for HS256 signing/validation
- `golang.org/x/crypto/bcrypt` at default cost
- `kinetix-api/internal/json` helpers for response shaping
- `kinetix-api/internal/middleware` for JWT verification on protected routes

## Commands

```bash
go build ./...
go test ./internal/auth/...                    # unit tests (auth + handler)
go test ./internal/integration_test/...        # against docker-compose stack
docker compose up -d                            # postgres + api
```

## Project Structure

```
internal/auth/
├── auth.go         # JWT helpers (Claims, Generate, Validate, Extract)
├── handler.go      # HTTP layer (4 routes under /api/v1/auth)
├── service.go      # business logic (login, register, refresh)
├── model.go        # DTOs (LoginRequest, RegisterOrganizerRequest, ...)
├── auth_test.go    # JWT unit tests
└── handler_test.go # validator + request-shape unit tests
```

JWT middleware (`JWTAuth`), CORS, IP rate limiter, and the `AuthRateLimit`
wrapper live in `internal/middleware/`.

## Endpoints

| Method | Path                                | Body                          | Returns                                |
|--------|-------------------------------------|-------------------------------|----------------------------------------|
| POST   | `/api/v1/auth/login`                | `LoginRequest`                | `RegisterResponse` (200)               |
| POST   | `/api/v1/auth/register/organizer`   | `RegisterOrganizerRequest`    | `RegisterResponse` (201)               |
| POST   | `/api/v1/auth/register/participant` | `RegisterParticipantRequest`  | `RegisterResponse` (201)              |
| POST   | `/api/v1/auth/refresh`              | `RefreshTokenRequest`         | `AuthResponse` (200)                   |

All routes share the IP rate limit `AuthRateLimit()` (= `IPRateLimit(10, 1m)`).

### JWT shape

```go
type Claims struct {
    UserID string `json:"user_id"`
    Role   string `json:"role"`
    jwt.RegisteredClaims
}
```

- Algorithm: `HS256`
- `TokenExpiry`: 24h; `RefreshExpiry`: 7d
- `Issuer`: `kinetix-api`
- Secret read from `config.Load().JWT.SecretKey` (panics if unset)

### Request/response shapes

```json
POST /api/v1/auth/login
{ "email": "contact@kinetix.com", "password": "seed1234" }

200 OK
{
  "user": { "id": "...", "email": "...", "role": "organizer", "organizer_id": "..." },
  "access_token": "eyJ...",
  "refresh_token": "eyJ...",
  "expires_in": 86400
}
```

`RegisterOrganizerRequest` validates: `Email` (required,email), `Password`
(required,min=8), `BusinessName` (required,min=2,max=255), `RFC` (required,
len=12|13 — Mexican tax ID), billing fields, `ContactEmail`, `ContactPhone`
(required,min=10,max=20), optional `LogoUrl` (omitempty,url).

`RegisterParticipantRequest` validates: `Email`, `Password`, `Gender`
(required,max=20), `FirstName`/`LastName`, optional `SecondLastName`,
`BirthDate` (required), `Phone` (required,min=10,max=20).

## Code Style

```go
// internal/auth/handler.go
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
    var req LoginRequest
    if err := jsonlib.Read(r, &req); err != nil {
        jsonlib.WriteError(w, http.StatusBadRequest, "invalid request body")
        return
    }
    if err := h.validate.Struct(&req); err != nil {
        jsonlib.WriteError(w, http.StatusBadRequest, err.Error())
        return
    }
    tokens, user, err := h.service.Login(r.Context(), req)
    if err != nil {
        switch err {
        case ErrInvalidCredentials:
            jsonlib.WriteError(w, http.StatusUnauthorized, "invalid email or password")
        case ErrUserInactive:
            jsonlib.WriteError(w, http.StatusForbidden, "account is inactive")
        default:
            jsonlib.WriteError(w, http.StatusInternalServerError, "internal server error")
        }
        return
    }
    jsonlib.Write(w, http.StatusOK, RegisterResponse{...})
}
```

Handlers: decode → validate → service → switch on sentinel error → response.

## Testing Strategy

- `internal/auth/auth_test.go` covers `GenerateToken` round-trip,
  `ValidateToken` with a wrong secret, expired-token rejection, and
  `ExtractTokenFromHeader` (bearer, lowercase-bearer rejection, double-space
  edge cases).
- `internal/auth/handler_test.go` covers validator-tag enforcement on
  `LoginRequest`, `RefreshTokenRequest`, `RegisterOrganizerRequest`,
  `RegisterParticipantRequest`, and bearer-extraction edge cases.
- `internal/integration_test/integration_test.go` covers end-to-end:
  `TestLogin_Organizer`, `TestLogin_Participant`, `TestLogin_BadCredentials`,
  `TestLogin_InvalidPayload`, `TestRefreshToken_Organizer`,
  `TestProtectedRoute_RequiresAuth`.

## Boundaries

- **Always do:** Read `JWT_SECRET` via `config.Load()` (panics if empty);
  use `internal/json` helpers; switch on sentinel errors from the service
  rather than inspecting strings.
- **Ask first:** Changing token expiry defaults, rotating the signing algorithm
  away from HS256, exposing user lookup endpoints, switching from bcrypt to
  argon2.
- **Never do:**
  - Log or echo the raw password or the JWT secret.
  - Return 401 with a body that distinguishes "email not found" from "wrong
    password" — both must funnel through `ErrInvalidCredentials`.
  - Bind a session to an inactive user.

## Success Criteria

- `POST /auth/login` with seeded credentials returns 200 and an HS256 token
  whose claims include `user_id` and `role`.
- `POST /auth/login` with bad password returns 401 (not 404, not 500).
- `POST /auth/register/organizer` and `…/participant` create the linked
  organizer/participant row plus the `users` row in a single request, return
  201, and the `users.last_login` is updated on login.
- `POST /auth/refresh` with a valid refresh token mints a new pair; with an
  expired/wrong-secret token returns 401.
- `JWTAuth` middleware rejects requests with no Authorization header, wrong
  prefix, or invalid signature with 401 and a JSON error body.
- 11th request from the same IP within a minute to `/api/v1/auth/*` returns
  429 with body `{"error":"too many requests, please try again later"}`.
- `go test ./internal/auth/... ./internal/integration_test/...` passes.

## Open Questions

- Token revocation: today JWTs are valid until expiry. No blocklist. Should
  refresh-token rotation add a server-side record?
- Should `RegisterParticipantRequest` capture a country/locale to drive the
  age-category catalog? Currently the request is locale-agnostic.
