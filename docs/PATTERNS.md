# Patterns

How code in a slice is written. If existing code disagrees, new code follows this file and the old code is fixed when touched.

## Handler

A handler only translates HTTP. Every handler follows the same five steps.

```go
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.Read(r, &req); err != nil {                 // 1. decode
		json.WriteError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := h.validate.Struct(req); err != nil {             // 2. validate
		httpx.WriteValidation(w, err)
		return
	}
	claims := middleware.ClaimsFrom(r.Context())               // 3. identity
	out, err := h.service.Create(r.Context(), claims, req)     // 4. call service
	if err != nil {
		httpx.WriteServiceError(w, r, err)                     // 5. map error once
		return
	}
	json.Write(w, http.StatusCreated, map[string]any{"data": out})
}
```

Rules:
- Path IDs are parsed with one shared helper (`httpx.UUIDParam(r, "id")`), never re-implemented per handler.
- No SQL, no business rules, no `http.Error`, no `fmt.Fprintf` to the writer.
- Status codes and envelopes follow [API_CONTRACTS.md](API_CONTRACTS.md).

## Errors

- Services return sentinel or typed errors from a shared package:

```go
var (
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict")
	ErrForbidden    = errors.New("forbidden")
	ErrInvalidInput = errors.New("invalid input")
)
```

- Wrap with context: `fmt.Errorf("create event %s: %w", slug, apperr.ErrConflict)`.
- Compare with `errors.Is` / `errors.As`, never `==` or `switch err`.
- Repositories translate `pgx.ErrNoRows` to `ErrNotFound` and unique violations (`23505`) to `ErrConflict`.
- `WriteServiceError` is the only place that maps errors to status codes. Unknown errors become `500`, are logged with the request ID, and are never sent to the client.
- Never discard errors (`_, _ = repo.Create(...)`). If a side effect is best-effort, log the failure.

## Service

```go
type Repository interface {
	GetByID(ctx context.Context, id uuid.UUID) (Event, error)
	Create(ctx context.Context, p CreateParams) (Event, error)
}

type Service struct {
	repo  Repository
	clock func() time.Time
}

func NewService(repo Repository, clock func() time.Time) *Service
```

- Interfaces are declared by the consumer, kept small, and used for test fakes.
- Inject time, ID generation and HTTP clients so tests are deterministic.
- First parameter is always `ctx context.Context`. Pass it down to every query and HTTP call.
- Authorization (ownership) is checked here: `if event.OrganizerID != claims.OrganizerID { return ErrForbidden }`.

## Repository

- Thin wrapper over `store.Queries`. Converts between sqlc rows and domain models.
- `pgtype` conversions go through `internal/database/convert`, never inline.
- Multi-row writes go through `store.ExecTx` ([DATABASE.md](DATABASE.md#transactions)).

## DTO and validation

- Request/response types live in `dto.go`. Domain types live in `model.go`. Do not expose sqlc rows in responses.
- Validation is declarative via `validate:` tags; cross-field rules go in the service.
- JSON tags are `snake_case`. Optional fields are pointers with `omitempty`.

## Configuration

- All env vars are read in `internal/config` and passed down as typed structs.
- Required secrets (`JWT_SECRET`, `MP_WEBHOOK_SECRET` outside `APP_ENV=local`) have no default; startup fails if they are missing.
- Every new env var is added to `.env.example` and the README in the same change.

## Logging

- Use `log/slog` with a JSON handler. Always include `request_id`; include `user_id` when authenticated.
- Levels: `Error` for failed operations that need action, `Warn` for rejected input from external systems (bad webhook signature), `Info` for state transitions (order paid, payout sent).
- Never log passwords, tokens, full card or OAuth payloads.

## External calls

- Every client (`internal/mp`) has an explicit timeout, receives `ctx`, and returns typed errors.
- Calls with side effects (payouts) send an idempotency key.
- Retries only for idempotent requests, with backoff and a cap.

## Naming

- Packages: short, lowercase, singular domain noun (`events`, `orders`). No `utils`, `common`, `helpers` packages.
- Constructors: `NewX`. Interfaces: noun describing the role (`Repository`, `PaymentGateway`), no `I` prefix.
- Files: `handler.go`, `service.go`, `repository.go`, `model.go`, `dto.go`, plus `<thing>.go` for focused logic (`webhook.go`).
- Comments and identifiers in English.
