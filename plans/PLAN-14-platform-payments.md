# PLAN-14: Platform payments with configurable commission

- Status: Done
- Created: 2026-09-27 | Updated: 2026-11-06
- Spec: docs/specs/SPEC-platform-payments.md
- ADRs: ADR-7, ADR-8, ADR-12

## Goal
The platform charges participants through MercadoPago, keeps a configurable commission, and records and sends the net amount to each organizer.

## Scope
### In
- `admin` role, global `commission_pct` in `platform_config`, organizer payout records, MP OAuth credentials, manual and cron payout dispatch.
### Out
- Per-organizer commission override.
- MP token refresh job (credentials expire; refresh is a follow-up).
- Multi-currency (currency column is free text, MXN assumed).
- Correctness fixes found later (organizer resolution, idempotency, fail-closed secret): PLAN-15 (B4, B5, B6).

## Design
- MP Marketplace API: the split happens at MP; the backend records the amounts (ADR-7).
- `internal/admin` (handler/service/repository) owns config, payouts and OAuth; `internal/mp/client.go` wraps OAuth and payouts calls; `internal/orders` inserts a pending payout after payment (`recordOrganizerPayout`).
- Net amount via `admin.CalculatePayoutAmount(total, pct)`, rounded down, in integer cents (ADR-12).
- `cmd/payouts-cron` lists pending payouts and calls `admin.Service.SendPayout` in a separate binary (ADR-8).
- Migrations: `021_create_platform_config` (singleton row, `CHECK 0..100`), `022_create_organizer_payouts` (status `pending|sent|failed`), `023_create_organizer_mp_credentials`.
- Routes under `/api/v1/admin/*` require JWT plus `RequireRole("admin")`: `GET|PUT /admin/config`, `GET /admin/payouts`, `GET /admin/organizers/{id}/connect-mp`, `POST /admin/organizers/callback`, `POST /admin/payouts/{id}/send`.

## Tasks
- [x] T1 Add migrations 021-023 with working Down — test first: n/a (schema), verified by goose up/down — files: `internal/adapters/postgresql/migrations/021_create_platform_config.sql`, `022_create_organizer_payouts.sql`, `023_create_organizer_mp_credentials.sql`
- [x] T2 Add `admin` module (config get/update, payouts list, payout amount calculation) — test first: `internal/admin/handler_test.go` (`TestUpdateConfigRequest_Validate`, `TestUpdateConfig_InvalidCommissionReturns400`, `TestCalculatePayoutAmount`) — files: `internal/admin/*.go`
- [x] T3 Add `RequireRole` middleware and mount admin routes — test first: `internal/middleware/role_test.go` — files: `internal/middleware/middleware.go`, `cmd/api/main.go`
- [x] T4 Add MP client (OAuth authorize URL, code exchange, create payout) and OAuth endpoints — test first: none yet (no MP client tests; see PLAN-15 T11) — files: `internal/mp/client.go`, `internal/admin/handler.go`
- [x] T5 Record a pending `organizer_payout` when an order is paid — test first: none yet (webhook payout path untested; see PLAN-15) — files: `internal/orders/service.go`
- [x] T6 Add payout dispatch: `POST /admin/payouts/{id}/send` and `cmd/payouts-cron` — test first: none yet — files: `internal/admin/service.go`, `cmd/payouts-cron/main.go`

## Verification
```bash
go build ./... && go vet ./...
go test -race -count=1 ./internal/admin/... ./internal/middleware/...
goose -dir internal/adapters/postgresql/migrations postgres "$DB_DSN" up
```
Acceptance: admin endpoints reject missing/non-admin tokens; a paid order creates a pending payout with `amount_cents = total * (100 - pct) / 100` rounded down.

## Risks and rollback
- Known defects in the webhook payout path (wrong organizer id, swallowed error, no idempotency, empty secret skips verification) are fixed in PLAN-15 (B4-B6).
- Rollback: goose down 023, 022, 021 (Down drops indexes and tables).

## Open questions
- Who refreshes organizer MP tokens before `expires_at`?
- Should failed payouts retry automatically in the cron?
