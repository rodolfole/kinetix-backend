# ADR-8: Payouts cron as a separate binary

- Status: Accepted
- Date: 2026-11-06
- Deciders: Kinetix team
- Related: ADR-7, ADR-9, docs/ARCHITECTURE.md, docs/SPEC-platform-payments.md

## Context
Sending money to organizers must not run in the HTTP request path nor depend on the API process being alive. docs/SPEC-platform-payments.md cut "auto-cron" to v2 and left only the manual `POST /admin/payouts/{id}/send`. The code now has `cmd/payouts-cron/main.go`, which reuses `admin.Service.SendPayout`.

## Decision
- Disbursement automation is a one-shot binary, `cmd/payouts-cron`, scheduled externally (cron, Task Scheduler or a Kubernetes CronJob, as its header comment says).
- Each run lists up to 100 `pending` payouts, calls `adminSvc.SendPayout` for each, and logs OK/FAIL per payout.
- The run has a 5 minute context timeout and exits with status `1` if any payout failed, so the scheduler can alert.
- Failed payouts are marked `failed` by `SendPayout`; they are not retried automatically.
- It shares config, `store`, `admin` and `mp` packages with the API; no separate module.

## Consequences
### Positive
- Failure isolation: a stuck MP call cannot block API requests.
- Same code path as the manual endpoint; scheduling is infrastructure's choice.

### Negative / risks
- `Dockerfile` builds only `cmd/api`, `cmd/seed` and `scripts`; the cron binary is not in the image and `docker-compose.yml` has no schedule for it.
- Overlapping runs can double-send: `SendPayout` reads `status = pending` and then calls MP without locking or an idempotency key (`internal/admin/service.go:77-101`). Use `SELECT ... FOR UPDATE SKIP LOCKED` (ADR-14) and an MP idempotency key.
- Only the first 100 pending payouts are processed per run.
- If MP succeeds but `MarkPayoutSent` fails, the payout stays `pending` and is paid again next run.
- Uses stdlib `log`, not the structured logger required by docs/PATTERNS.md.

## Alternatives considered
- Goroutine ticker inside the API: rejected, every replica would run it and a deploy would interrupt it.
- Send payout synchronously in the webhook: rejected, couples payment confirmation to a second external call.
- Message queue and worker: rejected as overkill for current volume.
