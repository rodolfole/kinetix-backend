# Spec: Routes Module

- Status: Implemented
- Plan: plans/PLAN-10-routes.md

## Objective

Accept GPX files from organizers and persist a parsed route (GeoJSON
LineString, elevation profile, total distance, gain, min/max elevation)
against an event and optional distance. Public listing by event/distance
for the event page; JWT-protected upload/delete.

## Tech Stack

- Go 1.22+, Chi v5 router
- PostgreSQL via pgx/v5 + sqlc (`pgtype.Numeric` for distances / elevation)
- `go-playground/validator/v10`
- `kinetix-api/internal/json` helpers (consistent style)
- `encoding/xml` for GPX parsing
- `math` for Haversine distance
- `kinetix-api/internal/routeparser` package owns the GPX → ParseResult
  pipeline

## Commands

```bash
go build ./...
go test ./internal/routes/... ./internal/routeparser/...
go run ./cmd/api
```

## Project Structure

```
internal/routes/
├── dto.go         # request/response types (RouteResponse, RouteListItem)
├── handler.go     # HTTP layer (multipart upload)
├── repository.go  # sqlc-backed Repository + toResponse/toListItem converters
└── service.go     # business logic, calls routeparser.ParseGPX

internal/routeparser/
└── gpx.go         # GPX XML decode, Haversine distance, GeoJSON + elevation emission
```

## Endpoints

| Method | Path                                                   | Auth   | Body                  | Returns          |
|--------|--------------------------------------------------------|--------|-----------------------|------------------|
| POST   | `/api/v1/routes/upload`                                | JWT    | multipart/form-data   | `GenericResponse`|
| GET    | `/api/v1/events/event/{event_id}/routes`               | Public | —                     | `GenericResponse`|
| GET    | `/api/v1/events/routes/distance/{distance_id}`         | Public | —                     | `GenericResponse`|
| GET    | `/api/v1/routes/{id}`                                  | JWT    | —                     | `GenericResponse`|
| DELETE | `/api/v1/routes/{id}`                                  | JWT    | —                     | `map[string]string`|

### Multipart form fields (`POST /routes/upload`)

| Field         | Required | Notes                                  |
|---------------|----------|----------------------------------------|
| `event_id`    | yes      | UUID string                            |
| `distance_id` | no       | UUID string; nullable link to a distance |
| `name`        | no       | Fallback to GPX track name; default `"Route"` |
| `file`        | yes      | GPX file, max 32 MB (enforced by `r.ParseMultipartForm(32 << 20)`) |

## GPX parsing pipeline

The parser lives in `internal/routeparser/gpx.go` and is the canonical
example of how to go from GPX bytes to a persisted route.

```go
// 1. Decode XML
var gpxData gpx
if err := xml.NewDecoder(r).Decode(&gpxData); err != nil { ... }

// 2. Reject empty tracks
if len(gpxData.Tracks) == 0 { ... }

// 3. Flatten track points across segments
var allPoints []trkpt
for _, seg := range track.Segments {
    allPoints = append(allPoints, seg.Points...)
}

// 4. Compute per-point cumulative distance (Haversine, R=6371km)
// 5. Compute total elevation gain (only ascending deltas)
// 6. Compute min/max elevation
// 7. Emit:
//      parsed.GeoJSONLineString()      // GeoJSON LineString JSON bytes
//      parsed.ElevationProfile()       // [[distance_km, elevation_m], ...] JSON bytes
```

`Haversine` lives in the same file as the parser — small enough that the
math co-locates with the consumer.

`ParseGPX` returns errors for: invalid XML, no tracks, no points in track.
The service surfaces these as 400.

### Response shape

`RouteResponse` carries GeoJSON, elevation profile, and derived metrics:

```json
{
  "id": "uuid",
  "event_id": "uuid",
  "distance_id": "uuid",
  "name": "10K Course",
  "geojson": { "type": "LineString", "coordinates": [[lon,lat], ...] },
  "elevation_profile": [[0.0, 12.3], [0.05, 12.4], ...],
  "total_distance_km": 10.02,
  "total_elevation_gain": 87.5,
  "min_elevation": 5.2,
  "max_elevation": 92.7,
  "created_at": "2026-01-01T00:00:00Z"
}
```

`RouteListItem` strips `geojson` and `elevation_profile` to keep list
endpoints light.

## Code Style

The service is the entry point — handler just hands the file through:

```go
// internal/routes/service.go
parsed, err := routeparser.ParseGPX(file)
name := req.Name
if name == "" && parsed.Name != "" { name = parsed.Name }
if name == "" { name = "Route" }

route, err := s.repo.Create(ctx, db.CreateRouteParams{
    EventID:            req.EventID,
    DistanceID:         distanceID,
    Name:               name,
    Geojson:            parsed.GeoJSONLineString(),
    ElevationProfile:   parsed.ElevationProfile(),
    TotalDistance:      convert.ToPgNumeric(parsed.TotalDistanceKm),
    TotalElevationGain: convert.ToPgNumeric(parsed.TotalElevationGain),
    MinElevation:       convert.ToPgNumeric(parsed.MinElevation),
    MaxElevation:       convert.ToPgNumeric(parsed.MaxElevation),
})
```

The converter handles nullable pgx numerics and `pgtype.UUID` defensively
— empty cells become `omitempty` pointers on the JSON response.

## Testing Strategy

- `TestEventRoutes_Public` already exists in `internal/integration_test/
  integration_test.go` — extend with a real fixture (upload a known GPX,
  list, fetch, delete).
- `internal/routeparser/gpx_test.go` should cover:
  - valid single-track GPX → expected total distance, elevation gain,
    min/max elevation
  - empty `<gpx>` (no tracks) → error
  - track with no points → error
  - multi-segment GPX → points are flattened in document order
- `internal/routes/service_test.go` should cover the `name` fallback
  chain: GPX name wins over empty, `Route` wins over empty GPX name.

## Boundaries

- **Always do:** Read the raw multipart body before parsing — keep the
  upload size limit (`32 << 20`) explicit.
- **Ask first:** Changing the Haversine constant, switching to a real GPX
  library, raising the upload size limit, storing GeoJSON as `geography`
  rather than `jsonb`.
- **Never do:**
  - Trust caller-supplied distances or elevation — always recompute from
    the GPX points.
  - Skip `r.ParseMultipartForm` — without it `r.FormFile` panics.
  - Pass the parsed result without re-emitting GeoJSON/elevation — the
    response shape expects raw bytes for both.

## Success Criteria

- `POST /api/v1/routes/upload` accepts a multipart form with a GPX file,
  parses it, persists the route, returns 201 with the full
  `RouteResponse`.
- `GET /api/v1/events/event/{event_id}/routes` is public and returns the
  list of routes for that event.
- `GET /api/v1/events/routes/distance/{distance_id}` is public and returns
  the routes for that distance.
- `GET /api/v1/routes/{id}` returns the full route including GeoJSON and
  elevation profile.
- `DELETE /api/v1/routes/{id}` removes the route.
- An invalid GPX returns 400 with a clear error; an empty GPX (no tracks)
  returns 400.

## Open Questions

- Should the upload size limit be configurable? Today it's a magic number
  `32 << 20`.
- Should we accept zipped GPX (`.gpx.gz`)?
- The `name` fallback chain has three cases — `req.Name` → `parsed.Name`
  → `"Route"`. Confirm the default of `"Route"` is intentional.
