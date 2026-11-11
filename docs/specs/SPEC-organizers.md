# Spec: Organizers Module

- Status: Implemented
- Plan: plans/PLAN-4-organizers.md

## Objective

CRUD API for the organizer entity — the legal/business owner of an event.
Includes a public read-by-id endpoint so event listings can render the
organizer's name without exposing edit operations.

Organizers are written two ways:
1. **Inline at registration** — `auth.RegisterOrganizer` creates the row plus
   a `users` row in a single transaction.
2. **Direct CRUD** — this module's `Create` is used when an admin or import
   job needs to seed an organizer without a user account. It enforces RFC
   uniqueness and standard validation.

Mounted at `/api/v1/organizers/*`. `GET /{id}` is public; everything else is
behind `JWTAuth`.

## Tech Stack

- Go 1.22+, Chi v5 router
- `go-playground/validator/v10` for request validation
- sqlc-generated queries via `kinetix-api/internal/store`
- `kinetix-api/internal/json` helpers for response shaping

## Commands

```bash
go build ./...
go test ./internal/organizers/...
go test ./internal/integration_test/...        # TestOrganizers_ProtectedList, TestPublicOrganizerDetail
```

## Project Structure

```
internal/organizers/
├── handler.go      # HTTP layer
├── service.go      # business logic + validation
├── repository.go   # sqlc-backed CRUD + GetByRFC
└── model.go        # DTOs + alias to sqlc.Organizer
```

## Endpoints

| Method | Path                       | Body                       | Returns                       | Auth |
|--------|----------------------------|----------------------------|-------------------------------|------|
| POST   | `/api/v1/organizers`       | `CreateOrganizerRequest`   | `OrganizerResponse` (201)     | JWT  |
| GET    | `/api/v1/organizers`       | — (`?limit=&offset=`)      | `OrganizersListResponse`      | JWT  |
| GET    | `/api/v1/organizers/{id}`  | —                          | `OrganizerResponse` (200)     | none |
| PUT    | `/api/v1/organizers/{id}`  | `UpdateOrganizerRequest`   | `OrganizerResponse` (200)     | JWT  |
| DELETE | `/api/v1/organizers/{id}`  | —                          | `{"message": "..."}` (200)    | JWT  |

`CreateOrganizerRequest` validates: `BusinessName` (required,min=2,max=255),
`RFC` (required,len=12|len=13 — Mexican tax ID, unique), `BrandName`
(omitempty), billing zip/state/city (required), contact name/email/phone,
optional `LogoUrl` (omitempty,url,max=500).

`UpdateOrganizerRequest` mirrors `CreateOrganizerRequest` minus the `RFC`
field (RFC is immutable after creation).

### Example

```json
POST /api/v1/organizers
{
  "business_name": "Acme Corp",
  "brand_name":    "Acme",
  "rfc":           "ABC010101ABC",
  "billing_zip_code": "44100",
  "billing_state":   "Jalisco",
  "billing_city":    "Guadalajara",
  "contact_name":    "John Doe",
  "contact_email":   "owner@example.com",
  "contact_phone":   "+5255555555",
  "logo_url":        "https://cdn.example.com/logo.png"
}

201 Created
{ "message": "Organizer created successfully", "data": { "id": "...", "rfc": "...", ... } }
```

## Code Style

```go
// internal/organizers/handler.go
func (h *Handler) CreateOrganizer(w http.ResponseWriter, r *http.Request) {
    var req CreateOrganizerRequest
    if err := json.Read(r, &req); err != nil {
        http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
        return
    }
    org, err := h.service.Create(r.Context(), req)
    if err != nil {
        json.WriteError(w, http.StatusBadRequest, err.Error())
        return
    }
    json.Write(w, http.StatusCreated, OrganizerResponse{
        Message: "Organizer created successfully",
        Data:    org,
    })
}
```

The handler uses `http.Error(w, "...", code)` for the read/parse path but
`json.WriteError` for service errors so that DB error messages containing
quotes do not break JSON. `service.go` validates first, then checks RFC
uniqueness, then calls `repo.Create`.

## Testing Strategy

- Repository: integration tests only (sqlc calls require the live store).
- Service: unit tests via `internal/organizers/service_test.go` — fake
  repository that records the calls and returns canned RFC-collision
  errors.
- Integration: `TestOrganizers_ProtectedList` (auth-required list) and
  `TestPublicOrganizerDetail` (no-auth GET by id) live in
  `internal/integration_test/integration_test.go`. No create/update/delete
  integration coverage yet — see PLAN.

## Boundaries

- **Always do:** Validate at the struct tag level; convert nullable columns
  through `kinetix-api/internal/database/convert`; treat RFC as immutable on
  update.
- **Ask first:** Schema changes to `organizers` (new column, new index),
  changing the RFC uniqueness invariant, allowing soft-delete.
- **Never do:**
  - Return a raw DB error string to the client without first wrapping it in
    `json.WriteError` (handler inconsistency noted in `SPEC-participants.md`
    applies here too — organizers still uses `http.Error` for read failures).
  - Allow `RFC` to change on `Update`.
  - Delete an organizer whose events are still referenced — currently
    `ON DELETE` cascades; see Open Questions.

## Success Criteria

- `POST /organizers` with a unique RFC returns 201 with the created record.
- `POST /organizers` with a duplicate RFC returns 400 with the message
  `"organizer with RFC %s already exists"`.
- `GET /organizers/{id}` is reachable without a token and returns 200.
- `GET /organizers` without a token returns 401.
- `PUT /organizers/{id}` with the same fields but a different RFC in the
  body (since RFC is excluded from the update DTO) succeeds.
- `DELETE /organizers/{id}` returns 200 and the row is gone on a follow-up
  GET.
- `go test ./internal/organizers/... ./internal/integration_test/...` passes.

## Open Questions

- Soft-delete vs hard-delete: today's cascade breaks any events owned by
  the deleted organizer. Should we add `deleted_at` and block the delete
  if active events exist?
- Public GET currently exposes the full `Organizer` row including billing
  fields. Should we project only the public-safe fields (business_name,
  brand_name, logo_url, contact_email)?
