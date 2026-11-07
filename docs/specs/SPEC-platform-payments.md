# Spec: Platform payments with configurable commission

- Status: Implemented
- Plan: plans/PLAN-14-platform-payments.md

## Objective

Allow the platform to charge participants for event registrations and pay each organizer the registration amount minus a configurable platform commission. An admin can read/write the commission % via a UI.

**User stories**:
- As **admin**, I read/write the platform commission % from a `/admin/settings` page.
- As **admin**, I see a list of organizer payouts (pending, sent, failed) and can manually trigger a payout.
- As **organizer**, when my event is paid for, a pending `organizer_payout` is created with `amount_cents = order_total * (100 - commission_pct) / 100`.

**Decisions taken** (from prior plan):
- MP Marketplace API (Option 2): split happens at MP level; backend records the splits
- New role `'admin'` added to `users.role`
- Single global `commission_pct` in `platform_config` (no per-organizer override)

## Tech Stack

- Go 1.25 + sqlc + chi
- PostgreSQL 16
- Existing `internal/orders/` and `internal/middleware/` (don't re-author)
- New module: `internal/admin/`

## Commands

```bash
# Run migrations
goose -dir internal/adapters/postgresql/migrations postgres "$DB_DSN" up

# Regenerate sqlc
sqlc generate

# Build
go build ./...

# Test
go test ./internal/admin -count=1
```

## Project Structure

```
internal/
  adapters/postgresql/
    migrations/
      021_platform_config.sql       (new)
      022_organizer_payouts.sql     (new)
      023_organizer_mp_credentials.sql (new)
    sqlc/queries.sql                (modified: add 3 new tables)
  admin/                            (new module)
    handler.go
    service.go
    repository.go
    model.go
    dto.go
    handler_test.go                (new, Tier 3)
  orders/
    service.go                      (modified: append `organizer_payout` insert on payment)
cmd/api/main.go                     (modified: mount /admin/* routes)
```

## Code Style

- Module pattern matches existing (`internal/orders/`, `internal/events/`)
- DTOs use `validate:"..."` tags consistent with other handlers
- Repository methods: `GetConfig`, `UpdateConfig`, `ListPayouts`, `CreatePayout`, `MarkPayoutSent`
- Handlers return 400/401/404/500 using existing `kinetixjson.*` helpers
- Service layer does NOT touch DB directly; goes through `repo` (consistent with orders/events)

## Testing Strategy

- `internal/admin/handler_test.go`:
  - DTO validation (commission_pct range 0-100, comment field length)
  - GET /admin/config without auth → 401
  - GET /admin/config with non-admin token → 403
  - PUT /admin/config validation
- `internal/orders/service_test.go` (new, since the order module now writes to payouts):
  - When order webhook processes, verify a `organizer_payout` row is created with the right amount

## Boundaries

- **Always do**: Use `internal/json` helpers; run `go vet ./...`; run migrations on compose up
- **Ask first**: Add new tables, change orders handler/service signature
- **Never do**: Hardcode commission_pct in env, modify sqlc.yaml, remove existing tests

## Success Criteria

- `admin` role in JWT claims is recognized by middleware
- `GET /api/v1/admin/config` returns the current commission_pct
- `PUT /api/v1/admin/config` updates it and returns the new value
- `GET /api/v1/admin/payouts?status=pending` lists pending payouts
- When a `paid` order is recorded, a new `organizer_payout` row appears with `amount_cents = order_total * (100 - commission_pct) / 100` rounded down
- Migration runs cleanly on compose up
- All existing tests still pass

## Open Questions

- Currency: assuming MXN throughout. The currency column in `organizer_payouts` is free-text, so later can be added.
- Payout trigger: the spec only describes a `POST /admin/payouts/:id/send` for manual dispatch. The actual MP Payouts API call is a follow-up.

## Cut-line

If running short on time, cut in this order:
1. (don't cut) Migrations + admin/config endpoints — core requirement
2. (cut if needed) Payouts list endpoint
3. (cut if needed) Orders → payouts insert (this is the wiring that produces real value)
4. (cut first) Manual send endpoint (auto-cron comes in v2)
