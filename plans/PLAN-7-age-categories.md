# PLAN-7: Age categories test hardening

- Status: Draft
- Created: 2026-09-25 | Updated: 2026-11-06
- Spec: docs/specs/SPEC-agecategories.md
- ADRs: ADR-1

## Goal
The single public endpoint has a test that locks its shape, ordering and no-auth contract.

## Scope
### In
- Tighten `TestListAgeCategories_Public`, add a no-auth test.
### Out
- A service layer (one-line repository call; ADR-1 allows thin slices).
- Adding `gender` to the catalog (changes response shape).

## Design
- No cache: row count is tiny. `# ponytail: no cache; add when row count exceeds 100`.
- Response shape stays stable; the public event page consumes it directly.

## Tasks
- [ ] T1 Assert 200, non-empty `data`, each entry has `id`, `name`, `display_order`, ordered ascending — test first: tighten `TestListAgeCategories_Public` (today it checks only 200) — files: `internal/integration_test/integration_test.go`
- [ ] T2 Lock the no-auth contract — test first: `TestAgeCategories_NoAuthRequired` — files: `internal/integration_test/integration_test.go`
- [ ] T3 (optional) 60s in-memory cache, or mark skipped with a reason — test first: existing integration tests stay green — files: `internal/agecategories/handler.go`

## Verification
```bash
go test -race -count=1 ./internal/agecategories/... ./internal/integration_test/... -run AgeCategories
```
Acceptance: shape and ordering asserted; public event detail unaffected.

## Risks and rollback
- Do not assert "no extra fields" so benign column additions do not break the test.
- T3 risks stale reads if the catalog becomes editable.

## Open questions
- Will the catalog be editable at runtime (invalidates the no-cache decision)?
