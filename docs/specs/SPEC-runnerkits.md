# Spec: Runner Kits Module

- Status: Implemented
- Plan: plans/PLAN-9-runner-kits.md

## Objective

Per-event list of kit items (shirt, medal, bib, …) shown to participants
and managed by organizers. Read access is public so the registration page
can render the kit; write access is JWT-protected.

## Tech Stack

- Go 1.22+, Chi v5 router
- PostgreSQL via pgx/v5 + sqlc
- `go-playground/validator/v10`
- `kinetix-api/internal/json` helpers (this module IS consistent — uses
  `json.Read`, `json.Write`, `json.WriteError`)

## Commands

```bash
go build ./...
go test ./internal/runnerkits/...
go run ./cmd/api
```

## Project Structure

```
internal/runnerkits/
├── dto.go         # request/response types
├── handler.go     # HTTP layer
├── repository.go  # sqlc-backed Repository + toResponse converter
└── service.go     # business logic + validation
```

## Endpoints

| Method | Path                                       | Auth     | Body                          | Returns          |
|--------|--------------------------------------------|----------|-------------------------------|------------------|
| POST   | `/api/v1/runner-kits`                      | JWT      | `CreateRunnerKitItemRequest`  | `GenericResponse`|
| GET    | `/api/v1/runner-kits/event/{event_id}`     | Public   | —                             | `GenericResponse`|
| GET    | `/api/v1/runner-kits/{id}`                 | JWT      | —                             | `GenericResponse`|
| PUT    | `/api/v1/runner-kits/{id}`                 | JWT      | `UpdateRunnerKitItemRequest`  | `GenericResponse`|
| DELETE | `/api/v1/runner-kits/{id}`                 | JWT      | —                             | `map[string]string`|

Public route is mounted in `cmd/api/main.go` at
`/api/v1/events/event/{event_id}/runner-kits` (line 174) — the handler's
`Routes()` registers under `/runner-kits` but the public endpoint goes
through `runnerKitHandler.ListByEvent` directly.

### Request shape

```json
{
  "event_id": "uuid",
  "name": "T-Shirt",
  "icon": "shirt"
}
```

`name` ≤ 100 chars; `icon` ≤ 50 chars, optional.

## Code Style

The handler is the canonical example of the codebase's preferred error
shape — every error body is JSON with `{"error": "..."}`:

```go
// internal/runnerkits/handler.go
http.Error(w, `{"error":"invalid runner kit item ID"}`, http.StatusBadRequest)
return
// ... or, more idiomatically:
json.WriteError(w, http.StatusBadRequest, "invalid runner kit item ID")
return
```

Note: the file actually mixes both styles — some endpoints use
`http.Error` with an inline JSON string (manually escaped via `sanitize`),
others use `json.WriteError`. Keep that in mind when migrating (see
Boundaries).

Response envelopes are a `GenericResponse{ Message, Data }`:

```go
type GenericResponse struct {
    Message string      `json:"message"`
    Data    interface{} `json:"data"`
}
```

The repository returns the raw sqlc `db.RunnerKitItem` and converts to
`RunnerKitItemResponse` via `toResponse` — `created_at` is formatted as a
`2006-01-02T15:04:05Z` string.

## Testing Strategy

- Add a public-read integration test to `internal/integration_test/
  integration_test.go` (one already exists: `TestRunnerKits_Public`).
- Add a JWT-protected create/update/delete test using the seeded
  organizer token.
- The `icon` field is free-text — do not over-test length; trust the
  validator tag.

## Boundaries

- **Always do:** Use `kinetix-api/internal/json` helpers — this module is the
  reference implementation of the codebase's preferred style.
- **Ask first:** Adding validation to the `icon` field (e.g. enum), changing
  the response envelope, renaming `event_id` to anything else (frontend
  contract).
- **Never do:**
  - Mix `http.Error` with a JSON-stringified body in new code — use
    `json.WriteError`.
  - Add a `ListAll` endpoint — always scope to an event.
  - Delete a kit item referenced by a registration. (No FK exists today;
    adding one is a schema change.)

## Success Criteria

- `GET /api/v1/events/event/{event_id}/runner-kits` is public and returns a
  list with `name`, `icon`, `id`, `event_id`, `created_at`.
- `POST /api/v1/runner-kits` (JWT) creates a new item, returns 201 with the
  record.
- `PUT /api/v1/runner-kits/{id}` updates `name` and `icon` (event_id is
  not editable here).
- `DELETE /api/v1/runner-kits/{id}` removes the item.
- All error responses are JSON with `{"error": "..."}`.

## Open Questions

- The `icon` field has no validation beyond length — should there be a
  restricted enum (e.g. `shirt`, `medal`, `bib`, `cap`)?
- The handler mixes `http.Error` (with manual JSON) and `json.WriteError`
  inside the same file. Plan to normalise as part of a broader sweep?
