# ADR-7: MercadoPago marketplace payments

- Status: Accepted
- Date: 2026-11-06
- Deciders: Kinetix team
- Related: ADR-8, ADR-12, ADR-14, docs/specs/SPEC-platform-payments.md, docs/API_CONTRACTS.md, docs/DATABASE.md

## Context
Participants pay for registrations; the platform keeps a configurable commission and pays the organizer the rest. The "Decisions taken" block of docs/specs/SPEC-platform-payments.md fixes the approach: MercadoPago Marketplace API (split at MP level, the backend records the splits), a new `admin` role, and a single global `commission_pct` in `platform_config` (no per-organizer override). Code: `internal/orders` (webhook), `internal/admin` (config, payouts, OAuth), `internal/mp/client.go`, migrations 021-023.

## Decision
- MercadoPago notifies payments at `POST /api/v1/orders/webhook` (public route). The handler validates an HMAC-SHA256 signature, then marks the order `paid` (`internal/orders/webhook.go`, `internal/orders/service.go:ProcessWebhook`).
- After payment a `pending` row is inserted into `organizer_payouts` with `amount_cents = floor(total_cents * (100 - commission_pct) / 100)` (`admin.CalculatePayoutAmount`, `internal/admin/service.go:48`).
- Organizers connect their MP account through OAuth (`/admin/organizers/{id}/connect-mp` and `/callback`); tokens are stored in `organizer_mp_credentials`.
- Payouts are sent through `mp.Client.CreatePayout` with a 15s HTTP timeout (`internal/mp/client.go`), by `POST /admin/payouts/{id}/send` or the cron (ADR-8).
- Commission is read and written by admin only (`GET/PUT /admin/config`, `RequireRole("admin")`).

## Consequences
### Positive
- The platform does not hold funds in its own logic; MP performs the split.
- Commission changes need no deploy.

### Negative / risks
- The signature scheme is an HMAC of the raw body compared to `X-Merchant-Token`/`X-Signature` (`internal/orders/webhook.go`), not MercadoPago's `x-signature` (`ts`, `v1`) scheme. An empty `MP_WEBHOOK_SECRET` skips verification entirely.
- The handler never confirms the payment with the MP payments API; any signed payload with a known payment id marks the order `paid`. Replays are not idempotent (`mercado_pago_payment_id` is not unique, docs/DATABASE.md).
- `recordOrganizerPayout` writes `OrganizerID: order.ParticipantID` and discards the `CreatePayout` error with `_, _ =` (`internal/orders/service.go`, `recordOrganizerPayout`), so payouts can point to a non-organizer or be silently lost; `organizer_payouts` has no `order_id` link in the schema (`022_create_organizer_payouts.sql`).
- Order status update and payout insert are not in one transaction (ADR-14).
- `totalCents := int64(order.Amount * 100)` converts through `float64` (ADR-12).
- The webhook returns `err.Error()` as `500` and does not distinguish unknown orders (`internal/orders/handler.go`).
- `internal/mp/client.go` is marked as placeholder endpoints; `CreatePayout` sends no idempotency key and `PayoutRequest.Amount` is a `float64`. OAuth `state` is the raw organizer UUID (`internal/admin/handler.go`). MP tokens are stored in plain text.

## Alternatives considered
- Charge into the platform account and pay organizers manually: rejected, platform becomes the holder of funds and reconciliation is manual.
- Per-organizer commission: rejected for now (spec "Decisions taken"); a single `platform_config` row is enough.
- Another gateway (Stripe Connect): rejected, MercadoPago is the market-appropriate choice (MXN, `auth.mercadopago.com.mx`).
