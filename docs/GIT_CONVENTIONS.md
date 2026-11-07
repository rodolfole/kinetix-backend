# Git Conventions

## Branches

- `main` is always deployable and protected: no direct pushes, merge only through a reviewed PR with green CI.
- Work branches: `<type>/<short-kebab-description>`, optionally with the issue number.
  - `feat/registration-bib-sequence`, `fix/142-webhook-idempotency`, `docs/adr-money-cents`, `chore/ci-goose-migrations`.
- One branch per change. Rebase on `main` before review; delete after merge.

## Commits: Conventional Commits

```
<type>(<scope>): <imperative summary, max 72 chars>

<optional body: why, not what>

<optional footer: Refs #142 | BREAKING CHANGE: ...>
```

| Type | Use |
|---|---|
| `feat` | New behavior |
| `fix` | Bug fix |
| `refactor` | Behavior unchanged |
| `test` | Tests only |
| `docs` | Documentation only |
| `chore` | Tooling, deps, config |
| `ci` | Pipeline |
| `perf` | Performance |
| `build` | Docker, build scripts |

- Scope is the slice or area: `auth`, `events`, `orders`, `admin`, `db`, `mp`, `middleware`, `ci`.
- Tests and docs go in the same commit as the behavior they cover.
- Each commit compiles and passes `go test ./...`.
- A migration and the `sqlc generate` output it requires are one commit.
- `BREAKING CHANGE:` footer for any change listed as breaking in [API_CONTRACTS.md](API_CONTRACTS.md#versioning).

## Pull requests

- Target size: under ~400 changed lines (generated sqlc code excluded). Split larger work into stacked PRs.
- Title follows the commit format. Description: what, why, how to test, risk, linked issue/plan/ADR.
- Checklist before requesting review:
  - [ ] `gofmt -l .` and `go vet ./...` clean
  - [ ] `go test -race ./...` green, new behavior has tests written first
  - [ ] `docs/API_CONTRACTS.md` updated if an endpoint changed
  - [ ] migration has a working `Down`
  - [ ] new env vars in `.env.example`
- Merge strategy: squash merge; the squash message is a Conventional Commit.
- At least one approval. Changes to `auth`, `middleware`, `orders`, `admin`, `mp` or migrations need a second reviewer.

## Releases

- Semantic versioning tags on `main`: `vMAJOR.MINOR.PATCH`.
- Changelog generated from Conventional Commits.

## Repository hygiene

- Never commit: binaries (`*.exe`, `bin/`), `.env`, credentials, real user data, coverage output.
- Seed credentials for local use live in `.env.example` comments or the README, never in a tracked `USERS.md` with real passwords.
- Line endings are normalized by `.gitattributes`:

```
* text=auto eol=lf
*.exe binary
*.png binary
```

  Windows contributors set `git config core.autocrlf false`.
- Secrets that reach a commit are rotated, not just deleted.
