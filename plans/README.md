# Plans

Plans describe how a change gets built: goal, scope, tasks (test first), verification and risks. Specs (`docs/specs/`) say what the system must do; ADRs (`docs/adrs/`) record why a decision was taken. A plan links to both and never duplicates their content.

## Naming

`PLAN-<N>-<kebab-slug>.md`, where `N` is sequential and never reused (an abandoned plan keeps its number). Specs are `docs/specs/SPEC-<feature>.md` and reference their plan in the header (`- Plan: plans/PLAN-<N>-....md`).

## Lifecycle

```
Draft -> Approved -> In progress -> Done
                 \-> Abandoned (from any state, keep the file and note the reason)
```

- Draft: the author edits freely.
- Approved: a reviewer other than the author accepted scope, design, verification and tasks.
- In progress: the first task is started.
- Done: every task is checked and the Definition of Done holds.

## Definition of Done

1. `gofmt -l .` prints nothing.
2. `go vet ./...` is clean.
3. `go test -race -count=1 ./...` passes.
4. Tests were written first (RED observed, then GREEN).
5. `docs/API_CONTRACTS.md` is updated for any endpoint, payload or error change.
6. Any migration has a working `Down` (verified with goose up and down).

## Template

```markdown
# PLAN-<N>: <Title>

- Status: Draft | Approved | In progress | Done | Abandoned
- Created: <YYYY-MM-DD> | Updated: <YYYY-MM-DD>
- Spec: docs/specs/SPEC-<x>.md
- ADRs: ADR-x, ADR-y

## Goal
<1-3 lines, outcome not activity>

## Scope
### In
### Out

## Design
<short; decisions belong in ADRs, link them>

## Tasks
- [ ] T1 <imperative> — test first: <test name/file> — files: <paths>

## Verification
<commands and acceptance criteria>

## Risks and rollback

## Open questions
```

Check a task box only when the code shows it is done.

## Index

| N | Title | Status | Spec | ADRs |
|---|-------|--------|------|------|
| 1 | [Docker bootstrap](PLAN-1-docker-bootstrap.md) | Done | none | ADR-9 |
| 2 | [Seed verification](PLAN-2-seed-verification.md) | Done | none | ADR-9 |
| 3 | [Auth hardening](PLAN-3-auth.md) | Draft | [SPEC-auth](../docs/specs/SPEC-auth.md) | ADR-4, ADR-11, ADR-6 |
| 4 | [Organizers](PLAN-4-organizers.md) | Draft | [SPEC-organizers](../docs/specs/SPEC-organizers.md) | ADR-6, ADR-5 |
| 5 | [Events](PLAN-5-events.md) | Draft | [SPEC-events](../docs/specs/SPEC-events.md) | ADR-13, ADR-1 |
| 6 | [Distances](PLAN-6-distances.md) | Draft | [SPEC-distances](../docs/specs/SPEC-distances.md) | ADR-13 |
| 7 | [Age categories](PLAN-7-age-categories.md) | Draft | [SPEC-agecategories](../docs/specs/SPEC-agecategories.md) | ADR-1 |
| 8 | [Tags](PLAN-8-tags.md) | Draft | [SPEC-tags](../docs/specs/SPEC-tags.md) | ADR-1 |
| 9 | [Runner kits](PLAN-9-runner-kits.md) | Draft | [SPEC-runnerkits](../docs/specs/SPEC-runnerkits.md) | ADR-6 |
| 10 | [Routes](PLAN-10-routes.md) | In progress | [SPEC-routes](../docs/specs/SPEC-routes.md) | ADR-10 |
| 11 | [Participants](PLAN-11-participants.md) | Draft | [SPEC-participants](../docs/specs/SPEC-participants.md) | ADR-6 |
| 12 | [Registrations](PLAN-12-registrations.md) | Draft | [SPEC-registrations](../docs/specs/SPEC-registrations.md) | ADR-14 |
| 13 | [Orders](PLAN-13-orders.md) | In progress | [SPEC-orders](../docs/specs/SPEC-orders.md) | ADR-7, ADR-12 |
| 14 | [Platform payments](PLAN-14-platform-payments.md) | Done | [SPEC-platform-payments](../docs/specs/SPEC-platform-payments.md) | ADR-7, ADR-8, ADR-12 |
| 15 | [Strict TDD and bug fixes](PLAN-15-strict-tdd-and-bug-fixes.md) | Approved | auth, orders, platform-payments, events | ADR-4, ADR-5, ADR-7, ADR-11, ADR-12, ADR-14 |
| 16 | [Platform hardening and completion](PLAN-16-platform-hardening.md) | Approved | platform-payments, orders, events | ADR-6, ADR-7, ADR-12, ADR-14 |
