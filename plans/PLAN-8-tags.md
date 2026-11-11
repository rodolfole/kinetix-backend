# PLAN-8: Event tags coverage

- Status: Draft
- Created: 2026-09-25 | Updated: 2026-11-06
- Spec: docs/specs/SPEC-tags.md
- ADRs: ADR-1

## Goal
Tags have public-read and organizer CRUD tests, and the `(event_id, name)` uniqueness behavior is decided and recorded.

## Scope
### In
- Integration tests, uniqueness decision, optional typed DTOs.
### Out
- A `Service` layer, unless symmetry with `runnerkits`/`routes` is explicitly requested.

## Design
- Keep `DeleteByNames` in the repository even without a caller.
- Module lives in `internal/features/tags` (the only package outside the top-level layout).

## Tasks
- [ ] T1 Public read integration test for `/events/{eventId}/tags` (200, empty list allowed) — test first: `TestEventTags_Public` — files: `internal/integration_test/integration_test.go`
- [ ] T2 Organizer create/list/delete integration test — test first: `TestEventTags_OrganizerCRUD` — files: `internal/integration_test/integration_test.go`
- [ ] T3 Confirm `(event_id, name)` uniqueness in the schema; if missing, decide on a constraint and map duplicates to 409 — test first: integration test posting a duplicate (only if a constraint is added) — files: `docs/specs/SPEC-tags.md`, possibly a new migration and `internal/features/tags/handler.go`
- [ ] T4 (optional) Introduce a `Service` layer — test first: `go test ./internal/features/tags/...` stays green — files: `internal/features/tags/service.go`, `internal/features/tags/handler.go`
- [ ] T5 (optional) Replace `map[string]interface{}` with typed `TagResponse`/`TagListResponse` — test first: response-shape test — files: `internal/features/tags/dto.go`, `internal/features/tags/handler.go`

## Verification
```bash
go test -race -count=1 ./internal/features/tags/... ./internal/integration_test/... -run Tag
```
Acceptance: tag read and CRUD paths covered; uniqueness decision recorded in the spec.

## Risks and rollback
- T3 may reveal a missing constraint; agree on it before adding a migration (Down must work).

## Open questions
- Unique `(event_id, name)`: yes or no?
- Service layer and typed DTOs: required or optional?
