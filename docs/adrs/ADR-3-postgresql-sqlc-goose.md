# ADR-3: PostgreSQL with sqlc and goose

- Status: Accepted
- Date: 2026-09-15
- Deciders: Kinetix team
- Related: ADR-9, ADR-12, ADR-14, docs/DATABASE.md, sqlc.yaml

## Context
The domain is relational (events -> distances -> pricing/awards, registrations -> orders -> payouts) and needs constraints, JSONB (itinerary, GeoJSON) and UUID keys. The team wants typed queries without an ORM. `sqlc.yaml` generates `pgx/v5` code into `internal/adapters/postgresql/sqlc`, using the migrations directory as schema.

## Decision
- PostgreSQL 16 is the only datastore, accessed through a `pgx/v5` pool (`internal/database`).
- Queries are written in SQL (`internal/adapters/postgresql/sqlc/queries.sql`); `sqlc generate` produces typed Go. Generated files are never edited by hand.
- Schema is managed by `goose` SQL migrations in `internal/adapters/postgresql/migrations` (`NNN_<verb>_<subject>.sql`, currently 001-023), append-only, each with an `Up` and `Down`.
- `store.Store` wraps the pool and `*sqlc.Queries`; repositories receive the store (`internal/store/store.go`).
- Type overrides in `sqlc.yaml`: `uuid` -> `uuid.UUID`, `timestamptz`/`date` -> `time.Time`, `decimal` -> `sql.Float64`; JSONB stays `[]byte`.
- Enumerations are `VARCHAR + CHECK`; catalogs are seeded in migrations, demo data in `cmd/seed`.

## Consequences
### Positive
- SQL is explicit and reviewable; compile-time checked queries.
- Same migration files run in Docker (ADR-9) and are the sqlc schema, so there is one source of truth.

### Negative / risks
- `Store` exposes no transaction helper, so multi-row writes are not atomic (`internal/store/store.go`; registration in `internal/auth/service.go`). See ADR-14.
- The `decimal` override does not match `NUMERIC` columns, so money is converted manually (see ADR-12).
- `awards.place` is added in both 004 and 020; `orders.mercado_pago_payment_id` is not unique (see docs/DATABASE.md, "Data integrity gaps").
- Auth repositories reach `store.Queries` directly instead of going through a slice repository (`internal/auth/service.go`).

## Alternatives considered
- GORM/ent: rejected, hidden SQL and runtime reflection; harder to review queries.
- Raw `pgx` without sqlc: rejected, repetitive scanning code and no compile-time query checks.
- Other migration tools (golang-migrate, atlas): rejected, goose is already in the image (`Dockerfile`) and plain SQL fits sqlc.
