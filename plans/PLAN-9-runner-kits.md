# PLAN-9: Runner kits consistency and coverage

- Status: Draft
- Created: 2026-09-25 | Updated: 2026-11-06
- Spec: docs/specs/SPEC-runnerkits.md
- ADRs: ADR-6

## Goal
Runner-kit handlers use the shared JSON helpers and the CRUD and service validation paths are tested.

## Scope
### In
- Integration CRUD test, service unit tests, handler error normalization.
### Out
- Splitting the `db.RunnerKitItem` converter into its own package.
- `icon` enum validation unless product confirms the values.

## Design
- Public read stays at `/api/v1/events/event/{event_id}/runner-kits`, wired directly in `cmd/api/main.go`.
- `http.Error` plus hand-built JSON in `handler.go` is replaced by `json.WriteError` / `json.Write` (ADR-6).

## Tasks
- [ ] T1 Organizer CRUD integration test (create, list by event, update, delete) — test first: `TestRunnerKits_OrganizerCRUD` — files: `internal/integration_test/integration_test.go`
- [ ] T2 Service unit tests (nil UUID on `Update` and `GetByID` returns the invalid-id error; bad `name` length) — test first: `internal/runnerkits/service_test.go` — files: `internal/runnerkits/service_test.go`
- [ ] T3 Normalize the five handler methods to `json.WriteError`/`json.Write` — test first: handler tests asserting `{"error":"..."}` bodies, `TestRunnerKits_Public` stays green — files: `internal/runnerkits/handler.go`
- [ ] T4 (optional) Validate `icon` against a fixed set (`shirt`, `medal`, `bib`, `cap`, `towel`, `other`) — test first: unit test asserting 400 on an unknown icon — files: `internal/runnerkits/dto.go`, `internal/runnerkits/service_test.go`

## Verification
```bash
go test -race -count=1 ./internal/runnerkits/... ./internal/integration_test/...
```
Acceptance: previous status codes and messages unchanged except the normalized error body.

## Risks and rollback
- T3 may change an error message the frontend relies on: diff bodies before and after.
- T4 must not invent enum values the frontend does not use; defer until confirmed.

## Open questions
- Is the `icon` enum wanted?
- Keep `GenericResponse{Message, Data}` or match the envelope used elsewhere?
