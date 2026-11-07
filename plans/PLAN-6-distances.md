# PLAN-6: Distances coverage

- Status: Draft
- Created: 2026-09-25 | Updated: 2026-11-06
- Spec: docs/specs/SPEC-distances.md
- ADRs: ADR-13

## Goal
Distance write paths, the `km` enrichment and the distance-type existence check are tested; the `Km` sourcing rule is documented.

## Scope
### In
- Integration CRUD and `ListByEvent` tests, service unit test, doc comment.
### Out
- Moving `events.Distance` into this package (ADR-13).
- Dropping the `km` column or typing `time_limit` (schema changes).

## Design
- Keep the `Km` join behavior: `toDistanceResponse` prefers the joined type when present, else the row value.

## Tasks
- [ ] T1 Assert type catalog entries have `id`, `name`, `km > 0` — test first: extend `TestListDistanceTypes_Public` — files: `internal/integration_test/integration_test.go`
- [ ] T2 Integration CRUD (create `capacity=42`, get, update to 99, delete; unknown `distance_type_id` returns 400) — test first: `TestDistances_CRUD` — files: `internal/integration_test/integration_test.go`
- [ ] T3 Assert `/distances/event/{id}` rows carry `km` equal to the catalog — test first: `TestDistances_ListByEvent_Km` — files: `internal/integration_test/integration_test.go`
- [ ] T4 Service unit test for the `distance_type_id` existence check (not found wraps error; found succeeds) — test first: `internal/distances/service_test.go` — files: `internal/distances/service_test.go`
- [ ] T5 Document the `Km` sourcing rule on `toDistanceResponse` — test first: n/a (comment) — files: `internal/distances/repository.go`

## Verification
```bash
go test -race -count=1 ./internal/distances/... ./internal/integration_test/...
```
Acceptance: seeded event's distance list unchanged.

## Risks and rollback
- Created distances must be deleted in `t.Cleanup`; resolve the type id at runtime via `/distances/types`.

## Open questions
- Remove `distances.km` and always derive from `distance_types`?
- Typed interval for `time_limit`?
