# PLAN-10: Routes and GPX upload hardening

- Status: In progress
- Created: 2026-09-25 | Updated: 2026-11-06
- Spec: docs/specs/SPEC-routes.md
- ADRs: ADR-10

## Goal
GPX parsing, route name fallback and the full upload flow are tested; upload limits are an explicit decision.

## Scope
### In
- Parser unit tests, service name-fallback tests, upload integration test.
### Out
- Splitting `routeparser` math from parsing.
- `.gpx.gz` support (YAGNI until requested).

## Design
- Parsing stays server-side in `internal/routeparser` (ADR-10).
- Default upload limit stays `32 << 20` (hardcoded in `internal/routes/handler.go`) unless made configurable.

## Tasks
- [x] T1 GPX parser unit tests (valid file, no tracks, no points, invalid XML, multi-segment flattening, GeoJSON lon/lat order, empty elevation profile, haversine) — test first: `internal/routeparser/gpx_test.go` — files: `internal/routeparser/gpx_test.go`
- [ ] T2 Service tests for the name fallback (`req.Name`, then GPX track name, then `"Route"`) — test first: `internal/routes/service_test.go` — files: `internal/routes/service_test.go`
- [ ] T3 Full upload integration test (upload multipart GPX, list by event, fetch with GeoJSON, delete) — test first: `TestRoutes_UploadListFetchDelete` — files: `internal/integration_test/integration_test.go`
- [ ] T4 (optional) Make the upload limit configurable (`Config.Upload.MaxBytes`, default `32 << 20`) — test first: handler test with a 16-byte limit rejects a real GPX — files: `internal/config/config.go`, `cmd/api/main.go`, `internal/routes/handler.go`
- [ ] T5 (optional) Accept `.gpx.gz` uploads via `gzip.NewReader` — test first: compressed fixture parses — files: `internal/routes/handler.go`

## Verification
```bash
go test -race -count=1 ./internal/routeparser/... ./internal/routes/... ./internal/integration_test/... -run 'Route|GPX|Haversine|GeoJSON|Elevation'
```
Acceptance: parser and service covered; upload round trip works against the seeded stack.

## Risks and rollback
- T3 races on shared seed data: use a fresh event slug per run.
- T4: mirror an existing config field name and type.

## Open questions
- Configurable upload limit and gzip support: wanted?
- Keep default name `"Route"` or reject empty input?
