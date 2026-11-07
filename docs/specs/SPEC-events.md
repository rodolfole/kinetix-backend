# Spec: Events Module

- Status: Implemented
- Plan: plans/PLAN-5-events.md

## Objective

Full CRUD + the "event detail" composition (everything needed to render a
public event page in one shot) plus four nested sub-resources scoped to the
event:

- **Awards** — per-distance prize tiers (1st/2nd/3rd place per age category).
- **Pricing stages** — early-bird / regular / late windows per distance.
- **Sponsors** — sponsor logos grouped by tier.
- **Registration locations** — physical pickup points with schedules.

Plus two config surfaces:
- **Participant field config** — toggle which fields the registration form
  collects per event (`country`, `city`, `state`, `zip_code`, `medical_info`,
  `waiver_signed_at`, `emergency_contact`).
- **Event field config** — toggle which sections the event detail page
  renders (`categories`, `sponsors`, `registrations`, `judges`, `rules`,
  `risks`, `transit`).

Itinerary is a JSONB array of `{key, time, icon}` items; required at create
and update time.

## Tech Stack

- Go 1.22+, Chi v5 router
- sqlc-generated queries via `kinetix-api/internal/store`
- `kinetix-api/internal/json` helpers
- `kinetix-api/internal/database/convert` for pgtype conversions

## Commands

```bash
go build ./...
go test ./internal/events/...
go test ./internal/integration_test/...        # covers public read paths
```

## Project Structure

```
internal/events/
├── handler.go      # 25 routes — events + 4 sub-resources + 2 configs
├── service.go      # business logic + validation, including the
│                   # `validFields`/`validSections` allow-lists
├── repository.go   # sqlc CRUD + GetFullEventDetail composition
└── model.go        # 14 DTOs (Event, Distance, Award, PricingStage, ...)
```

Routes are mounted individually in `cmd/api/main.go` — `events.Handler.Routes`
is **not** used; the public/protected split is enforced by the main.go
wiring so `GET /events` stays public while write paths go behind `JWTAuth`.

## Endpoints — event core

| Method | Path                                       | Auth | Notes                                              |
|--------|--------------------------------------------|------|----------------------------------------------------|
| POST   | `/api/v1/events`                           | JWT  | Creates with status `draft`                        |
| GET    | `/api/v1/events`                           | none | `?status=&limit=&offset=`                          |
| GET    | `/api/v1/events/{id}`                      | none | Single event                                       |
| GET    | `/api/v1/events/{id}/detail`               | none | Composes event + distances + awards + ...          |
| GET    | `/api/v1/events/slug/{slug}`               | none |                                                    |
| PUT    | `/api/v1/events/{id}`                      | JWT  |                                                    |
| PATCH  | `/api/v1/events/{id}/status`               | JWT  | One of `draft`, `published`, `cancelled`, `completed` |
| PUT    | `/api/v1/events/{id}/participant-fields`   | JWT  |                                                    |
| GET    | `/api/v1/events/{id}/field-config`         | none |                                                    |
| PUT    | `/api/v1/events/{id}/field-config`         | JWT  | Overwrites the entire config                       |
| DELETE | `/api/v1/events/{id}`                      | JWT  |                                                    |

## Endpoints — awards (per distance)

| Method | Path                                              | Auth |
|--------|---------------------------------------------------|------|
| POST   | `/api/v1/events/awards`                           | JWT  |
| GET    | `/api/v1/events/awards/{id}`                      | JWT  |
| GET    | `/api/v1/events/distance/{distance_id}/awards`    | none |
| PUT    | `/api/v1/events/awards/{id}`                      | JWT  |
| DELETE | `/api/v1/events/awards/{id}`                      | JWT  |

## Endpoints — pricing stages (per distance)

| Method | Path                                                       | Auth |
|--------|------------------------------------------------------------|------|
| POST   | `/api/v1/events/pricing`                                   | JWT  |
| GET    | `/api/v1/events/pricing/{id}`                              | JWT  |
| GET    | `/api/v1/events/distance/{distance_id}/pricing`            | none |
| GET    | `/api/v1/events/distance/{distance_id}/pricing/current`   | none |
| PUT    | `/api/v1/events/pricing/{id}`                              | JWT  |
| DELETE | `/api/v1/events/pricing/{id}`                              | JWT  |

## Endpoints — sponsors (per event)

| Method | Path                                                    | Auth |
|--------|---------------------------------------------------------|------|
| POST   | `/api/v1/events/sponsors`                               | JWT  |
| GET    | `/api/v1/events/sponsors/{id}`                          | JWT  |
| GET    | `/api/v1/events/event/{event_id}/sponsors`              | none |
| PUT    | `/api/v1/events/sponsors/{id}`                          | JWT  |
| DELETE | `/api/v1/events/sponsors/{id}`                          | JWT  |

## Endpoints — registration locations (per event)

