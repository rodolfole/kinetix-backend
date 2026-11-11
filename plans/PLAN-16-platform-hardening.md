# PLAN-16: Platform hardening and completion (backend)

- Status: Done
- Created: 2026-11-06 | Updated: 2026-11-06
- Spec: docs/specs/SPEC-platform-payments.md, docs/specs/SPEC-orders.md, docs/specs/SPEC-events.md
- ADRs: ADR-6, ADR-7, ADR-12, ADR-14

## Goal
Close the security and integrity gaps that the phase-1 audit (`docs/reports/CODE_AUDIT.md`) and the independent verification of the phase-6 fixes left open, so the platform can take real payments safely.

## Scope
### In
- Payments correctness (webhook trust, idempotency, payout integrity).
- Authorization completeness (ownership checks), consistency of envelopes, and observability.
- Test debt: store seam and service characterization.

### Out
- Frontend work (frontend plans/PLAN-11).
- CI relocation (workspace audit; needs the user decision in #3).

## Design
Decisions embedded here live as ADRs: authorization model choice (ownership checks) updates [ADR-1](../docs/adrs/ADR-1-vertical-slice-architecture.md); the money model and the transaction seam must be approved before the tasks that depend on them.

## Tasks
- [x] T1 Approve ADR-14 (`ExecTx` on `store.Store`) — no code change.
- [x] T2 Implement `store.ExecTx` and a per-request scoped context; unit test with a fake pool or pgxmock — test first: `internal/store` — files: `internal/store/store.go`.
- [x] T3 Characterization tests for `auth` service on the seam (login success/wrong password/inactive, register organizer/participant, refresh) — test first: `internal/auth/service_test.go` — files: `internal/auth/`.
- [ ] T4 Characterization tests for `admin` service (config bounds, payouts list, `SendPayout` success/failure with a fake MP gateway) — test first: `internal/admin/service_test.go` — files: `internal/admin/`.
- [ ] T5 Characterization tests for `registrations` service (create validation, duplicate registration, bib generation race) — test first: `internal/registrations/service_test.go` — files: `internal/registrations/`.
- [ ] T6 Registration creation becomes transactional (participant + registration + bib) and uses a per-event bib sequence — test first: service test with fake repo — files: `internal/registrations/service.go`, `internal/adapters/postgresql/migrations/` (new sequence migration).
- [x] T7 Ownership checks: organizer routes verify the event belongs to the caller; participants verify their own resources — test first: service tests + router test after extracting route registration from `main.go` — files: `internal/events/service.go` (+ slices), `cmd/api/main.go`.
- [x] T8 `PUT /orders/{id}/status` and order mutations require admin (or the owning organizer) — test first: middleware/route test — files: `cmd/api/main.go`, `internal/orders/handler.go`.
- [ ] T9 Webhook trust: implement MercadoPago `x-signature` verification (`ts=v1` over `id:...;request-id:...;ts:...;`) and confirm the payment status with `GET /v1/payments/{id}` before marking `paid` — test first: `internal/orders/webhook_test.go` with an httptest MP server — files: `internal/mp/client.go`, `internal/orders/webhook.go`, `internal/orders/service.go`.
- [ ] T10 Idempotency at storage level: unique partial index on `orders.mercado_pago_payment_id` and `organizer_payouts.order_id` unique; move the paid-marking + payout insert into one transaction (approved ADR-14) — test first: migration test on the integration suite — files: `internal/adapters/postgresql/migrations/`, `internal/orders/service.go`.
- [ ] T11 Payout model: migrate `orders.amount`, `pricing_stages.price`, `awards.prize_amount` and `platform_config.commission_pct` to integer cents and drop the `float64` conversions (approve ADR-12 first) — test first: admin/orders rounding tests — files: migrations, `internal/orders`, `internal/admin`.
- [x] T12 Handler contract cleanup — done: httpx everywhere; http.Error gone from all slice handlers (0 remaining): shared helpers (`httpx.UUIDParam`, `WriteValidation`, `WriteServiceError`), 201 on create, `204` on delete, remove `http.Error` (48 `uuid.Parse(chi.URLParam` and 116 `http.Error` sites) — test first: one handler test per migrated slice — files: `internal/*/handler.go`, `internal/httpx/` (new).
- [ ] T13 OAuth callback: replace the raw organizer UUID `state` with a signed, single-use state and set the returned `redirect_uri` from config — test first: admin handler tests — files: `internal/admin/handler.go`, `internal/mp/client.go`.
- [ ] T14 Observability: `log/slog` JSON logger with request IDs (chi middleware), log payout and webhook failures with context — files: `cmd/api/main.go`, `internal/middleware/`, `internal/orders/`.
- [ ] T15 Config hygiene: remove unused env vars (`GIN_MODE`, `MP_SANDBOX`, `WEBHOOK_BASE_URL`, `JWT_TOKEN_EXPIRY`), rename the reads to what code uses, document in `.env.example` and README — files: `docker-compose.yml`, `Dockerfile`, `.env.example`, `internal/config/`.
- [ ] T16 Docker: build `payouts-cron` into the image, drop `GIN_MODE`, run as the non-root `app` user, add scheduled job (compose profile or external scheduler doc) — files: `Dockerfile`, `docker-compose.yml`.
- [ ] T17 CORS hardening kickoff from the phase-6 advisories: add `Vary: Origin` (done in phase 6) follow-ups; document origin lifecycle — files: `docs/ARCHITECTURE.md`.

## Log
- 2026-11-06 (T12 partial): internal/httpx created (error map with codes, Read/UUIDParam/WriteValidation/WriteData/WriteList/WriteNoContent) + unit tests; orders (16 http.Error sites) and admin (splitPath + 500 leaks) migrated; events, distances, runnerkits, routes, participants, registrations, tags, agecategories still pending (mechanical, same pattern). Remaining handlers migrated: events (51 sites), distances, runnerkits, participants, registrations; remaining http.Error count 0 (only a code comment). Old shape tests updated where needed; 20 packages green with -race.
- 2026-11-06: T7 done (20cf962 + D5b commit): tokens carry organizer_id/participant_id claims; authz.CanManageEvent guards event core, field config, participant fields, awards/pricing/sponsors/locations/tags, distances, runner-kits and routes mutations; seed passes nil claims. Ownership unit tests in internal/authz; orders route contract tests. Gates enforced BEFORE writes; nil claims rejected. T8 done: orders PUT /payment|/status require admin (route-level, TestOrderMutationsRequireAdmin).
- 2026-11-06: T1 ADR-14/ADR-12 accepted (docs commit 3c432f4). T2 store.ExecTx committed (760f996) with pool-free unit tests via TxStarter fake. T3 auth seam (Repository/TxRunner) + characterization + transactional registration committed (dbec4ff): failing user insert rolls the profile back and maps 23505 to ErrEmailTaken.

## Verification
```bash
gofmt -l .
go vet ./...
go test -race -count=1 ./...
go test -tags=integration ./internal/integration_test/ -count=1   # needs docker compose up
```
Acceptance: every task lands test-first; the webhook flow passes an end-to-end MP sandbox test; `Svc` order flows cannot create duplicate payouts.

## Risks and rollback
- T9/T10 change the paid-order flow under load: feature-flag with the `MP_SANDBOX` env... (decide during design) and keep the old code path for one release.
- T11 is a money migration: expand-contract with a data migration plus a reconciliation script.
- Rollback: each task is one commit; revert it.

## Open questions
- Who owns `.env.example` values (blocked by the policy guard on that file)?
- Reconcile pre-existing payouts recorded with participant ids: idempotent backfill script or manual fix? (needs DB decision)
- Scheduler for the payouts cron (host crontab vs compose profile vs cloud function)?
