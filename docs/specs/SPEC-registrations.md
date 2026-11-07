# Spec: Registrations Module

- Status: Implemented
- Plan: plans/PLAN-12-registrations.md

## Objective

Joins a participant to an event with a chosen distance and category, and
issues a bib number plus a QR code used at check-in. Also exposes the
"create participant + registration together" flow that the public
registration form posts to.

## Tech Stack

- Go 1.22+, Chi v5 router
- PostgreSQL via pgx/v5 + sqlc-generated queries
- `go-playground/validator/v10` for struct validation
- Raw `http.Error` in handler — same inconsistency as `participants` (see
  Boundaries)

## Commands

```bash
go build ./...
go test ./internal/registrations/...
go run ./cmd/api
```

## Project Structure

```
internal/registrations/
├── handler.go      # HTTP layer
├── service.go      # business logic, dedup check, dynamic field validation
├── registration.go # sqlc-backed Repository + bib/QR generation
└── model.go        # domain types, request/response DTOs
```

Mounted under `/api/v1/registrations` (protected by JWT).

## Endpoints

| Method | Path                                              | Body                          | Returns                       |
|--------|---------------------------------------------------|-------------------------------|-------------------------------|
| POST   | `/api/v1/registrations`                           | `CreateRegistrationRequest`   | `RegistrationResponse`        |
| POST   | `/api/v1/registrations/with-participant`          | `CreateWithParticipantRequest`| `RegistrationResponse`        |
| GET    | `/api/v1/registrations?event_id=…`                | —                             | `RegistrationsListResponse`   |
| GET    | `/api/v1/registrations/participant/{participant_id}` | —                          | `RegistrationsListResponse`   |
| GET    | `/api/v1/registrations/{id}`                      | —                             | `RegistrationResponse`        |
| DELETE | `/api/v1/registrations/{id}`                      | —                             | `GenericResponse`             |

`POST /with-participant` accepts either an existing `participant_id` (in
which case only team/size are updated on the existing participant) or a
fresh `participant` object (in which case the participant is created in the
same transaction).

### `CreateRegistrationRequest`

```json
{
  "event_id": "uuid",
  "participant_id": "uuid",
  "distance_id": "uuid",
  "category": "M-30",
  "time": "00:45:30"
}
```

### `CreateWithParticipantRequest`

```json
{
  "event_id": "uuid",
  "participant_id": "uuid (optional)",
  "participant": { "first_name": "...", ... },
  "registration": {
    "distance_id": "uuid",
    "category": "M-30",
    "time": "00:45:30",
    "team": "Runners Club",
    "size": "M"
  }
}
```

## Code Style

The service layer is the interesting part — it reads the per-event field
config (`event_participant_fields`) to decide which participant fields are
required for this event, then validates before persisting.

```go
// internal/registrations/service.go
requiredFields := make(map[string]bool)
for _, f := range fields {
    if f.IsRequired {
        requiredFields[f.FieldName] = true
    }
}
requiredFields["address"] = true   // always required
```

The repository generates a 4-digit zero-padded bib number per event by
querying the current max:

```go
nextNum := maxNum + 1
return fmt.Sprintf("%04d", nextNum), nil
```

QR codes are JSON strings embedding `reg_id`, `event_id`, `p_id` for
check-in scanning.

## Testing Strategy

- `internal/integration_test/integration_test.go` is the baseline — it does
  not currently exercise registrations; add coverage here.
- Test cases worth adding: dedup (same participant + event rejected),
  bib-number progression, dynamic-field requirement toggling.

## Boundaries

- **Always do:** Check for an existing `(event_id, participant_id)` pair
  before inserting (`GetByEventParticipant`).
- **Ask first:** Changing the bib-number format (currently `0001`), changing
  QR code schema (currently embedded JSON), adding new optional participant
  fields that the registration flow should honour.
- **Never do:**
  - Skip the dedup check — it is the only safeguard against duplicate
    registrations.
  - Trust caller-supplied bib numbers; the repository owns them.
  - Migrate to `internal/json` helpers without scoping the change
    explicitly — the `participants` package has the same inconsistency and
    they should be addressed together.

## Success Criteria

- `POST /api/v1/registrations` returns the created record with a fresh
  `bib_number` (e.g. `0007`) and a populated `qr_code`.
- Submitting a duplicate `(event_id, participant_id)` returns an error
  (handler currently surfaces it as a 500 — acceptable for now, see Open
  Questions).
- `GET /api/v1/registrations?event_id=…` returns only registrations for that
  event.
- `GET /api/v1/registrations/participant/{participant_id}` returns
  registrations for that participant.
- `POST /with-participant` with a `participant_id` updates team/size and
  creates a registration in one call.
- `POST /with-participant` with a full `participant` object creates both
  records and respects the per-event field-config required flags.

## Open Questions

- The dedup error becomes a 500 instead of a 409 because the handler uses
  `http.StatusInternalServerError` for any non-validation error. Should this
  be mapped to 409 Conflict?
- Should `with-participant` be transactional (both inserts in one tx)? Today
  the participant is created, then the registration — a failure between
  them leaves an orphan participant.
- `participant.address` is required for `with-participant` but lives on the
  participant table; is this the intended model?
