# ADR-5: Request validation with go-playground/validator

- Status: Accepted
- Date: 2026-09-15
- Deciders: Kinetix team
- Related: ADR-1, ADR-6, docs/PATTERNS.md, docs/API_CONTRACTS.md

## Context
Every create/update endpoint receives JSON that must be checked (lengths, ranges, formats) before reaching SQL. The checks should be declared next to the types and be uniform across slices. Request structs already carry tags such as `validate:"required,min=1,max=50"` (`internal/events/model.go`).

## Decision
- Use `go-playground/validator/v10`; one `validator.New()` is created in `cmd/api/main.go` and injected into services and handlers.
- Declare rules with `validate:` struct tags on request types. Cross-field and business rules go in the service.
- Decode with `json.Read` (rejects unknown fields and trailing bytes) before validating (`internal/json/json.go`).
- Return validation failures as `400`.

## Consequences
### Positive
- Rules are declarative and live with the DTO; no hand-written checks per field.
- A shared validator instance caches struct metadata.

### Negative / risks
- Validator error text is returned verbatim as the `400` message, leaking Go field names; the target is `{"error":"validation failed","code":"VALIDATION_FAILED","fields":{...}}` (docs/API_CONTRACTS.md).
- Most services validate (`internal/events/service.go`, `internal/orders/service.go`, `internal/distances/service.go`), but `auth` and `admin` validate in the handler, and `routes`, `agecategories` and `tags` do not validate at all; there is no single rule on where it happens.
- Validation errors are wrapped with `fmt.Errorf("validation error: %w", ...)` (`internal/orders/service.go`) with no sentinel, so handlers cannot tell them from failures.
- `json.ReadAllowUnknown` exists but is only used by its own test (`internal/json/json_test.go`).

## Alternatives considered
- Manual `if` checks per handler: rejected, inconsistent and verbose.
- Validation via OpenAPI schema/middleware: rejected, there is no maintained OpenAPI spec to drive it.
- Validate only at the database (CHECK constraints): kept as a second guard, but it yields poor client errors.
