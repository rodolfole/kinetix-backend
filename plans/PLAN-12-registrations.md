# PLAN-12: Registrations coverage and transactional safety

- Status: Draft
- Created: 2026-09-25 | Updated: 2026-11-06
- Spec: docs/specs/SPEC-registrations.md
- ADRs: ADR-14

## Goal
Registrations have end-to-end coverage, duplicates map to a clear status, and `with-participant` never leaves an orphan participant.

## Scope
### In
- Integration flow, bib-number test, duplicate error mapping, transaction around `with-participant`.
### Out
- Migrating the handler to `internal/json` (handled with PLAN-11 T4 if approved).

## Design
- Bib-number generation stays in the repository (it owns the DB state and format).
- `with-participant` orchestration stays in the service; it needs a transaction handle from the store (ADR-14, `store` transactions via `ExecTx`).

## Tasks
- [ ] T1 Integration create/list/get/delete using a seeded participant — test first: `TestRegistrations_CreateListGetDelete` — files: `internal/integration_test/integration_test.go`
- [ ] T2 Bib-number progression: two creates for one event yield consecutive bibs — test first: `internal/registrations/registration_test.go` (`BibNumber`) — files: `internal/registrations/registration_test.go`
- [ ] T3 Map duplicate `(event_id, participant_id)` to `ErrAlreadyRegistered` and a 409 — test first: unit test for the 409 mapping plus integration duplicate test — files: `internal/registrations/service.go`, `internal/registrations/handler.go`
- [ ] T4 Wrap the no-`participant_id` branch of `with-participant` in a DB transaction — test first: unit test failing between participant insert and registration insert leaves no participant — files: `internal/registrations/service.go`

## Verification
```bash
go test -race -count=1 ./internal/registrations/... ./internal/integration_test/...
```
Acceptance: duplicate returns 409; failure mid-flow leaves no orphan row.

## Risks and rollback
- T4 changes commit boundaries: only wrap the creation branch; leave the supplied-`participant_id` path untouched.
- T3 breaks clients expecting 500 on duplicates: note in release notes.
- If the store exposes no usable transaction handle, park T4 (see ADR-14).

## Open questions
- Duplicate as 409 or keep 500?
- Does `store.Store` expose a transaction usable from the service layer?
- Is `address` the right participant field for the address?
