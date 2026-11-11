# ADR-10: GPX route parsing on the server

- Status: Accepted
- Date: 2026-09-15
- Deciders: Kinetix team
- Related: ADR-3, ADR-6, docs/API_CONTRACTS.md, docs/DATABASE.md

## Context
Organizers attach a course to a distance by uploading a GPX file. Runners need a map line, total distance and an elevation profile without downloading the raw file. The package `internal/routeparser` (`gpx.go`) parses GPX; `internal/routes` stores the result in the `routes` table (migration 018, `geojson JSONB`, `elevation_profile JSONB`).

## Decision
- `POST /routes/upload` accepts `multipart/form-data` (`file`, `event_id`, optional `distance_id`, `name`), with a 32 MB form limit (`internal/routes/handler.go`).
- `routeparser.ParseGPX` decodes the first `<trk>` with `encoding/xml` (stdlib, no dependency), joins all segments, and computes cumulative distance with the haversine formula (Earth radius 6371 km), elevation gain, min and max elevation.
- The service persists derived values: GeoJSON `LineString` (`GeoJSONLineString`), `ElevationProfile`, distance, gain, min/max elevation. The original GPX file is not stored.
- Read endpoints return the stored GeoJSON and profile; list views omit GeoJSON (`RouteListItem`).

## Consequences
### Positive
- No new dependency; parsing is pure and unit tested (`internal/routeparser/gpx_test.go`).
- Clients get ready-to-render GeoJSON and do not need to parse GPX again to display a stored route.

### Negative / risks
- Elevation gain sums raw point-to-point increases with no smoothing, so GPS noise inflates it. Only the first track is used.
- `ParseMultipartForm(32 << 20)` limits memory, not total upload size; there is no `http.MaxBytesReader`, no `Content-Type` check and no cap on point count.
- `UploadGPX` returns every service error as `400` with `err.Error()` (`internal/routes/handler.go`), including database failures.
- The original GPX is discarded, so re-processing with a better algorithm needs a re-upload.
- Parsing happens in the request; very large files hold a handler goroutine up to the 60s `Timeout`.
- The frontend also parses GPX for previews, so parsing logic exists twice (frontend ADR-7).

## Alternatives considered
- Parse only in the browser and store GeoJSON sent by the client: rejected, the server must not trust client-computed distance and elevation.
- Third-party GPX library: rejected, the subset needed is small.
- Store the raw GPX in object storage and compute on read: rejected, repeats work per request and adds infrastructure.
