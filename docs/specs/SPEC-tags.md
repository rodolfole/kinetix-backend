# Spec: Tags Module

- Status: Implemented
- Plan: plans/PLAN-8-tags.md

## Objective

Per-event free-text tags (`"flat"`, `"night race"`, `"kids friendly"`, …)
shown on event pages. Read access is public; create/delete is JWT-protected.

## Tech Stack

- Go 1.22+, Chi v5 router
- PostgreSQL via pgx/v5 + sqlc
- `kinetix-api/internal/json` helpers (consistent style)

## Commands

```bash
go build ./...
go test ./internal/features/tags/...
go run ./cmd/api
```

## Project Structure

```
internal/features/tags/
├── handler.go      # HTTP layer
└── repository.go   # sqlc-backed Repository + DeleteByNames helper
```

**No service layer** — this module is small enough that the handler calls
the repository directly. Worth flagging — see Open Questions.

## Endpoints

| Method | Path                                              | Auth   | Body                              | Returns        |
|--------|---------------------------------------------------|--------|-----------------------------------|----------------|
| GET    | `/api/v1/events/{eventId}/tags`                   | Public | —                                 | `{"message","data"}` |
| POST   | `/api/v1/events/tags`                             | JWT    | `{"event_id","name"}`             | `{"message","data"}` |
| DELETE | `/api/v1/events/tags/{id}`                        | JWT    | —                                 | `{"message"}`  |

### Request shape

```json
{
  "event_id": "uuid",
  "name": "flat"
}
```

`name` is required and non-empty.

## Code Style

The handler uses `kinetixjson.Write` / `kinetixjson.WriteError` with
explicit status codes — note the package is aliased:

```go
// internal/features/tags/handler.go
import (
    kinetixjson "kinetix-api/internal/json"
)
kinetixjson.Write(w, http.StatusCreated, map[string]interface{}{
    "message": "Tag created successfully",
    "data":    tag,
})
```

The response shape uses raw `map[string]interface{}` rather than a typed
DTO. That works but loses type safety — see Open Questions.

## Repository helpers

```go
// internal/features/tags/repository.go
func (r *Repository) ListByEvent(ctx context.Context, eventID uuid.UUID) ([]db.EventTag, error)
func (r *Repository) Create(ctx context.Context, eventID uuid.UUID, name string) (db.EventTag, error)
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error
func (r *Repository) DeleteByNames(ctx context.Context, eventID uuid.UUID, names []string) error
```

`DeleteByNames` is **not** wired to any handler — it's a bulk-cleanup
helper, available for future use.

## Testing Strategy

- Add a public-read integration test (currently absent).
- Add a JWT-protected create/delete test using the seeded organizer token.
- Unit-test the repository's `Create` and `Delete` paths with the sqlc
  store.

## Boundaries

- **Always do:** Validate `name` is non-empty at the handler boundary
  (already done).
- **Ask first:** Adding a unique constraint on `(event_id, name)`,
  introducing a service layer, switching response envelopes from
  `map[string]interface{}` to a typed DTO.
- **Never do:**
  - Trust the URL path `eventId` without parsing.
  - Allow empty `name`.
  - Add bulk-create or bulk-update routes without explicit need (YAGNI —
    the helper exists for batch cleanup, not for batch insertion).

## Success Criteria

- `GET /api/v1/events/{eventId}/tags` returns a list of tags for that event
  with no auth required.
- `POST /api/v1/events/tags` (JWT) creates a tag with the given `name`
  against the given `event_id`; 400 if `name` is empty or `event_id` is
  nil.
- `DELETE /api/v1/events/tags/{id}` (JWT) removes the tag.
- All responses are JSON; no leaked internal errors.

## Open Questions

- Should there be a `Service` layer? Currently the handler holds a
  `*Repository` directly — fine for a 3-endpoint module, but inconsistent
  with `runnerkits`, `routes`, etc.
- Should responses use a typed DTO instead of `map[string]interface{}`?
- Is duplicate `(event_id, name)` currently allowed? No unique constraint
  exists in the handler — worth confirming the schema.
