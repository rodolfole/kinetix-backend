# Spec: Distances Module

- Status: Implemented
- Plan: plans/PLAN-6-distances.md

## Objective

Two things:
1. **Distance type catalog** — the closed list of race distances (5K, 10K,
   half marathon, etc.) seeded into `distance_types`. Public read-only.
2. **Distance CRUD** — concrete race distances attached to an event, each
   referencing a `distance_type`. Capacity, surface, optional elevation
   gain, time limit, and start/end coordinates. Protected.

Mounted at `/api/v1/distances/*`. `GET /types` is public; everything else
is behind `JWTAuth`.

## Tech Stack

- Go 1.22+, Chi v5 router
- sqlc-generated queries via `kinetix-api/internal/store`
- `kinetix-api/internal/json` helpers
- `kinetix-api/internal/database/convert` for pgx types (`pgtype.Numeric`,
  `pgtype.UUID`)

## Commands

```bash
go build ./...
go test ./internal/distances/...
go test ./internal/integration_test/...        # TestListDistanceTypes_Public, TestEventDistances_Public
```

## Project Structure

```
internal/distances/
├── handler.go      # HTTP layer (public /types, protected CRUD)
├── service.go      # business logic + validation; ListByEventWithTypes
├── repository.go   # sqlc CRUD + toDistanceResponse converter
└── dto.go          # request/response DTOs (Capacity validated as `gt=0`)
```

## Endpoints

| Method | Path                                       | Auth | Notes                                |
|--------|--------------------------------------------|------|--------------------------------------|
| GET    | `/api/v1/distances/types`                  | none | Distance type catalog                |
| GET    | `/api/v1/distances/event/{event_id}`       | JWT  | Distances enriched with type info    |
| POST   | `/api/v1/distances`                        | JWT  | `DistanceTypeID` must exist          |
| GET    | `/api/v1/distances/{id}`                   | JWT  |                                      |
| PUT    | `/api/v1/distances/{id}`                   | JWT  |                                      |
| DELETE | `/api/v1/distances/{id}`                   | JWT  |                                      |

Note: `/api/v1/events/event/{event_id}/distances` is also public (registered
in `cmd/api/main.go`) — it is the same `ListByEvent` handler.

### Request shapes

```json
POST /api/v1/distances
{
  "event_id":         "...",
  "distance_type_id": "...",
  "capacity":         500,
  "surface":          "asphalt",
  "elevation":        120,
  "time_limit":       "01:30:00",
  "start_lat": 19.4326,
  "start_lng": -99.1332,
  "end_lat":   19.4400,
  "end_lng":  -99.1400
}
```

Validation: `Capacity` (required,gt=0), `Surface` (required,max=50),
`EventID`/`DistanceTypeID` (required UUIDs), coordinates optional.

`UpdateDistanceRequest` drops `EventID` (the distance belongs to one event
for its lifetime) and keeps the rest.

### Response shape (single distance)

`DistanceResponse` carries `Km` populated from the linked `DistanceType`
when fetched via `ListByEventWithTypes`; otherwise from the row itself
(`Km` defaults to 0 at insert time and is overwritten by the join).

## Code Style

```go
// internal/distances/repository.go
func toDistanceResponse(d db.Distance, dt *db.DistanceType) DistanceResponse {
    km := convert.PgNumericToFloat64(d.Km)
    if dt != nil {
        km = convert.PgNumericToFloat64(dt.Km)
    }
    resp := DistanceResponse{
        ID:        d.ID,
        EventID:   d.EventID,
        Km:        km,
        Capacity:  int(d.Capacity),
        ...
    }
    if d.DistanceTypeID.Valid {
        resp.DistanceTypeID = d.DistanceTypeID.Bytes
    }
    return resp
}
```

The `Km` join happens in `ListByEventWithTypes`, not in the basic CRUD
handlers. Single-row `GetByID` returns `Km` as stored.

## Testing Strategy

- `internal/integration_test/integration_test.go`:
  - `TestListDistanceTypes_Public` — public GET returns at least one type.
  - `TestEventDistances_Public` — distances for the seeded event have at
    least one row.
- Service / repository unit tests are **not** present. See PLAN.

## Boundaries

- **Always do:** Validate `Capacity > 0`; verify `DistanceTypeID` exists
  before creating/updating (in `service.go`); keep `EventID` immutable
  after creation.
- **Ask first:** Adding new columns to `distances`; letting the
  `distance_types` catalog be edited at runtime; supporting per-distance
  pricing stage windows.
- **Never do:**
  - Return raw DB error strings from the handler.
  - Allow `Km` to be set in the request body — `Km` always comes from
    the linked `distance_types` row.

## Success Criteria

- `GET /distances/types` without auth returns 200 with a non-empty list
  from the seed.
- `POST /distances` without auth returns 401.
- `POST /distances` with an unknown `distance_type_id` returns 400 with
  `distance type not found: ...`.
- `POST /distances` with `capacity: 0` returns 400 (validator).
- `GET /distances/event/{id}` returns the enriched list including `Km`
  from the joined type.
- `DELETE /distances/{id}` removes the row and a follow-up GET returns
  404.

## Open Questions

- Should `Distance.Km` be removed from the insert payload entirely, and
  be a derived field always? Today `repository.Create` passes
  `convert.ToPgNumeric(0)` because the row has its own Km column that is
  overwritten by the join. This is confusing.
- Should `time_limit` move from `string` to `pgtype.Interval`? Today it is
  parsed by callers as an opaque string.
