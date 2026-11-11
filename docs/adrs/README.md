# Architecture Decision Records

An ADR records one significant technical decision: the forces behind it, what was decided, and what it costs. ADRs keep the "why" next to the code so new contributors do not re-litigate settled choices and so known gaps are written down with the decision that caused them. They complement, and do not replace, [ARCHITECTURE.md](../ARCHITECTURE.md) and the other guides in `docs/`.

## Naming

`ADR-<N>-<kebab-slug>.md`, for example `ADR-3-postgresql-sqlc-goose.md`. `N` is sequential, starts at 1 and is never reused or renumbered, even if the ADR is deprecated. Links to an ADR use this file name.

## Lifecycle

```
Proposed -> Accepted -> Deprecated
                    \-> Superseded by ADR-N
```

- **Proposed**: described and under discussion; the code may not match yet.
- **Accepted**: in force; the code follows it or the gaps are listed under Consequences.
- **Deprecated**: no longer applies, with no replacement.
- **Superseded by ADR-N**: replaced by a newer ADR; keep the file, update `Status`.

An accepted ADR is not edited to change the decision. Write a new ADR that supersedes it. Fixing typos, links, or adding newly found gaps is fine.

## When to write one

Write an ADR when a decision is hard to reverse or affects several slices: a framework, datastore, auth scheme, money representation, transaction policy, deployment unit, or a convention every slice must follow. Do not write one for local choices inside a single slice.

## Template

```markdown
# ADR-<N>: <Title>

- Status: Accepted | Proposed
- Date: <YYYY-MM-DD>
- Deciders: Kinetix team
- Related: ADR-x, plans/..., docs/...

## Context
<forces and constraints, 3-8 lines, with path references>

## Decision
<what we do, imperative, concrete>

## Consequences
### Positive
### Negative / risks

## Alternatives considered
<2-3 bullets with why rejected>
```

## Index

| N | Title | Status | Date |
|---|---|---|---|
| 1 | [Vertical slice architecture](ADR-1-vertical-slice-architecture.md) | Accepted | 2026-09-15 |
| 2 | [chi router with stdlib handlers](ADR-2-chi-router-stdlib-handlers.md) | Accepted | 2026-09-15 |
| 3 | [PostgreSQL with sqlc and goose](ADR-3-postgresql-sqlc-goose.md) | Accepted | 2026-09-15 |
| 4 | [JWT authentication](ADR-4-jwt-authentication.md) | Accepted | 2026-09-15 |
| 5 | [Request validation with go-playground/validator](ADR-5-request-validation-go-playground.md) | Accepted | 2026-09-15 |
| 6 | [JSON response and error envelope](ADR-6-json-response-and-error-envelope.md) | Accepted | 2026-09-15 |
| 7 | [MercadoPago marketplace payments](ADR-7-mercadopago-marketplace-payments.md) | Accepted | 2026-11-06 |
| 8 | [Payouts cron as a separate binary](ADR-8-payouts-cron-binary.md) | Accepted | 2026-11-06 |
| 9 | [Docker image and compose local stack](ADR-9-docker-compose-local-stack.md) | Accepted | 2026-09-15 |
| 10 | [GPX route parsing on the server](ADR-10-gpx-route-parsing-server-side.md) | Accepted | 2026-09-15 |
| 11 | [In-memory rate limit on auth endpoints](ADR-11-in-memory-auth-rate-limit.md) | Accepted | 2026-09-15 |
| 12 | [Money as integer cents](ADR-12-money-as-integer-cents.md) | Proposed | 2026-11-06 |
| 13 | [Keep the events package unsplit](ADR-13-keep-events-package-unsplit.md) | Accepted | 2026-11-06 |
| 14 | [Store transactions with ExecTx](ADR-14-store-transactions-exectx.md) | Proposed | 2026-11-06 |
