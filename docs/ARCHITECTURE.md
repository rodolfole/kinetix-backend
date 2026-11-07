# Architecture

Kinetix API is the backend of a running-events platform: organizers publish races (distances, pricing stages, awards, runner kits, GPX routes), participants register and pay through MercadoPago, and the platform keeps a commission and pays organizers out.

## Stack

| Concern | Choice | Notes |
|---|---|---|
| Language | Go 1.25 | Module `kinetix-api` |
| HTTP | `go-chi/chi/v5` | stdlib `net/http` handlers |
| Database | PostgreSQL 16 | `pgx/v5` pool |
| Queries | `sqlc` | Generated into `internal/adapters/postgresql/sqlc` |
| Migrations | `goose` | `internal/adapters/postgresql/migrations` |
| Validation | `go-playground/validator/v10` | `validate:` struct tags |
| Auth | JWT HS256 (`golang-jwt/jwt/v5`) + bcrypt | See [ADR-4](adrs/ADR-4-jwt-authentication.md) |
| Payments | MercadoPago (webhook + OAuth + payouts) | See [ADR-7](adrs/ADR-7-mercadopago-marketplace-payments.md) |
| Runtime | Docker multi-stage image + docker compose | See [ADR-9](adrs/ADR-9-docker-compose-local-stack.md) |

## Binaries

| Path | Purpose |
|---|---|
| `cmd/api` | HTTP server. Wires every slice by hand and owns the route table. |
| `cmd/seed` | Loads demo data into a migrated database. |
| `cmd/payouts-cron` | One-shot job that sends pending organizer payouts. Exits `1` if any payout fails. |
| `scripts/create_db.go` | Creates the database (`kinetix-dbinit` in the image). |

## Layout

```
cmd/<binary>/main.go          entry points, composition root
internal/
  <feature>/                  one vertical slice per domain
    handler.go                HTTP: decode, validate, call service, encode
    service.go                business rules, orchestration
    repository.go             data access over store.Queries (sqlc)
    model.go                  domain types
    dto.go                    request/response types with validate tags
    *_test.go
  adapters/postgresql/
    migrations/               goose SQL files, also the sqlc schema
    sqlc/                     queries.sql + generated code (never hand-edit *.go)
  store/                      Store{pool, Queries}: the only DB entry point
  database/                   pool creation, pgtype converters
  middleware/                 JWTAuth, RequireRole, CORS, rate limit
  json/                       Read / Write / WriteError helpers
  config/                     env loading
  mp/                         MercadoPago HTTP client
  routeparser/                GPX parsing, distance, GeoJSON, elevation
  testhelpers/, integration_test/
```

## Request flow

```
chi router -> global middleware -> [JWTAuth] -> [RequireRole] -> Handler -> Service -> Repository -> store.Queries -> PostgreSQL
```

Global middleware order (`cmd/api/main.go`): `RequestID`, `RealIP`, `Logger`, `Recoverer`, `Timeout(60s)`, `CORS`. `/api/v1/auth/*` adds `AuthRateLimit` (10 req/min per client).

## Rules

1. **One slice per domain** under `internal/<feature>`. Every slice has the same files: `handler.go`, `service.go`, `repository.go`, `model.go`, `dto.go`. A slice with no business rules still gets a thin service so handlers never call repositories.
2. **Dependencies point inward**: handler -> service -> repository -> store. A slice never imports another slice's handler or repository; cross-slice reads go through the other slice's service interface.
3. **Services depend on interfaces**, declared in the consuming package (`type Repository interface{...}` in `service.go`). This is what makes services unit-testable without a database.
4. **Routes live in each slice** as `func (h *Handler) Routes(r chi.Router)`. `cmd/api` only mounts them and decides public/protected groups. No duplicated or dead `Routes()` functions.
5. **Authorization is explicit per route**: public, `JWTAuth`, or `JWTAuth + RequireRole(...)`. Ownership checks (organizer owns the event) belong in the service, using claims from the context.
6. **Multi-statement writes run in one transaction** (`store.ExecTx`). See [DATABASE.md](DATABASE.md).
7. **Configuration comes only from `internal/config`**. No `os.Getenv` in slices, no hardcoded origins, secrets or URLs.
8. **Errors cross layers as typed/sentinel errors** and are mapped to HTTP once, in the handler. See [PATTERNS.md](PATTERNS.md).

## Known gaps (target state)

| Gap | Where | Target |
|---|---|---|
| Slices use different file sets (`participant.go`, `order.go`, `registration.go` as repos; `agecategories`, `tags` without service) | `internal/*` | Rule 1 |
| `tags` lives in `internal/features/tags` | `internal/features/tags` | Move to `internal/tags` |
| `events.Handler.Routes()` and `admin.Handler.Routes()` are dead; routes are listed one by one in `main.go` | `internal/events/handler.go`, `internal/admin/handler.go` | Rule 4 |
| Most protected routes check only that a JWT exists, not the role or ownership | `cmd/api/main.go` | Rule 5 |
| No transactions; registration writes 3 rows sequentially | `internal/auth/service.go` | Rule 6 |
| CORS origin hardcoded to `http://localhost:4322`; `CORS_ALLOWED_ORIGINS` is ignored | `internal/middleware/middleware.go` | Rule 7 |
| Two identical `auth.Config` values built in `main.go` | `cmd/api/main.go` | One config |
| stdlib `log` only | everywhere | `log/slog` JSON logger with request ID |
| `payouts-cron` not built into the Docker image | `Dockerfile` | Build and schedule it |

Fixes are tracked in [plans/](../plans/README.md) (see PLAN-15) and specs live in [docs/specs/](specs/) and the audit in [reports/CODE_AUDIT.md](reports/CODE_AUDIT.md).
