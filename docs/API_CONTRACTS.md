# API Contracts

Base URL: `/api/v1`. JSON over HTTP. This file is the contract the frontend codes against; change it in the same PR as the handler.

## Conventions

### Requests
- `Content-Type: application/json`, except `POST /routes/upload` (`multipart/form-data`).
- Unknown fields are rejected (`json.Read` uses `DisallowUnknownFields`). Trailing bytes are rejected.
- IDs are UUID v4 in the path. A malformed UUID returns `400`.
- Dates: `YYYY-MM-DD`. Timestamps: RFC 3339 UTC.
- Money: decimal number in MXN with 2 decimals (`"amount": 450.00`). Payouts use integer cents (`amount_cents`). See [ADR-12](adrs/ADR-12-money-as-integer-cents.md).

### Authentication
- `Authorization: Bearer <access_token>` on every protected route.
- Missing header, wrong scheme or invalid token: `401 {"error": "..."}`.
- Authenticated but wrong role: `403 {"error": "insufficient role"}`.

### Responses (target, enforce on every new or touched handler)

| Case | Status | Body |
|---|---|---|
| Read one | `200` | `{"data": {...}}` |
| Read list | `200` | `{"data": [...], "total": n}` |
| Create | `201` | `{"data": {...}}` |
| Update | `200` | `{"data": {...}}` |
| Delete | `204` | empty |
| Error | `4xx/5xx` | `{"error": "human message", "code": "MACHINE_CODE"}` |

Error codes: `VALIDATION_FAILED`, `INVALID_ID`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `CONFLICT`, `RATE_LIMITED`, `UPSTREAM_ERROR`, `INTERNAL`.

Rules:
- Never return `err.Error()` of a `500` or a database error to the client. Log it with the request ID, return `{"error":"internal error","code":"INTERNAL"}`.
- Validation errors list fields: `{"error":"validation failed","code":"VALIDATION_FAILED","fields":{"email":"required"}}`.
- Every response sets `Content-Type: application/json` (use `internal/json`). `http.Error` is forbidden in handlers.

### Current deviations (fix when touching the handler)
- `orders` handlers return plain-text errors via `http.Error`, `200` on create, and `{message, data}`.
- Auth `register`/`login` return the token object without `data`; `refresh` returns `{message, data}`.
- Validator text is returned verbatim as the `400` message.
- `/health` does not set `Content-Type`.

## Versioning
- Breaking changes (removed field, changed type, new required field, changed status code) need `/api/v2` or a deprecation window announced in the changelog.
- Adding an optional request field or a response field is not breaking.

## Endpoints

Legend: **P** public, **A** any authenticated user, **O** organizer (owner), **R** participant (owner), **AD** admin. "Target" is the access level to enforce; today every non-public route except `/admin/*` only checks **A**.

### System
| Method | Path | Access | Response |
|---|---|---|---|
| GET | `/health` (root, no `/api/v1`) | P | `200 {"status":"ok","timestamp"}` |

### Auth (`/auth`, rate limited 10 req/min per client)
| Method | Path | Access | Body | Response |
|---|---|---|---|---|
| POST | `/auth/login` | P | `{email, password}` | `200 {user, access_token, refresh_token, expires_in}`; `401` bad credentials; `403` inactive user |
| POST | `/auth/register/organizer` | P | `{email, password(min 8), business_name, brand_name, rfc(12-13), billing_zip_code, billing_state, billing_city, contact_name, contact_email, contact_phone, logo_url}` | `201` same as login; `409` email taken |
| POST | `/auth/register/participant` | P | `{email, password, first_name, last_name, second_last_name, gender, birth_date, phone}` | `201` same as login; `409` |
| POST | `/auth/refresh` | P | `{refresh_token}` | `200 {data: tokens}`; `401` invalid refresh token |

Access tokens last 24h. Refresh tokens must carry `typ: "refresh"` and be rejected as access tokens (and vice versa).

