# Database

PostgreSQL 16. Schema is owned by goose migrations; queries are owned by sqlc. See [ADR-3](adrs/ADR-3-postgresql-sqlc-goose.md).

## Workflow

| Step | Command |
|---|---|
| New migration | `goose -dir internal/adapters/postgresql/migrations create <verb_subject> sql` |
| Apply | `goose -dir internal/adapters/postgresql/migrations postgres "$DATABASE_URL" up` |
| Roll back one | `... down` |
| Regenerate queries | `sqlc generate` |
| Local stack | `docker compose up` (runs `db-init`, `migrations`, `seed`, `api`) |

Rules:
1. **Migrations are append-only.** Never edit a migration that has run in any shared environment; write a new one.
2. File name: `NNN_<verb>_<subject>.sql`, three-digit sequence (`024_add_orders_payment_unique.sql`). Sequence numbers never repeat.
3. Every migration has a working `-- +goose Down`. Use `StatementBegin/StatementEnd` around functions and multi-statement blocks.
4. Never hand-edit files in `internal/adapters/postgresql/sqlc/*.go`. Change `queries.sql`, run `sqlc generate`, commit both.
5. Schema changes that drop or rewrite data need a plan in `plans/` and a backup step. Prefer expand -> migrate -> contract over a single destructive migration.
6. Seed data for catalogs (`distance_types`, `age_categories`, `platform_config`) lives in migrations. Demo data lives in `cmd/seed` only.

## Naming

- Tables: plural `snake_case` (`pricing_stages`). Columns: `snake_case`.
- Primary key: `id UUID DEFAULT gen_random_uuid()`. Exception: 1:1 tables keyed by the parent (`organizer_mp_credentials.organizer_id`).
- Foreign keys: `<entity>_id`, always indexed.
- Timestamps: `created_at`, `updated_at` as `TIMESTAMPTZ NOT NULL DEFAULT now()`. Every table with `updated_at` gets the `update_updated_at_column()` trigger.
- Enumerations: `VARCHAR + CHECK (... IN (...))` (current convention). Keep the CHECK and the Go constants in sync.
- Indexes: `idx_<table>_<columns>`; unique: `uq_<table>_<columns>`.

## Schema map

| Table | Purpose | Key constraints |
|---|---|---|
| `organizers` | Organizer business profile | `rfc` UNIQUE |
| `users` | Login accounts | `email` UNIQUE, `role IN (admin, organizer, participant)`, FKs to `organizers` / `participants` `ON DELETE SET NULL` |
| `participants` | Runner profile | |
| `events` | Race | `slug` UNIQUE, `status IN (draft, published, cancelled, completed)`, `itinerary JSONB`, FK `organizers` `ON DELETE CASCADE` |
| `distance_types` | Catalog of distances (km) | `km` UNIQUE, seeded |
| `distances` | Distance offered by an event | FK `events` |
| `pricing_stages` | Price windows per distance | `end_date >= start_date` |
| `awards` | Prizes per distance/category/gender/place | `gender IN (male, female, general)`, `place >= 1`, `uq_awards_distance_age_gender_place` |
| `age_categories` | Catalog | seeded |
| `registrations` | Participant in an event distance | `bib_number` UNIQUE, UNIQUE(`event_id`, `participant_id`, `distance_id`), `status IN (pending, confirmed, cancelled)` |
| `orders` | Payment for a registration | `status IN (pending, paid, failed, refunded, cancelled)`, `currency` default `MXN` |
| `sponsors`, `registration_locations`, `runner_kit_items`, `event_tags` | Event sub-resources | FK `events` |
| `event_participant_fields` | Custom registration fields | UNIQUE(`event_id`, `field_name`) |
| `event_field_config` | Optional form sections per event | UNIQUE(`event_id`, `section_name`) |
| `routes` | Parsed GPX route | `geojson JSONB`, `elevation_profile JSONB` |
| `platform_config` | Global commission | single row, `commission_pct NUMERIC(5,2)` 0..100 |
| `organizer_payouts` | Money owed to organizers | `amount_cents BIGINT >= 0`, `status IN (pending, sent, failed)` |
| `organizer_mp_credentials` | MercadoPago OAuth tokens | PK `organizer_id` |

## Transactions

Any use case that writes more than one row runs inside one transaction. Target API on `store.Store`:

```go
func (s *Store) ExecTx(ctx context.Context, fn func(q *sqlc.Queries) error) error
```

Required today (no transaction exists yet):
- `auth.RegisterOrganizer` / `RegisterParticipant`: profile row + user row.
- Registration creation with bib number: `GetMaxBibNumberByEvent` + insert races under concurrency. Use a per-event sequence or `SELECT ... FOR UPDATE`, and keep the UNIQUE constraint as the last guard.
- Webhook: order `paid` + payout insert.

## Money

- New columns store integer cents (`BIGINT`, suffix `_cents`). See [ADR-12](adrs/ADR-12-money-as-integer-cents.md).
- Existing `DECIMAL(10,2)` columns (`orders.amount`, `pricing_stages.price`, `awards.prize_amount`) stay until migrated; convert with `pgtype.Numeric`, never via `float64 * 100`.

## Data integrity gaps

| Gap | Fix |
|---|---|
| `orders.mercado_pago_payment_id` index is not unique, so a replayed webhook can be processed twice | Partial UNIQUE index where not null + idempotent handler |
| `organizer_payouts` has no link to the order, so duplicates cannot be detected | Add `order_id UUID UNIQUE REFERENCES orders(id)` |
| `organizer_mp_credentials` stores tokens in plain text | Encrypt at application level (AES-GCM, key from env) |
| `update_updated_at_column()` trigger only on 4 tables | Attach to every table with `updated_at` |
| `awards.place` added in 004 and again in 020 | Leave (append-only); document as no-op |
| sqlc `decimal` override does not match `NUMERIC` columns | Add a `numeric` override or keep `pgtype.Numeric` explicitly |
