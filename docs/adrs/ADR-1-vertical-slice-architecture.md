# ADR-1: Vertical slice architecture

- Status: Accepted
- Date: 2026-09-15
- Deciders: Kinetix team
- Related: ADR-2, ADR-13, docs/ARCHITECTURE.md, docs/PATTERNS.md, plans/PLAN-events.md

## Context
The API covers many small domains (events, distances, routes, orders, admin...). Each one changes mostly on its own, and the team is small. `internal/` already holds one package per domain (`internal/events`, `internal/orders`, `internal/runnerkits`, ...), each with `handler.go`, `service.go`, `repository.go` and `model.go`/`dto.go`. All wiring is manual in `cmd/api/main.go`.

## Decision
- Organize code by domain slice under `internal/<feature>`, not by technical layer.
- Every slice follows handler -> service -> repository -> `store.Queries`. Handlers never call repositories.
- A slice never imports another slice's handler or repository; cross-slice reads go through the other slice's service interface.
- `cmd/<binary>/main.go` is the only composition root: it builds repositories, services and handlers by hand (no DI framework).
- Shared technical code lives in small packages (`store`, `database`, `middleware`, `json`, `config`, `mp`, `routeparser`).

## Consequences
### Positive
- A feature change touches one directory; slices are testable with fakes (`internal/*/handler_test.go`).
- No framework magic; the dependency graph is readable in `cmd/api/main.go`.

### Negative / risks
- Slices are inconsistent: repositories named `participant.go`, `order.go`, `registration.go`; `agecategories` and `tags` have no service (`internal/agecategories`, `internal/features/tags`).
- `tags` sits in `internal/features/tags`, unlike every other slice.
- `orders.NewService` takes the concrete `*admin.Repository` (`internal/orders/service.go:17`), coupling two slices instead of using an interface.
- `cmd/api/main.go` grows with every slice and already builds two identical `auth.Config` values (`jwtCfg`, `authCfg`).

## Alternatives considered
- Layered packages (`handlers/`, `services/`, `repositories/`): rejected, every feature would touch three or more directories.
- Hexagonal with ports for all code: rejected as overhead for a CRUD-heavy API; interfaces are declared only where tests need them.
- DI container (wire/fx): rejected, manual wiring is short enough and explicit.
