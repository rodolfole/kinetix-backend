# ADR-14: Store transactions with ExecTx

- Status: Proposed
- Date: 2026-11-06
- Deciders: Kinetix team
- Related: ADR-3, ADR-7, ADR-8, ADR-12, docs/DATABASE.md, docs/PATTERNS.md

## Context
`store.Store` wraps a `pgxpool.Pool` and a `*sqlc.Queries` bound to the pool and has no transaction support (`internal/store/store.go`). Use cases that write several rows therefore run as separate statements:
- `auth.RegisterOrganizer`/`RegisterParticipant` create the profile row, then the user row (`internal/auth/service.go`); if the second fails an orphan profile remains, even though the doc comments say "single transaction".
- Registration creation reads `GetMaxBibNumberByEvent` and inserts, which races under concurrency.
- The webhook updates the order to `paid` and then inserts the payout (`internal/orders/service.go`).
- `SendPayout` calls MP and then marks the payout sent (`internal/admin/service.go`).

## Decision
- Add to `internal/store`:

```go
func (s *Store) ExecTx(ctx context.Context, fn func(q *sqlc.Queries) error) error
```

  It begins a `pgx` transaction, builds `Queries.WithTx(tx)`, runs `fn`, commits on `nil`, rolls back on error or panic, and returns the original error.
- Any use case that writes more than one row runs inside one `ExecTx`. Repositories expose a method taking the `*sqlc.Queries` of the transaction, or the service passes `store.ExecTx` through a small interface so tests can fake it.
- Never call an external service (MP) inside a transaction; record intent first, call after commit, then update state.
- Bib numbers use a per-event sequence or `SELECT ... FOR UPDATE`; the `UNIQUE` constraint stays as the last guard.

Migration path: (1) add `ExecTx` with tests against a real Postgres (testhelpers); (2) convert registration of users, then the webhook (order + payout, with the unique `order_id` link in `organizer_payouts`); (3) convert registrations/bib number; (4) make `SendPayout` claim rows with `FOR UPDATE SKIP LOCKED` before calling MP (ADR-8).

## Consequences
### Positive
- Atomic multi-row writes; no orphan profiles or paid orders without a payout.
- A single, tested place for commit and rollback rules.

### Negative / risks
- Services need the store or a transaction runner, changing constructor signatures (`registrations.NewService` already receives `appStore`).
- Unit tests with fakes cannot prove transactional behavior; integration tests on Postgres are required.
- Long transactions hold connections from the pool; keep `fn` short.
- Until implemented, the gaps above remain open.

## Alternatives considered
- Per-slice ad hoc `pool.Begin` calls: rejected, duplicated rollback logic and inconsistent behavior.
- Compensating deletes on failure: rejected, they can also fail and are not atomic.
- Saga/outbox for payouts: rejected for now; revisit if payouts span more than one database transaction.
