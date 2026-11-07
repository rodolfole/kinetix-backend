# Backend code audit

Report only: nothing was deleted, moved or staged. Date: 2026-11-06. Scope: `backend/` (Go, chi, sqlc, goose). Each item was checked with `grep`/`find`/`git ls-files`/`git status --short`.

Status legend: **tracked** = in `git ls-files`; **untracked** = shown by `git status` as `??`; **ignored** = matched by `.gitignore`. The working tree has many pending user changes (modified handlers, new `cmd/payouts-cron/`, `internal/admin/`, `internal/mp/`, migrations 021-023); those are left alone.

## Unnecessary files

| Item | Evidence (path:line) | Why | Action (Delete / Merge / Move / Rename / Keep) | Risk |
|---|---|---|---|---|
| `api.exe`, `payouts-cron.exe`, `seed.exe` (15-22 MB each) | `.gitignore:2` (`*.exe`); `git ls-files --error-unmatch api.exe` fails (ignored, not tracked) | Local Windows build output; rebuildable with `go build` | Delete (local only) | Low |
| `bin/kinetix-api.exe` (20 MB) | `.gitignore:3` (`bin/`); ignored, not tracked | Same as above | Delete (local only) | Low |
| `internal/database/queries/*.sql` (11 files: award, distance, event, order, organizer, participant, pricing_stage, registration, registration_location, runner_kits, sponsor) | tracked, shown as ` D` in `git status`; `sqlc.yaml:5` now points only at `internal/adapters/postgresql/sqlc/queries.sql`; no non-doc reference to `internal/database/queries` (grep) | Superseded by the single sqlc query file; the deletion is already pending in the working tree | Delete (finish the pending deletion, user's own commit) | Low |
| `scripts/fix_handler_errors.py` | tracked; `scripts/fix_handler_errors.py:1` "Bulk-replace string-concat http.Error pattern"; no reference anywhere | One-off codemod already applied (the pattern is gone from the listed files) | Delete | Low |
| `test.sh` | tracked; `test.sh:9` runs only `./internal/auth` and `./internal/integration_test`; no reference except itself | Stale: ignores every other package with tests (e.g. `internal/middleware`, `internal/json`, `internal/routeparser`, which CI runs), README documents `go test ./...` (`README.md:252`) | Delete (or Rewrite to `go test ./...`; Keep only if a wrapper is wanted) | Low |
| `QUICKSTART.md` (226 lines) | tracked; overlaps `README.md` "Getting Started" (`README.md:51`, `:85`); `QUICKSTART.md:148` says "11 migration files" (23 exist) | Second, outdated setup guide | Merge (unique bits into README) then Delete | Low |
| `TODOS.md` (18 lines) | tracked, modified; content is a Spanish scratch note plus goose/netstat commands | Personal scratch notes, not a backlog | Move (to an issue or `docs/`) or Delete | Low |
| `USERS.md` (6 lines) | tracked, modified; holds seed credentials (values deliberately not copied here) | Credentials committed in git history (`b2b93f5`); seed passwords belong in `cmd/seed` output or `.env.example` | Delete from the index and rotate if reused outside local dev | Medium (history keeps it) |
| `scripts/create_db.go` | tracked; no reference besides itself (`go run` by hand) | Probably superseded by `docker-compose.yml` (`POSTGRES_DB`); not verified with the owner | Keep (verify with the owner; low value) | Low |
| `plans/`, `docs/specs/` | tracked; part of this baseline work | Not unnecessary | Keep | Low |

## Duplicated code

| Item | Evidence (path:line) | Why | Action (Delete / Merge / Move / Rename / Keep) | Risk |
|---|---|---|---|---|
| `auth.Config` built twice (`jwtCfg`, `authCfg`) | `cmd/api/main.go:69-73` and `:94-98` (identical, hardcoded 24h/7d) | Same struct twice; `cfg.JWT.TokenExpiry` is parsed in `internal/config/config.go:71` but never used | Merge (build once, reuse; optionally feed from config) | Low |
| `GET /events/{id}/field-config` registered twice | `cmd/api/main.go:182` (public) and `:228` (protected group, next to `PUT` at `:229`) | Second registration is shadowed by the public route; confusing intent | Delete one (keep public GET, keep PUT protected) | Low |
| `GET /distances/types` registered twice | `cmd/api/main.go:162` (public) and `internal/distances/handler.go:31` (`Routes()`, mounted at `main.go:198`) | Same handler on the same path inside the protected subrouter | Delete the `Routes()` line (`handler.go:30-31`) | Low |
| `uuid.Parse(chi.URLParam(...))` boilerplate | 48 occurrences in 8 handlers: events 25, distances 4, routes 4, runnerkits 4, orders 3, participants 3, registrations 3, tags 2 (`grep -rn "uuid.Parse(chi.URLParam" internal`) | Each repeats parse + 400 response | Merge into one helper, e.g. `httpx.URLParamUUID(r, "id")` | Low |
| `http.Error(` calls | 116 in 7 handlers: events 51, orders 18, registrations 13, participants 10, runnerkits 9, distances 8, organizers 7 | Parallel with `json.WriteError` in `internal/json/json.go:53`; mixed error shapes | Merge (route all errors through `json.WriteError`; extends the codemod already applied once) | Medium (response shape/headers could change; needs contract tests) |
| decode + validate + error sequence | 14 `json.NewDecoder`/`Decode(&` sites in `internal` vs helper `json.Read` (`internal/json/json.go:21`) | Handlers re-implement decode+validate | Merge into one `decodeAndValidate` helper | Medium |
| `splitPath` vs chi params | `internal/admin/handler.go:115`, `:174`, helper `:193`; `main.go:267,269` registers `{id}` routes through chi | Hand-parses a path chi already parsed (`chi.URLParam(r,"id")`) | Merge (use `chi.URLParam`, delete `splitPath`) | Low |
| Repo wrapper types in auth service | `internal/auth/service.go:43-48` (`organizerRepoImpl`, `participantRepoImpl` hold only `store`), used at `:118`, `:184` | Wrappers add nothing: both call `store.Queries` which `s.repo.store` already exposes (`:62`) | Delete the two types and fields, use `s.repo.store.Queries` | Low |
| Migration 020 re-adds `awards.place` | `migrations/004_create_awards.sql:9,17` already has the column and unique index; `020_add_awards_place_column.sql:14-17` repeats both with `IF NOT EXISTS` | Idempotent compat fix for an environment that missed 004 | Keep (history; never edit applied migrations), document in `docs/DATABASE.md` | Low |

## Dead code

| Item | Evidence (path:line) | Why | Action (Delete / Merge / Move / Rename / Keep) | Risk |
|---|---|---|---|---|
| `events.Handler.Routes()` | `internal/events/handler.go:28-29` defined; no call (`grep eventHandler.Routes cmd` is empty; `main.go` registers each path by hand, `:156-229`) | Unused router and drift risk against the real route table | Delete | Low |
| `admin.Handler.Routes()` | `internal/admin/handler.go:42-49`; `main.go:264-269` registers admin routes individually | Unused (nothing mounts it at `/admin/*`) | Delete | Low |
| `var _ = time.Time{}`, `var _ = pgtype.Numeric{}` | `internal/admin/handler.go:17-18` | Placeholders keeping unused imports alive | Delete lines and the two imports if unused | Low |
| `var _ = strings.TrimSpace` | `internal/auth/handler_test.go:218` | Same pattern in a test | Delete (and import if unused) | Low |
| `PaymentResponse` | `internal/orders/model.go:54`; no other Go reference | Unused type | Delete | Low |
| `OrganizerCreatedResponse`, `ParticipantCreatedResponse` | `internal/auth/model.go:51`, `:58`; no other Go reference | Unused types | Delete | Low |
| `GenericResponse` | no Go definition (grep over `*.go` is empty); only cited in `docs/specs/SPEC-*.md` | Spec refers to a type that does not exist | Rename in specs to the real shape (`json.Write`) | Low |
| `auth.Config.RefreshExpiry` | `internal/auth/auth.go:21`; set at `main.go:72,97`; never read (only `TokenExpiry` is used at `auth.go:30`, `service.go:99,165,227`) | No refresh-token expiry logic exists | Delete field, or implement refresh expiry | Low |
| `JWT_TOKEN_EXPIRY` | `internal/config/config.go:71`; `.env.example:13`; `docker-compose.yml:94`; `README.md:120` | Read into config, never consumed (hardcoded 24h in `main.go`) | Delete, or wire to `auth.Config` | Low |
| `CORS_ALLOWED_ORIGINS`, `MP_SANDBOX`, `WEBHOOK_BASE_URL` | `docker-compose.yml:95,98,99`; `.env.example:9,18,19`; no `getEnv` reads in `internal/config/config.go` | Env vars nothing reads; CORS is hardcoded (see `docs/ARCHITECTURE.md:81`) | Delete from compose/example, or implement (PLAN-15 T3 covers CORS) | Low |
| `GIN_MODE=release` | `Dockerfile:46` | Leftover from Gin; chi ignores it | Delete | Low |

## Stale documentation

| Item | Evidence (path:line) | Why | Action (Delete / Merge / Move / Rename / Keep) | Risk |
|---|---|---|---|---|
| README migration path | `README.md:87`, `:239` say `cd internal/database/migrations`; real dir is `internal/adapters/postgresql/migrations` | Commands fail as written | Rename path | Low |
| README project tree | `README.md:37` and structure block omit `admin`, `mp`, `routes`, `runnerkits`, `agecategories`, `cmd/payouts-cron` | Describes the repo before migrations 012-023 | Keep file, update | Low |
| `QUICKSTART.md` counts | `QUICKSTART.md:148` "11 migration files", `:7` checklist style status | Outdated snapshot | Delete (see Unnecessary files) | Low |
| `config.Load` comment vs code | `internal/config/config.go:49-52` says "no fallback in production" and panics on empty, but `getEnv` supplies a default secret, so the panic is unreachable | Misleading comment and a weak default secret | Keep, fix comment or remove the default (behavior change, needs a test) | Medium |
| `docs/specs/SPEC-*.md` `GenericResponse` | `SPEC-agecategories.md:35,45`, `SPEC-participants.md:53`, `SPEC-registrations.md:50`, `SPEC-routes.md:49` | Cites a non-existent Go type | Rename wording | Low |
| `.env.example` stale keys | `.env.example:9,13,18,19` (see Dead code) | Documents options that do nothing | Delete lines | Low |

## Recommended order

1. Delete local-only build outputs (`api.exe`, `payouts-cron.exe`, `seed.exe`, `bin/`): ignored, zero repo impact.
2. Delete `scripts/fix_handler_errors.py` and the placeholder lines `var _ = ...` (`admin/handler.go:17-18`, `auth/handler_test.go:218`).
3. Delete unused types (`PaymentResponse`, `OrganizerCreatedResponse`, `ParticipantCreatedResponse`) and `RefreshExpiry`; run `go build ./... && go vet ./...`.
4. Delete `events.Handler.Routes()` and `admin.Handler.Routes()`.
5. Remove duplicate route registrations (`main.go:228`, `distances/handler.go:31`) and merge `jwtCfg`/`authCfg` (`main.go:69-98`).
6. Remove dead env vars (`GIN_MODE`, `CORS_ALLOWED_ORIGINS`, `MP_SANDBOX`, `WEBHOOK_BASE_URL`, `JWT_TOKEN_EXPIRY`) from `Dockerfile`, compose, `.env.example`, README, or implement them (PLAN-15 T3).
7. Finish the pending deletion of `internal/database/queries/*.sql` (the user's own commit, not part of this branch).
8. Delete the `organizerRepoImpl`/`participantRepoImpl` wrappers; replace `splitPath` with `chi.URLParam`.
9. Fix README/specs (migration paths, tree, `GenericResponse`); merge `QUICKSTART.md` into README and delete it; delete or rewrite `test.sh`.
10. Handle `USERS.md` and `TODOS.md` (remove from index; rotate seed passwords if they were ever used outside local dev).
11. Introduce `URLParamUUID`/`decodeAndValidate` helpers, then migrate the 48 `uuid.Parse(chi.URLParam` and 116 `http.Error` sites one handler at a time, each behind a RED test (largest, behavior-adjacent change).
