# ADR-6: JSON response and error envelope

- Status: Accepted
- Date: 2026-09-15
- Deciders: Kinetix team
- Related: ADR-2, ADR-5, docs/API_CONTRACTS.md, docs/PATTERNS.md

## Context
The Astro frontend consumes the API through typed actions and needs a predictable body shape for success and errors. Responses were written ad hoc in each slice. A shared helper package exists: `internal/json` (`Write`, `WriteError`, `Read`).

## Decision
- All handlers write JSON through `internal/json`: `json.Write(w, status, data)` sets `Content-Type: application/json`; `json.WriteError(w, status, msg)` writes `{"error": msg}`.
- Target envelope (API_CONTRACTS.md): success `{"data": ...}` (lists add `total`), errors `{"error": "...", "code": "MACHINE_CODE"}`; delete returns `204`.
- `500` responses never expose `err.Error()`; they are logged and return a generic message.
- `http.Error` and `fmt.Fprintf` to the writer are forbidden in handlers.
- Errors are mapped to HTTP once, in the handler, from sentinel errors (docs/PATTERNS.md).

## Consequences
### Positive
- One helper means consistent headers and a single place to evolve the format.
- Clients can branch on `code` instead of parsing messages.

### Negative / risks
- Current deviations (API_CONTRACTS.md): `internal/orders/handler.go` uses `http.Error` and returns `{message, data}`, and the webhook returns `err.Error()` on `500` (`internal/orders/handler.go:193-196`); auth login/register return the token object without `data`.
- `json.Write` ignores encoder errors (`internal/json/json.go`).
- `/health` does not set `Content-Type` (`cmd/api/main.go`).
- `internal/routes/handler.go` returns service errors as `400` with `err.Error()` for any failure, including internal ones.
- No `code` field or shared `WriteServiceError` exists yet; the target is in docs/PATTERNS.md.

## Alternatives considered
- RFC 7807 `application/problem+json`: rejected, heavier than the frontend needs; may be revisited with `/api/v2`.
- Raw objects without a `data` wrapper: rejected, no room for `total` or metadata.
- Per-slice response helpers: rejected, they already drifted.
