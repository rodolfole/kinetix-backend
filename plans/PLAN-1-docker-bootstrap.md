# PLAN-1: Docker bootstrap

- Status: Done
- Created: 2026-09-14 | Updated: 2026-11-06
- Spec: none (infrastructure)
- ADRs: ADR-9

## Goal
A single `docker compose up` leaves the backend fully functional (postgres, database, migrations, seed, API) with no manual step and no local Postgres or goose install.

## Scope
### In
- Multi-stage `Dockerfile` building `kinetix-api`, `kinetix-seed`, `kinetix-dbinit` and `goose`.
- `docker-compose.yml` chaining `postgres -> db-init -> migrations -> seed -> api`.
- `scripts/create_db.go` reading `DB_*` environment variables.
### Out
- HTTPS / reverse proxy (Caddy, nginx).
- Production deployment (managed database, secret store).
- Frontend port alignment (tracked in the frontend repo).

## Design
- Build stage `golang:1.25.5-alpine`: `go mod download`, `goose v3.22.1`, static binaries with `-trimpath -ldflags="-s -w"`.
- Runtime stage `alpine:3.20`: non-root user `app`, the three binaries, `goose` and the migrations directory; `EXPOSE 8080`, `CMD /app/kinetix-api`.
- Compose services are chained with `depends_on` + `condition`; `postgres:16-alpine` has a `pg_isready` healthcheck; volume `kinetix-pgdata` persists data.
- API is published on `8085:8080` (8080 was taken on the dev machine); `CORS_ALLOWED_ORIGINS` allows the Astro dev port `4325`.
- `JWT_SECRET` in compose is a dev placeholder; production must inject it from a secret store.
- `create_db.go` reads `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` (defaults `localhost`, `5432`, `postgres`, empty, `kinetix-db`) and quotes the name with `%q`.

## Tasks
- [x] T1 Add multi-stage Dockerfile (api + seed + dbinit + goose) — test first: n/a (infrastructure), verified by `docker compose build` — files: `Dockerfile`
- [x] T2 Add docker-compose orchestration postgres/db-init/migrations/seed/api — test first: n/a (infrastructure), verified by smoke requests — files: `docker-compose.yml`
- [x] T3 Read `DB_*` env vars in `create_db.go` instead of hardcoded localhost — test first: n/a (script) — files: `scripts/create_db.go`

## Verification
```bash
docker compose build --no-cache api
docker compose up -d postgres db-init migrations seed api
curl -s http://localhost:8085/health                       # {"status":"ok",...}
curl -s http://localhost:8085/api/v1/events | head -c 200   # events data
curl -s -X POST http://localhost:8085/api/v1/auth/login -H "Content-Type: application/json" \
  -d '{"email":"contact@kinetix.com","password":"seed1234"}' # access_token + refresh_token
```
Acceptance: all 20 migrations applied at the time (now 23), seed finishes, health returns 200. Last run 2026-09-15.

## Risks and rollback
- Dev `JWT_SECRET` placeholder leaking to production: production compose/secret store must override it (see PLAN-15 B2).
- Rollback: `docker compose down -v` removes containers and the data volume.

## Open questions
- None.
