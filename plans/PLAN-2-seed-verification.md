# PLAN-2: Seed verification and smoke API

- Status: Done
- Created: 2026-09-14 | Updated: 2026-11-06
- Spec: none (operational check)
- ADRs: ADR-9

## Goal
After `docker compose up`, the seed provides enough data for the frontend and E2E tests, and three smoke requests prove the stack is healthy.

## Scope
### In
- Verify seed output: main organizer `contact@kinetix.com` plus mock organizers, 4 events (`Carrera Kinetix 2026`, `10K Valencia Marathon`, `THE NEON DASH`, `SUMMIT PEAK`) with distances, awards, pricing, sponsors and runner kits, 6 participants and 6 users (password `seed1234`), distance types 3K/5K/10K/21K/25K/42K.
- Smoke API: `GET /health`, `GET /api/v1/events`, `POST /api/v1/auth/login`.
### Out
- Unit tests for the seed (would need a mocked pgxpool).
- A `make seed-reset` target (use `docker compose down -v && docker compose up -d`).

## Design
- `cmd/seed/main.go` logs each step; the log is the verification artifact.
- The seed is idempotent: it looks up by `rfc` / `slug` before inserting, so `docker compose up -d seed` can run repeatedly without duplicating data.

## Tasks
- [x] T1 Document seed log expectations and smoke commands — test first: n/a (documentation of an operational check) — files: this plan
- [x] T2 Run smoke requests against the compose stack and record results — test first: n/a (manual smoke) — files: none

## Verification
```bash
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8085/health          # 200
curl -s http://localhost:8085/api/v1/events | grep -o '"id":"' | wc -l         # >= 4
curl -s -X POST http://localhost:8085/api/v1/auth/login -H "Content-Type: application/json" \
  -d '{"email":"contact@kinetix.com","password":"seed1234"}'                   # access_token present
```
Acceptance: Postgres healthy, migrations applied, seed re-runnable, JWT signed with the configured secret.

## Risks and rollback
- Seed credentials (`seed1234`) are for local use only; never run the seed against a shared or production database.
- Rollback: `docker compose down -v`.

## Open questions
- None.
