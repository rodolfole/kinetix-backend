# ADR-9: Docker image and compose local stack

- Status: Accepted
- Date: 2026-09-15
- Deciders: Kinetix team
- Related: ADR-3, ADR-8, plans/plan-1-docker-bootstrap.md, plans/plan-2-seed-verification.md

## Context
A new developer or the frontend repo needs the full API with data running by one command, without installing Go, Postgres or goose. The `Dockerfile` and `docker-compose.yml` were introduced for this (plan-1, plan-2).

## Decision
- Multi-stage `Dockerfile`: `golang:1.25.5-alpine` builds static binaries (`CGO_ENABLED=0`, `-trimpath`, stripped) for `kinetix-api`, `kinetix-seed` and `kinetix-dbinit`; goose is installed at v3.22.1. The runtime stage is `alpine:3.20`, creates an `app` user (but see risks), and copies migrations to `/app/migrations`.
- `docker-compose.yml` starts, in dependency order: `postgres` (16-alpine, healthcheck) -> `db-init` (creates `kinetix-db`) -> `migrations` (`goose up`) -> `seed` -> `api`.
- The API is published on host port `8085` (container `8080`) and has a `/health` healthcheck.
- Data persists in the `kinetix-pgdata` volume.
- Configuration is injected through environment variables; the local stack uses development secrets only.

## Consequences
### Positive
- `docker compose up` yields a migrated, seeded API; the same image applies migrations in other environments.
- Static binaries keep the runtime image small.

### Negative / risks
- The `runtime` stage never switches to the `app` user (no `USER` instruction in `Dockerfile`), so the container runs as root.
- `payouts-cron` is not built into the image (ADR-8).
- `docker-compose.yml` sets `CORS_ALLOWED_ORIGINS`, `MP_SANDBOX` and `WEBHOOK_BASE_URL`, which nothing reads; CORS is hardcoded to `http://localhost:4322` (`internal/middleware/middleware.go`) while compose maps the frontend at `4325`.
- Credentials (`postgres/postgres`, `dev-secret-change-in-production`) are in the compose file and as `Dockerfile` ENV defaults; they must not be reused outside local.
- `postgres` publishes port `5432` on the host.

## Alternatives considered
- Documented manual setup (local Go + Postgres): rejected, slow onboarding and version drift.
- Dev containers only: rejected, they do not help running the API for the frontend.
- Kubernetes manifests now: rejected, no deployment target is defined yet.
