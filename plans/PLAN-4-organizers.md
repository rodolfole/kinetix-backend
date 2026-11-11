# PLAN-4: Organizers projection and write-path coverage

- Status: Draft
- Created: 2026-09-25 | Updated: 2026-11-06
- Spec: docs/specs/SPEC-organizers.md
- ADRs: ADR-6, ADR-5

## Goal
The public organizer endpoint exposes only safe fields, the handler uses the shared JSON helpers, and create/update/delete are covered end to end.

## Scope
### In
- Public projection, `internal/json` error helpers on parse/UUID paths, integration and unit tests for write paths.
### Out
- Moving the public GET behind `JWTAuth`.
- New types beyond the public projection.

## Design
- Keep handler/service/repository layering.
- Migrate only the read/UUID `http.Error` paths now (service-error path already uses `json.WriteError`) to keep the diff small (ADR-6).

## Tasks
- [ ] T1 Project the public GET to `id, business_name, brand_name, logo_url, contact_email, contact_phone` (`PublicOrganizerResponse`) — test first: `TestPublicOrganizerDetail` asserts billing fields absent — files: `internal/organizers/model.go`, `internal/organizers/handler.go`
- [ ] T2 Replace `http.Error` with `json.WriteError` in `CreateOrganizer`, `GetOrganizer`, `UpdateOrganizer`, `DeleteOrganizer` — test first: handler tests asserting `{"error": "..."}` body — files: `internal/organizers/handler.go`
- [ ] T3 Integration test for create plus RFC conflict (201, then 400 "already exists") — test first: `TestOrganizers_CreateAndConflict` — files: `internal/integration_test/integration_test.go`
- [ ] T4 Integration test for update plus delete (brand_name change, then GET returns 404) — test first: `TestOrganizers_UpdateDelete` — files: `internal/integration_test/integration_test.go`
- [ ] T5 Unit test for RFC uniqueness in the service with a fake repo — test first: `internal/organizers/service_test.go` — files: `internal/organizers/service_test.go`

## Verification
```bash
go test -race -count=1 ./internal/organizers/... ./internal/integration_test/...
```
Acceptance: public GET has no billing fields; seeded `contact@kinetix.com` can still log in and create an event.

## Risks and rollback
- T1 changes the GET body; grep the frontend for `billing_*` reads before merging.
- Tests must use a fresh RFC per run and clean up with `t.Cleanup`; never delete the seeded organizer.

## Open questions
- Should the public GET expose billing fields (product decision)?
- Should DELETE be blocked while active events reference the organizer?
