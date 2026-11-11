# ADR-13: Keep the events package unsplit

- Status: Accepted
- Date: 2026-11-06
- Deciders: Kinetix team
- Related: ADR-1, plans/PLAN-5-events.md, docs/ARCHITECTURE.md

## Context
`internal/events` holds the event itself plus its sub-resources (awards, pricing stages, sponsors, registration locations, field config) and the full-detail composition behind `GET /events/{id}/detail`. plans/PLAN-5-events.md lists three problems: the package is large, `Distance` is defined in `internal/events/model.go` although `internal/distances` owns the behavior, and `GetFullEventDetail` issues one query per distance for awards and pricing (an N+1). The plan records the decision below.

## Decision
- Do not split `internal/events` into several slices yet. The sub-resources stay co-located so the detail composition is one round-trip and one package.
- Keep `Distance` in `internal/events/model.go`; moving it is a follow-up because it changes imports across the codebase.
- Accept the N+1 in `GetFullEventDetail`. Profile first and optimize (a join or `WHERE distance_id = ANY($1)`) only when a seeded event with many distances shows a measurable regression. Mark the loop with a `# ponytail:` comment naming the upgrade path (PLAN-5-events.md, Task 7).
- Revisit this ADR when the package gains a second owner or when a sub-resource needs an independent lifecycle.

## Consequences
### Positive
- No cross-package churn or new interfaces for a package that works.
- Detail composition stays simple and testable (`internal/events/handler_test.go`).

### Negative / risks
- `internal/events/service.go` and `handler.go` are large (many CRUD methods) and violate the "one slice per domain" intent of ADR-1.
- The routes are listed one by one in `cmd/api/main.go`, and `events.Handler.Routes()` is dead.
- Detail latency grows with the number of distances. The `# ponytail:` marker is not yet in `internal/events/repository.go` (`grep ponytail internal` returns nothing).
- `events` exposes `Distance` that `distances` also models, so the type is duplicated.

## Alternatives considered
- Split into `awards`, `pricing`, `sponsors`, `locations` slices now: rejected, large import churn, and the detail view would need cross-slice orchestration.
- Rewrite the detail query as a single join immediately: rejected, premature without measurement.
- Move `Distance` into `internal/distances` now: deferred, no behavior gain relative to the cost.
