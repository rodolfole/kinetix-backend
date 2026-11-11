# Spec: Age Categories Module

- Status: Implemented
- Plan: plans/PLAN-7-age-categories.md

## Objective

Public, read-only catalog of age categories used by awards and registration
forms. Seeded into `age_categories` and exposed as a single GET.

Categories are ordered by `display_order` ascending and carry optional
`min_age` / `max_age` bounds (nullable so the catalog can also include
"general" categories without numeric bounds).

## Tech Stack

- Go 1.22+, Chi v5 router
- sqlc-generated queries via `kinetix-api/internal/store`
- `kinetix-api/internal/json` helpers
- `kinetix-api/internal/database/convert` for `pgtype.Int4`

## Commands

```bash
go build ./...
go test ./internal/integration_test/...        # TestListAgeCategories_Public
```

## Project Structure

```
internal/agecategories/
├── handler.go      # one endpoint, public read
├── repository.go   # List() + toResponse converter
└── dto.go          # AgeCategoryResponse + GenericResponse
```

Three files, one endpoint, no service layer — the module is a thin pass-
through to the repository.

## Endpoints

| Method | Path                       | Auth | Returns                          |
|--------|----------------------------|------|----------------------------------|
| GET    | `/api/v1/age-categories`   | none | `GenericResponse{Data: [...]}`   |

No write endpoints. The catalog is seed-managed.

### Response shape

```json
GET /api/v1/age-categories
{
  "message": "Age categories retrieved successfully",
  "data": [
    { "id": "...", "name": "Sub-18",   "min_age": 0,  "max_age": 17, "display_order": 1 },
    { "id": "...", "name": "18-29",    "min_age": 18, "max_age": 29, "display_order": 2 },
    { "id": "...", "name": "General",  "min_age": null, "max_age": null, "display_order": 99 }
  ]
}
```

`MinAge` / `MaxAge` are `*int` (omit when the column is NULL).

## Code Style

```go
// internal/agecategories/repository.go
func (r *Repository) List(ctx context.Context) ([]db.AgeCategory, error) {
    items, err := r.store.Queries.ListAgeCategories(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to list age categories: %w", err)
    }
    return items, nil
}

func toResponse(ac db.AgeCategory) AgeCategoryResponse {
    return AgeCategoryResponse{
        ID:           ac.ID,
        Name:         ac.Name,
        MinAge:       convert.PgInt4ToIntPtr(ac.MinAge),
        MaxAge:       convert.PgInt4ToIntPtr(ac.MaxAge),
        DisplayOrder: int(ac.DisplayOrder),
    }
}
```

The handler is the same shape as `distances.ListTypes`: load → convert →
wrap in `GenericResponse`. No validation layer needed — the data is read-
only and seed-managed.

## Testing Strategy

- `internal/integration_test/integration_test.go::TestListAgeCategories_Public`
  asserts the endpoint returns 200 (and currently does not assert the
  shape — see PLAN).
- No unit tests in the package.

## Boundaries

- **Always do:** Order results by `display_order` ascending in the
  underlying sqlc query; convert nullable age columns through
  `convert.PgInt4ToIntPtr`.
- **Ask first:** Adding write endpoints; introducing a `gender` bound;
  localising category names.
- **Never do:**
  - Add caching here — the row count is tiny (typically <10).
  - Change the response shape without a frontend coordination pass; today
    the response is consumed directly by the public event detail page.

## Success Criteria

- `GET /age-categories` returns 200 without a token.
- Response `data` is ordered by `display_order` ascending.
- Each entry includes `id`, `name`, `display_order`; `min_age` /
  `max_age` are present when not NULL and omitted otherwise.
- `go test ./internal/integration_test/... -run ListAgeCategories` passes.

## Open Questions

- Should `gender` be added to `age_categories` so awards can be defined
  per (category, gender) cleanly without the dedicated `awards` table
  column?
- Should the catalog be available in multiple locales? Today `Name` is a
  single `text` column.
