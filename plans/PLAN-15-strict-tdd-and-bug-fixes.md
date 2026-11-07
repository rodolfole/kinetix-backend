# PLAN-15: Strict TDD and bug fixes (backend)

- Status: Approved
- Created: 2026-11-06 | Updated: 2026-11-06
- Spec: docs/specs/SPEC-auth.md, docs/specs/SPEC-orders.md, docs/specs/SPEC-platform-payments.md, docs/specs/SPEC-events.md
- ADRs: ADR-4, ADR-5, ADR-7, ADR-11, ADR-12, ADR-14

## Goal
Every backend bug found in the audit (B1-B9) is fixed test-first, and the untested services (auth, orders, registrations, admin) have characterization tests that pin current behavior.

## Scope
### In
- B1-B9 with a RED test, then the minimal GREEN fix.
- Characterization tests for services `auth`, `orders`, `registrations`, `admin`.
### Out
- Frontend bugs F1-F9 (frontend repo plan).
- Refactors not required by a fix (handler migration to `internal/json`, event package split).
- Behavior changes beyond the bug list.

## Design
- Rule: write the failing test first and capture its failure, then the smallest fix, then rerun; characterization tests pin existing behavior and pass immediately.
- Services need seams (small interfaces over the store or repository) so tests run without a database; add them only where a test needs them.
- Integration tests run only with the `integration` build tag (B8).

## Tasks
- [ ] T1 B1 Refresh token interchangeable with access token: add a `typ` claim (`access`/`refresh`); `JWTAuth` rejects refresh tokens and `/auth/refresh` rejects access tokens — test first: `internal/auth/auth_test.go` and `internal/middleware` test (refresh token on a protected route returns 401; access token on `/auth/refresh` returns 401) — files: `internal/auth/auth.go`, `internal/auth/service.go`, `internal/middleware/middleware.go`
- [ ] T2 B2 `JWT_SECRET` default makes the required check unreachable: remove the default, fail on empty — test first: `internal/config/config_test.go` (empty secret panics/returns error) — files: `internal/config/config.go`, `docker-compose.yml` (dev value stays explicit)
- [ ] T3 B3 CORS hardcoded to `:4322`: read `CORS_ALLOWED_ORIGINS`, default `http://localhost:4325,http://localhost:4322` — test first: `internal/middleware/cors_test.go` (allowed origin echoed, unknown origin not, default list) — files: `internal/middleware/middleware.go`, `internal/config/config.go`, `cmd/api/main.go`
- [ ] T4 B4 Webhook payout uses `order.ParticipantID` as organizer and discards the error: resolve the organizer through the event, return/log the error — test first: `internal/orders/service_test.go` (payout `OrganizerID` equals the event's organizer; repo error is surfaced) — files: `internal/orders/service.go`
- [ ] T5 B5 Webhook not idempotent: an already-paid order creates no second payout — test first: `internal/orders/service_test.go` (two webhook calls, one payout) — files: `internal/orders/service.go`
- [ ] T6 B6 Empty `MP_WEBHOOK_SECRET` skips verification: fail closed — test first: update `TestValidateMercadoPagoWebhook_EmptySecretBypasses` (currently asserts the bypass) to expect rejection, plus handler test returning 401/403 — files: `internal/orders/webhook.go`, `internal/orders/handler.go`
- [ ] T7 B7 `orders.List` ignores UUID parse errors: return 400 for malformed `event_id`/`participant_id` — test first: `internal/orders/handler_test.go` (`TestList_BadEventIDReturns400`, `TestList_BadParticipantIDReturns400`) — files: `internal/orders/handler.go`
- [ ] T8 B8 `internal/integration_test` has no build tag: add `//go:build integration` — test first: `go test ./...` without tags must not compile or run them (observed before: tests need the live stack) — files: `internal/integration_test/*.go`, `test.sh`, docs/TESTING.md references if any
- [ ] T9 B9 Event-management routes accept any authenticated role: require organizer or admin — test first: router-level test (participant token on `POST /events` returns 403; organizer and admin pass) — files: `internal/middleware/middleware.go` (multi-role helper), `cmd/api/main.go`
- [ ] T10 Characterization tests for `auth` service (login, register organizer/participant, refresh, sentinel errors) — test first: `internal/auth/service_test.go` — files: `internal/auth/service_test.go` (plus a store seam if required)
- [ ] T11 Characterization tests for `orders` service (create, status update, webhook paid transition, payout amount) and the `mp` client with `httptest` — test first: `internal/orders/service_test.go`, `internal/mp/client_test.go` — files: same
- [ ] T12 Characterization tests for `registrations` service (create, bib generation, duplicate, `with-participant`) — test first: `internal/registrations/service_test.go` — files: same
- [ ] T13 Characterization tests for `admin` service and handlers (config get/update, payouts list, `SendPayout` success/failure with a fake MP client) — test first: `internal/admin/service_test.go` — files: same

## Verification
```bash
gofmt -l .
go vet ./...
go test -race -count=1 ./...
go test -race -count=1 -tags integration ./internal/integration_test/...   # needs the compose stack
```
Acceptance: each bug has a test that failed before its fix (output recorded in the commit/PR); all commands green; no unrelated behavior changed.

## Risks and rollback
- B1 invalidates already issued tokens without a `typ` claim: users must log in again (accept; tokens are short-lived).
- B2 and B6 make misconfigured environments fail at startup or reject webhooks: update compose and deployment secrets in the same change.
- B9 may block flows that relied on the loose check: verify with the seeded organizer and an admin.
- Rollback: each task is an independent commit; revert the commit.

## Open questions
- Which interface seams for stores are acceptable per ADR-14 to keep services unit-testable without a database?
- Should refresh tokens be rotated on use (out of scope here)?
