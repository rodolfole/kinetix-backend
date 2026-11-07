# Testing

## Workflow: strict TDD

1. **RED**: write one failing test for one rule. Run it and see it fail for the expected reason.
2. **GREEN**: write the minimum production code to pass it.
3. **REFACTOR**: clean up with all tests green.

A bug fix starts with a test that reproduces the bug. A PR that changes behavior without a test that would have failed before is not mergeable.

## Levels

| Level | Scope | Location | Dependencies | Run |
|---|---|---|---|---|
| Unit | Service rules, pure functions (`routeparser`, payout math, JWT) | `internal/<pkg>/*_test.go` | Fakes of the consumer-declared interfaces | `go test ./...` |
| Handler | Routing, decoding, validation, status codes, envelopes | `internal/<pkg>/handler_test.go` | `httptest` + fake service | `go test ./...` |
| Repository | SQL behavior, constraints | `internal/<pkg>/repository_test.go` | Real PostgreSQL (`//go:build integration`) | `go test -tags=integration ./...` |
| End-to-end API | Full flows against a running stack | `internal/integration_test` (`//go:build integration`) | `docker compose up`, seeded DB | `go test -tags=integration ./internal/integration_test` |

The default `go test ./...` must pass with no database or network. Anything that needs them carries the `integration` build tag.

## What a test must assert

- For handlers: status code, `Content-Type`, the exact JSON envelope, and that rejected input never reaches the service (fake records zero calls).
- For services: the returned value **and** the side effects (what was written, what was not).
- Every rule's boundaries and named error cases (empty, min, max, invalid format, not found, forbidden, duplicate).
- Through the public API of the package. Do not assert on unexported fields.

## Style

```go
func TestCalculatePayoutAmount(t *testing.T) {
	tests := []struct {
		name          string
		totalCents    int64
		commissionPct float64
		want          int64
	}{
		{"ten percent", 10000, 10, 9000},
		{"rounds down", 999, 10, 899},
		{"zero commission", 5000, 0, 5000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculatePayoutAmount(tt.totalCents, tt.commissionPct)
			if got != tt.want {
				t.Fatalf("got %d, want %d", got, tt.want)
			}
		})
	}
}
```

- Table-driven tests; subtest names describe the case, not the input.
- Test names: `Test<Unit>_<Behavior>` (`TestWebhook_DuplicatePaymentIsIgnored`).
- Fakes are hand-written structs in the test file (`fakeRepo`) recording calls. No mocking framework.
- Use `t.Helper()` in helpers, `t.Cleanup()` for teardown, `t.Parallel()` where there is no shared state.
- No `time.Sleep`; inject a clock. No dependence on test order.
- `t.Skip` is allowed only with a linked issue in the message.

## Commands

```bash
go test ./... -count=1 -race                      # unit + handler, required before every commit
go test ./... -coverprofile=cover.out && go tool cover -func=cover.out
go test -tags=integration ./... -count=1          # needs docker compose up
go vet ./... && gofmt -l .                        # must print nothing
```

## Coverage gates

| Package type | Minimum line coverage |
|---|---|
| Services, `auth`, `orders`, `admin`, `routeparser` | 80% |
| Handlers | 70% |
| Generated `sqlc`, `cmd/*` | excluded |

Coverage is a floor, not the goal: an assertion-free test that raises coverage is rejected in review.

## CI

Each pull request runs: `gofmt -l`, `go vet`, `go test -race ./...`, then the integration suite against a PostgreSQL service with all goose migrations applied (not a single SQL file).