### Catalogs and public event data
| Method | Path | Access |
|---|---|---|
| GET | `/events` | P |
| GET | `/events/{id}` | P |
| GET | `/events/slug/{slug}` | P |
| GET | `/events/{id}/detail` | P (event + distances + pricing + awards + kits + sponsors) |
| GET | `/events/{id}/field-config` | P |
| GET | `/events/{eventId}/tags` | P |
| GET | `/events/event/{event_id}/distances` | P |
| GET | `/events/event/{event_id}/sponsors` | P |
| GET | `/events/event/{event_id}/locations` | P |
| GET | `/events/event/{event_id}/runner-kits` | P |
| GET | `/events/event/{event_id}/routes` | P |
| GET | `/events/distance/{distance_id}/pricing` | P |
| GET | `/events/distance/{distance_id}/pricing/current` | P |
| GET | `/events/routes/distance/{distance_id}` | P |
| GET | `/distances/types` | P |
| GET | `/age-categories` | P |
| GET | `/organizers/{id}` | P |

### Event management
| Method | Path | Target access |
|---|---|---|
| POST | `/events` | O |
| PUT / DELETE | `/events/{id}` | O |
| PATCH | `/events/{id}/status` | O. Body `{status}` in `draft, published, cancelled, completed` |
| PUT | `/events/{id}/field-config` | O |
| POST, GET/PUT/DELETE `/{id}` | `/events/awards`, `/events/pricing`, `/events/sponsors`, `/events/locations` | O |
| POST / DELETE `/{id}` | `/events/tags` | O |
| POST, GET/PUT/DELETE `/{id}`, GET `/event/{event_id}` | `/distances` | O |
| POST, GET/PUT/DELETE `/{id}`, GET `/event/{event_id}` | `/runner-kits` | O |
| POST `/upload` (multipart: `file`, `event_id`, `distance_id`), GET/DELETE `/{id}` | `/routes` | O |
| POST, GET, PUT/DELETE `/{id}` | `/organizers` | AD (create/list/delete), O (update self) |

Field-level request schemas are the `*Request` structs with `validate` tags in each slice's `dto.go` / `model.go`.

### Participants and registrations
| Method | Path | Target access |
|---|---|---|
| POST, GET | `/participants` | AD |
| GET/PUT/DELETE | `/participants/{id}` | R (self) or AD |
| POST | `/registrations` | R |
| POST | `/registrations/with-participant` | R |
| GET | `/registrations?event_id=` | O (event owner) |
| GET | `/registrations/participant/{participant_id}` | R (self) |
| GET/DELETE | `/registrations/{id}` | R (self) or O |

### Orders and payments
| Method | Path | Target access | Notes |
|---|---|---|---|
| POST | `/orders` | R | Body `{registration_id, participant_id, event_id, amount, currency}`. Target: server computes `amount` from the current pricing stage; client amount is ignored. |
| GET | `/orders?event_id=\|participant_id=\|status=` | O / R | At least one filter, else `400` |
| GET | `/orders/{id}` | O / R | |
| PUT | `/orders/{id}/payment`, `/orders/{id}/status` | AD | |
| POST | `/orders/webhook` | P (signed) | See below |

**Webhook** `POST /orders/webhook`
- Payload: `{action, application_id, type, data: {id}}`.
- Signature: HMAC-SHA256 with `MP_WEBHOOK_SECRET`. Missing signature `401`, bad signature `403`, empty `data.id` `400`.
- Must be idempotent: the same `data.id` processed twice changes nothing and returns `200`.
- Target: verify MercadoPago's `x-signature` (`ts=...,v1=...` over `id:<data.id>;request-id:<x-request-id>;ts:<ts>;`) and confirm the payment status with the MP payments API before marking the order `paid`.
- An empty `MP_WEBHOOK_SECRET` must fail startup outside local development, never skip verification.

### Admin (`RequireRole("admin")`)
| Method | Path | Body / query | Response |
|---|---|---|---|
| GET | `/admin/config` | | `{data: {commission_pct, updated_by, updated_at}}` |
| PUT | `/admin/config` | `{commission_pct 0..100, updated_by}` | same |
| GET | `/admin/payouts` | `?status=pending\|sent\|failed` | `{data: [...], total}` |
| GET | `/admin/organizers/{id}/connect-mp` | | `{authorize_url, state}` |
| POST | `/admin/organizers/callback` | `?code=&state=` | `{organizer_id, mp_user_id, expires_in}` |
| POST | `/admin/payouts/{id}/send` | | `{payout_id, status}`; `502` if MP fails |

OAuth `state` must be an unguessable, single-use value bound to the organizer, not the raw organizer UUID.
