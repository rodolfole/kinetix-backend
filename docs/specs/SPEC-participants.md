# Spec: Participants Module

- Status: Implemented
- Plan: plans/PLAN-11-participants.md

## Objective

CRUD API for race participants. A participant is the human who registers for an
event — separate from the user account used to log in. Organizers create and
manage participants through the backend; participants can also be created
in-band when a registration is submitted (see `SPEC-registrations.md`).

The module exposes: create, list (paginated), get-by-id, update, delete,
plus a focused `UpdateTeamSize` helper used by registration flows.

## Tech Stack

- Go 1.22+, Chi v5 router
- PostgreSQL via pgx/v5 + sqlc-generated queries
- `go-playground/validator/v10` for struct validation
- `kinetix-api/internal/json` helpers — **not used here** (see Boundaries)

## Commands

```bash
go build ./...
go test ./internal/participants/...
go run ./cmd/api                  # full stack
docker compose up -d               # postgres + api
```

## Project Structure

```
internal/participants/
├── handler.go      # HTTP layer (raw http.Error — see Boundaries)
├── service.go      # business logic + validation
├── participant.go  # sqlc-backed Repository + model conversion
└── model.go        # domain structs + DTOs
```

Mounted at `POST /api/v1/participants` and friends, behind JWT auth
(`kinetix_middleware.JWTAuth`).

## Endpoints

| Method | Path                       | Body                              | Returns                   |
|--------|----------------------------|-----------------------------------|---------------------------|
| POST   | `/api/v1/participants`     | `CreateParticipantRequest`        | `ParticipantResponse`     |
| GET    | `/api/v1/participants`     | — (`?limit=&offset=`)             | `ParticipantsListResponse`|
| GET    | `/api/v1/participants/{id}`| —                                 | `ParticipantResponse`     |
| PUT    | `/api/v1/participants/{id}`| `UpdateParticipantRequest`        | `ParticipantResponse`     |
| DELETE | `/api/v1/participants/{id}`| —                                 | `GenericResponse`         |

`UpdateTeamSize` is **not** exposed as a route — only used internally by
`registrations.Service.CreateWithParticipant` when an existing participant
is re-using the same record.

### Request shape (`CreateParticipantRequest`)

```json
{
  "first_name": "Juan",
  "last_name": "Pérez",
  "second_last_name": "García",
  "gender": "male",
  "birth_date": "1990-04-12T00:00:00Z",
  "email": "juan@example.com",
  "phone": "+34600000000",
  "country": "ES",
  "municipality": "Valencia",
  "state": "Valencia",
  "zip_code": "46001",
  "team": "Runners Club",
  "size": "M",
  "emergency_contact_name": "Ana Pérez",
  "emergency_contact_phone": "+34600000001",
  "medical_info": "None"
}
```

`size` must be one of `XS S M L XL XXL`; `country` is exactly 2 chars.

## Code Style

```go
// internal/participants/handler.go
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
    var req CreateParticipantRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    participant, err := h.service.Create(r.Context(), req)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    json.NewEncoder(w).Encode(ParticipantResponse{...})
}
```

Handlers follow the standard `decode → service → encode` pattern. Service
applies struct validation before delegating to repository. Repository owns
sqlc types and converts to/from domain `Participant` via `toParticipant`.

## Testing Strategy

- `internal/integration_test/integration_test.go` is the integration baseline.
  It does not currently exercise participants; add tests here when expanding.
- Repository tests live alongside the package; use `internal/testhelpers` for
  fixtures.
- Validation cases to cover: missing email, bad country length, invalid size,
  invalid phone length, missing required fields.

## Boundaries

- **Always do:** Validate every required field at the struct tag level; convert
  nullable PostgreSQL columns via `internal/database/convert` helpers.
- **Ask first:** Adding new optional fields to `Participant` (schema change),
  changing validation rules, switching from `http.Error` to the shared
  `internal/json` helpers.
- **Never do:**
  - Use `internal/json` helpers — the participants handler is the **odd one
    out** in this codebase, using raw `http.Error` + inline `json.NewEncoder`.
    Track that as a known inconsistency.
  - Expose passwords or hashed credentials on this model — none exist here by
    design.
  - Cascade-delete without explicit confirmation in `Service.Delete`.

### Known inconsistency

`internal/participants/handler.go` uses `http.Error(w, err.Error(), ...)` and
inline `json.NewEncoder(w).Encode(...)` instead of the `internal/json.Write` /
`json.WriteError` helpers that `runnerkits`, `routes`, and `tags` use. Do not
silently rewrite it without scoping the change — it's a deliberate deviation
that should be tracked separately.

## Success Criteria

- `POST /api/v1/participants` returns 200 with the created record and a UUID id.
- Validation failures return 400 with a clear message; missing rows return 404.
- `GET /api/v1/participants?limit=10&offset=0` returns paginated list with
  `total` count.
- `DELETE` is idempotent in practice: a re-delete on a removed id returns the
  underlying DB error (500) — acceptable; do not silently 200.
- The `internal/integration_test` runner hits the participants endpoint at
  least once when expanded.

## Open Questions

- Should `email` be globally unique across participants? Currently the schema
  is permissive — a duplicate email creates two participant rows.
- Should `DELETE` be soft-delete to preserve registration history? Today it is
  hard-delete and breaks referential integrity with `registrations`.
