# ADR-12: Money as integer cents

- Status: Proposed
- Date: 2026-11-06
- Deciders: Kinetix team
- Related: ADR-3, ADR-7, ADR-14, docs/DATABASE.md, docs/API_CONTRACTS.md

## Context
Money is stored two ways today. `organizer_payouts.amount_cents` is `BIGINT` (migration 022). `orders.amount`, `pricing_stages.price` and `awards.prize_amount` are `DECIMAL(10,2)` (migrations 008, 005, 004) and reach Go as `float64`, because `sqlc.yaml` maps `decimal` to `sql.Float64`. The webhook converts with `int64(order.Amount * 100)` (`internal/orders/service.go`, `recordOrganizerPayout`), and `SendPayout` divides back with `float64(payout.AmountCents) / 100.0` (`internal/admin/service.go:93`). `CalculatePayoutAmount` multiplies a `float64` by `1 - pct/100` and floors (`internal/admin/service.go:48-59`). Floating point can lose a cent.

## Decision
- All money in code, storage and internal APIs is an integer count of minor units (cents), named `*_cents`, type `int64`/`BIGINT`; the currency is a separate ISO 4217 column (`currency`, default `MXN`).
- Commission stays `NUMERIC(5,2)`; the commission amount is computed with integer arithmetic (basis points) and a documented rounding rule (floor for the organizer net, as `CalculatePayoutAmount` does today).
- Conversion to a decimal happens only at the edge: the MP client (`PayoutRequest.Amount`) and display.
- Never use `float64` for money, nor `float64 * 100`.

Migration path (expand -> migrate -> contract, per docs/DATABASE.md):
1. Add `amount_cents`, `price_cents`, `prize_amount_cents` columns with a backfill `ROUND(x * 100)::BIGINT`; write both columns in repositories.
2. Switch reads and DTOs to cents; keep the legacy decimal in responses for one release (non-breaking, optional field).
3. Expose cents in `/api/v2` or announce the deprecation of the decimal fields (API_CONTRACTS.md, Versioning).
4. Drop the decimal columns in a later migration after a backup step.

## Consequences
### Positive
- Exact arithmetic for commission and payouts; no float drift.
- One representation across `orders`, pricing, awards and payouts.

### Negative / risks
- Touches the public contract (`amount` in `POST /orders`, pricing and award fields) and the frontend; needs coordination.
- Dual-write period adds migration code and a risk of the two columns diverging.
- Zero-decimal or three-decimal currencies need an exponent per currency if more than MXN is added.

## Alternatives considered
- Keep `DECIMAL` and use `pgtype.Numeric`/`shopspring/decimal` in Go: rejected as the default, it keeps two models, though it is acceptable for display-only fields.
- Keep `float64` and round at the edges: rejected, it is the current source of risk.
- A `Money` value type wrapping cents and currency: compatible with this decision and recommended as the Go type when implementing.
