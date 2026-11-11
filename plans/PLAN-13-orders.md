# PLAN-13: Orders coverage and webhook hardening

- Status: In progress
- Created: 2026-09-25 | Updated: 2026-11-06
- Spec: docs/specs/SPEC-orders.md
- ADRs: ADR-7, ADR-12

## Goal
Orders CRUD and the MercadoPago webhook are covered by tests and the webhook is idempotent.

## Scope
### In
- Orders CRUD integration test, webhook signature tests, idempotent `ProcessWebhook`, audit-storage decision.
### Out
- Changing the HMAC scheme (needs a decision first).
- Moving the webhook inside the JWT group.
- Fail-closed webhook secret, organizer resolution and payout fixes: PLAN-15 (B4, B5, B6).

## Design
- Webhook stays at `/api/v1/orders/webhook`, outside JWT (ADR-7).
- HMAC-SHA256 validation stays in `internal/orders/webhook.go`.
- Handler migration to `internal/json` is deferred to the consistency sweep (PLAN-11 T4).

## Tasks
- [ ] T1 Orders CRUD integration (create, list by event, get, update status) — test first: `TestOrders_CreateListGetStatusUpdate` — files: `internal/integration_test/integration_test.go`
- [x] T2 Webhook signature unit tests (missing, wrong, valid, body mismatch, empty-secret bypass) and handler dispatch tests — test first: `internal/orders/webhook_test.go` — files: `internal/orders/webhook_test.go`, `internal/orders/handler_test.go`
- [ ] T3 Decide on raw-body audit storage (`webhook_payloads` table); record the decision in the spec — test first: integration test asserting a stored row, only if approved — files: `docs/specs/SPEC-orders.md`, possibly a migration
- [ ] T4 Make `ProcessWebhook` idempotent (already-paid order is a no-op) — test first: two webhook calls cause one status transition (see PLAN-15 B5) — files: `internal/orders/service.go`
- [ ] T5 (optional) Migrate the handler to `internal/json` helpers — test first: existing webhook tests stay green — files: `internal/orders/handler.go`

## Verification
```bash
go test -race -count=1 ./internal/orders/... ./internal/integration_test/...
```
Acceptance: orders CRUD and webhook paths covered; repeated webhook does not repeat side effects.

## Risks and rollback
- Idempotency can mask real failures: log "already paid" webhook hits.
- Gate T3 on a product decision to avoid scope creep.

## Open questions
- Raw-body audit storage: yes or no?
- Should an empty webhook secret be accepted in dev? (PLAN-15 B6 makes production fail closed.)