| Method | Path                                                       | Auth |
|--------|------------------------------------------------------------|------|
| POST   | `/api/v1/events/locations`                                 | JWT  |
| GET    | `/api/v1/events/locations/{id}`                            | JWT  |
| GET    | `/api/v1/events/event/{event_id}/locations`                | none |
| PUT    | `/api/v1/events/locations/{id}`                            | JWT  |
| DELETE | `/api/v1/events/locations/{id}`                            | JWT  |

### Example — create

```json
POST /api/v1/events
{
  "organizer_id": "...",
  "name": "Test Event",
  "slug": "test-event-20260101",
  "description": "Created by integration tests",
  "event_date": "2026-12-01T08:00:00Z",
  "deadline":   "2026-11-15T23:59:59Z",
  "state": "Test State",
  "municipality": "Test City",
  "address": "123 Test Street",
  "itinerary": [{"key": "Salida", "time": "07:00", "icon": "flag"}]
}
```

Validation: `Name` (min=2,max=255), `Slug` (min=2,max=255), `Address`
(min=5,max=500), `Itinerary` (min=1), plus the standard
`go-playground/validator` tags.

### Example — full detail composition

`GET /api/v1/events/{id}/detail` returns:

```json
{
  "data": { "id": "...", "name": "...", "...": "..." },
  "distances": [ ... ],
  "awards":    [ ... ],
  "pricing_stages": [ ... ],
  "sponsors":  [ ... ],
  "registration_locations": [ ... ],
  "tags":      ["..."],
  "participant_fields_config": [...],
  "field_config": [...],
  "runner_kits": [ ... ]
}
```

## Code Style

```go
// internal/events/handler.go
func (h *Handler) CreateEvent(w http.ResponseWriter, r *http.Request) {
    var req CreateEventRequest
    if err := json.Read(r, &req); err != nil {
        json.WriteError(w, http.StatusBadRequest, "invalid request body")
        return
    }
    evt, err := h.service.Create(r.Context(), req)
    if err != nil {
        // Use json.WriteError so the error string is properly escaped.
        // Previously we did `http.Error(w, `{"error":"`+err.Error()+`"}`, ...)`
        // which broke JSON parsing on the client when err.Error() contained
        // quotes (e.g. Postgres "column does not exist" messages with quotes).
        json.WriteError(w, http.StatusBadRequest, err.Error())
        return
    }
    json.Write(w, http.StatusCreated, EventResponse{
        Message: "Event created successfully",
        Data:    evt,
    })
}
```

The inline comment in `CreateEvent` documents the JSON-escaping bug fixed
in-place. Use the same shape across all sub-resources.

## Testing Strategy

- `internal/integration_test/integration_test.go` covers the public read
  paths (`TestListEvents_Public`, `TestListEvents_SlugLookup`,
  `TestListEvents_SlugNotFound`, `TestEventDetail_OrganizerCanFetch`,
  `TestEventDistances_Public`, `TestPricingStages_Public`,
  `TestCurrentPricing_Public`, `TestEventSponsors_Public`,
  `TestEventLocations_Public`, `TestEventFieldConfig_Public`).
- `TestEvents_CreateRequiresAuth`, `TestEvents_CreateWithOrganizerToken`,
  `TestEvents_CreateInvalidPayload` cover the create path.
- Write coverage for awards / sponsors / locations / pricing is **not** yet
  in the integration suite — see PLAN.

## Boundaries

- **Always do:** Validate `Itinerary` length ≥ 1 in the service (the
  validator tag enforces `min=1,dive`, but the service has a belt-and-
  braces check); use `convert.ToPgJSONBFromBytes` for the itinerary column;
  allow-list section/field names against the maps in `service.go`.
- **Ask first:** Adding a new event status; widening the field/section
  allow-lists; changing itinerary shape.
- **Never do:**
  - Soft-delete events; today `DELETE` is hard.
  - Bypass the `IsEarlyBird`/`Place` defaults (1st place default when
    `place == 0`).
  - Allow pricing stages with `end_date <= start_date` (service checks).

## Success Criteria

- `POST /events` with a valid payload returns 201 with status `draft`.
- `POST /events` with empty `itinerary` returns 400.
- `PATCH /events/{id}/status` with an unknown status returns 400.
- `GET /events/{id}/detail` returns 200 with all 10 composed fields
  populated for a fully-configured seeded event.
- `PUT /events/{id}/field-config` with a section not in the allow-list
  returns 400 with `invalid section name: ...`.
- `GET /distances/types`, `GET /distances/event/{id}/distances`, and the
  four public sub-resource GETs all return 200 for seeded data.
- All public GETs return 200 without a token; all write paths return 401
  without a token.

## Open Questions

- `Distance` lives in `internal/events/model.go` and is owned by
  `internal/distances` — should `Distance` be moved to the distances
  package and imported by events?
- Should `PATCH /events/{id}/status` emit a domain event so registration
  flows can react to a `cancelled` event?
