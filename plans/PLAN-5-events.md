# PLAN-5: Events sub-resource coverage

- Status: Draft
- Created: 2026-09-25 | Updated: 2026-11-06
- Spec: docs/specs/SPEC-events.md
- ADRs: ADR-13, ADR-1

## Goal
Write paths of awards, pricing stages, sponsors, locations and field config are covered by tests, and the known N+1 in event detail is documented.

## Scope
### In
- Integration tests for sub-resource writes, status patch and itinerary validation, `ponytail` comment on the N+1.
### Out
- Splitting `internal/events` (ADR-13).
- Moving `Distance` to `internal/distances`.
- Batching or caching `GetFullEventDetail` until measured.
- Role checks on event routes: PLAN-15 (B9).

## Design
- Sub-resources stay co-located so the full-detail composition is one round trip (ADR-13).
- The N+1 on awards/pricing stays; marker: `# ponytail: N+1 on awards/pricing, profile then batch-join`.

## Tasks
- [ ] T1 Integration test for award CRUD (`Place=1, Gender=male, PrizeAmount=5000`) — test first: `TestEvents_Awards_CRUD` — files: `internal/integration_test/integration_test.go`
- [ ] T2 Integration test for pricing stage CRUD, 400 when `end_date <= start_date` — test first: `TestEvents_Pricing_CRUD` — files: `internal/integration_test/integration_test.go`
- [ ] T3 Integration test for sponsors and locations CRUD — test first: `TestEvents_SponsorsLocations_CRUD` — files: `internal/integration_test/integration_test.go`
- [ ] T4 Integration test for field-config: unknown section returns 400 `invalid section name: ...`, valid sections round-trip — test first: `TestEvents_FieldConfig_Update` — files: `internal/integration_test/integration_test.go`
- [ ] T5 Integration test for status patch (`"foo"` 400, `"published"` 200) — test first: `TestEvents_StatusPatch_Invalid` — files: `internal/integration_test/integration_test.go`
- [ ] T6 Unit test: empty itinerary rejected on `Create` and `Update` — test first: `internal/events/service_test.go` with a recording fake repo — files: `internal/events/service_test.go`
- [ ] T7 Add the `# ponytail:` comment on the awards/pricing fan-out naming the upgrade path (`WHERE distance_id = ANY($1)`) — test first: n/a (comment) — files: `internal/events/repository.go`

## Verification
```bash
go test -race -count=1 ./internal/events/... ./internal/integration_test/...
grep -n ponytail internal/events/repository.go
```
Acceptance: all sub-resource write paths covered; seeded event detail unchanged.

## Risks and rollback
- Tests must create their own rows (`time.Now()` suffix) and delete them in `t.Cleanup`; T4 uses a dedicated event, never the seeded one.

## Open questions
- Should `Distance` move to `internal/distances` (two differently shaped types today)?
- Cache the detail composition for popular events (9+ queries per hit)?
