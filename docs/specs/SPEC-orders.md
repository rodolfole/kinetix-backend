# Spec: Orders Module

- Status: Implemented
- Plan: plans/PLAN-13-orders.md

## Objective

Track monetary orders attached to a registration. Orders move through
`pending → paid` (or other statuses) via internal updates and an external
MercadoPago webhook. The webhook is the integration boundary with the
payment provider — signature verification is mandatory.

## Tech Stack

- Go 1.22+, Chi v5 router
- PostgreSQL via pgx/v5 + sqlc
- `go-playground/validator/v10`
- HMAC-SHA256 for MercadoPago signature verification
- Raw `http.Error` in handler — same inconsistency as `participants` /
  `registrations`

## Commands

```bash
go build ./...
go test ./internal/orders/...
go run ./cmd/api
```

## Project Structure

```
internal/orders/
├── handler.go   # HTTP layer (CRUD + webhook)
├── service.go   # business logic + ProcessWebhook
├── order.go     # sqlc-backed Repository
├── webhook.go   # HMAC signature verification
└── model.go     # domain types + DTOs
```

## Endpoints

| Method | Path                              | Body                              | Returns            |
|--------|-----------------------------------|-----------------------------------|--------------------|
| POST   | `/api/v1/orders`                  | `CreateOrderRequest`              | `OrderResponse`    |
| GET    | `/api/v1/orders?event_id=…`       | —                                 | `OrdersListResponse` |
| GET    | `/api/v1/orders?participant_id=…` | —                                 | `OrdersListResponse` |
| GET    | `/api/v1/orders?status=…`         | —                                 | `OrdersListResponse` |
| GET    | `/api/v1/orders/{id}`             | —                                 | `OrderResponse`    |
| PUT    | `/api/v1/orders/{id}/payment`     | `UpdateOrderPaymentInfoRequest`   | `OrderResponse`    |
| PUT    | `/api/v1/orders/{id}/status`      | `{ "status": "paid" }`            | `OrderResponse`    |
| POST   | `/api/v1/orders/webhook`          | `PaymentWebhookRequest` (MP)      | `OrderResponse`    |

`POST /orders/webhook` is mounted **outside** the JWT-protected group
(per `cmd/api/main.go` line 257).

## Webhook signature flow

`webhook.go` validates the signature before the handler ever touches the
payload:

```go
// internal/orders/webhook.go
func ValidateMercadoPagoWebhook(webhookSecret, body, signatureHeader string) error {
    mac := hmac.New(sha256.New, []byte(webhookSecret))
    mac.Write([]byte(body))
    expected := hex.EncodeToString(mac.Sum(nil))
    if !hmac.Equal([]byte(signatureHeader), []byte(expected)) {
        return ErrInvalidSignature
    }
    return nil
}
```

Header order of precedence (from `ExtractSignatureFromRequest`):
`X-Merchant-Token` → `x-merchant-token` → `X-Signature` → `x-signature`.

The handler:
1. Reads the raw body (required so the signature matches exactly).
2. Calls `ValidateWebhookRequest(secret, r, body)`.
3. Maps `ErrMissingSignature` → 401, any other validation error → 403.
4. Parses the JSON, calls `service.ProcessWebhook(paymentID)`.

`Service.ProcessWebhook` looks up the order by MercadoPago payment ID and
flips status to `paid`. No partial-failure handling beyond that — see
Open Questions.

### Sample webhook payload

```json
{
  "action": "payment.created",
  "application_id": "12345",
  "data": { "id": "MP_PAYMENT_ID" },
  "type": "payment"
}
```

## Code Style

```go
// internal/orders/handler.go
body, err := io.ReadAll(r.Body)
if err != nil {
    http.Error(w, "failed to read request body", http.StatusBadRequest)
    return
}
if err := ValidateWebhookRequest(h.webhookSecret, r, body); err != nil {
    if err == ErrMissingSignature {
        http.Error(w, "missing webhook signature", http.StatusUnauthorized)
        return
    }
    http.Error(w, "invalid webhook signature", http.StatusForbidden)
    return
}
```

Repository converts sqlc numerics (`pgtype.Numeric`) to float64 via
`internal/database/convert` so the API surface is plain JSON numbers.

## Testing Strategy

- Integration tests belong in `internal/integration_test/integration_test.go`
  (currently absent for orders).
- Webhook tests should cover: missing signature → 401, bad signature → 403,
  valid signature with unknown payment ID → 404, valid signature with known
  payment ID → 200 + status flipped to `paid`.
- A signed-payload fixture must use the same secret as the test config.

## Boundaries

- **Always do:** Verify the webhook signature before parsing or acting.
- **Ask first:** Changing the HMAC algorithm, changing the signature header
  precedence, adding new payment providers, persisting raw webhook payloads.
- **Never do:**
  - Trust the `data.id` field without signature validation.
  - Skip the body-read step — the signature is over the raw bytes, not the
    parsed JSON.
  - Use `hmac.Equal` with `==`; the helper exists for timing-safe comparison.
  - Migrate to `internal/json` helpers without explicit scope.

## Success Criteria

- `POST /api/v1/orders` creates a pending order with a generated UUID.
- `GET /api/v1/orders/{id}` returns the order, 404 if missing.
- `PUT /api/v1/orders/{id}/status` updates and returns the order.
- `POST /api/v1/orders/webhook` with a valid HMAC and known payment ID
  flips status to `paid` and returns the updated order.
- `POST /api/v1/orders/webhook` without a signature returns 401.
- `POST /api/v1/orders/webhook` with a bad signature returns 403.

## Open Questions

- Should `ProcessWebhook` be transactional? Today the status update is a
  single SQL statement — fine — but if any side-effects (email, ticket
  generation) are added, wrap it.
- Should we persist the raw webhook body for audit / replay? Not done today.
- Should the webhook accept signed requests with `webhookSecret == ""`?
  Currently yes (returns `nil`); flagged as not recommended in the code
  comment. Production deployments must set the secret.
