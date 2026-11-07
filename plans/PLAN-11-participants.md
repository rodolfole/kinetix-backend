# PLAN-11: Participants coverage and handler consistency

- Status: Draft
- Created: 2026-09-25 | Updated: 2026-11-06
- Spec: docs/specs/SPEC-participants.md
- ADRs: ADR-6

## Goal
Participants CRUD and field validation are tested, and the handler's `http.Error` usage is either migrated to `internal/json` or explicitly documented.

## Scope
### In
- Integration CRUD test, validation unit tests, handler consistency decision.
### Out
- Soft delete (hard delete can break registrations; tracked as an open question).
- Email uniqueness constraint.
- New route for `UpdateTeamSize` (stays service-only).

## Design
- Keep handler/service/repository layering.
- Migrating `handler.go` to `internal/json` is optional here to keep the diff focused; T3 documents the gap, T4 does the migration (ADR-6).

## Tasks
- [ ] T1 Integration CRUD with an organizer token (list, get, update, delete) — test first: `TestParticipants_CRUDOrganizerToken` — files: `internal/integration_test/integration_test.go`
- [ ] T2 Boundary validation tests (missing email, country not 2 chars, invalid `size`, short phone, future `birth_date`) — test first: `internal/participants/service_test.go` — files: `internal/participants/service_test.go`
- [ ] T3 Add a comment at the top of `handler.go` pointing to `internal/json` as the canonical helper — test first: n/a (comment) — files: `internal/participants/handler.go`
- [ ] T4 (optional) Migrate the five handler methods to `json.Write`/`json.WriteError`/`json.Read` preserving status codes and messages — test first: T1 and T2 tests stay green, bad-payload POST returns the same status/message as before — files: `internal/participants/handler.go`

## Verification
```bash
go test -race -count=1 ./internal/participants/... ./internal/integration_test/...
```
Acceptance: responses unchanged by the migration; CRUD covered.

## Risks and rollback
- T4 may silently change error messages: diff response bodies before and after.

## Open questions
- Is hard delete intentional, or soft delete with `deleted_at`?
- Apply T4 in this plan or a separate consistency sweep (shared with PLAN-12, PLAN-13)?
