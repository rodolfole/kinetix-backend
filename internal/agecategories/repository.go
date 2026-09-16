package agecategories

import (
	"context"
	"fmt"

	db "kinetix-api/internal/adapters/postgresql/sqlc"
	"kinetix-api/internal/database/convert"
	"kinetix-api/internal/store"
)

// Repository handles database operations for age categories
type Repository struct {
	store *store.Store
}

// NewRepository creates a new age categories repository
func NewRepository(s *store.Store) *Repository {
	return &Repository{store: s}
}

// List returns all age categories ordered by display_order
func (r *Repository) List(ctx context.Context) ([]db.AgeCategory, error) {
	items, err := r.store.Queries.ListAgeCategories(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list age categories: %w", err)
	}
	return items, nil
}

// toResponse converts a sqlc AgeCategory model to an API response
func toResponse(ac db.AgeCategory) AgeCategoryResponse {
	return AgeCategoryResponse{
		ID:           ac.ID,
		Name:         ac.Name,
		MinAge:       convert.PgInt4ToIntPtr(ac.MinAge),
		MaxAge:       convert.PgInt4ToIntPtr(ac.MaxAge),
		DisplayOrder: int(ac.DisplayOrder),
	}
}
