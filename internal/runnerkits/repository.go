package runnerkits

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	db "kinetix-api/internal/adapters/postgresql/sqlc"
	"kinetix-api/internal/database/convert"
	"kinetix-api/internal/store"
)

// Repository handles database operations for runner kit items
type Repository struct {
	store *store.Store
}

// NewRepository creates a new runner kit items repository
func NewRepository(s *store.Store) *Repository {
	return &Repository{store: s}
}

// Create inserts a new runner kit item
func (r *Repository) Create(ctx context.Context, req CreateRunnerKitItemRequest) (db.RunnerKitItem, error) {
	item, err := r.store.Queries.CreateRunnerKitItem(ctx, db.CreateRunnerKitItemParams{
		EventID:     req.EventID,
		Name:        req.Name,
		Icon:        req.Icon,
	})
	if err != nil {
		return db.RunnerKitItem{}, fmt.Errorf("failed to create runner kit item: %w", err)
	}
	return item, nil
}

// GetByID retrieves a runner kit item by ID
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (db.RunnerKitItem, error) {
	item, err := r.store.Queries.GetRunnerKitItem(ctx, id)
	if err != nil {
		return db.RunnerKitItem{}, fmt.Errorf("runner kit item not found: %w", err)
	}
	return item, nil
}

// ListByEvent retrieves all runner kit items for an event
func (r *Repository) ListByEvent(ctx context.Context, eventID uuid.UUID) ([]db.RunnerKitItem, error) {
	items, err := r.store.Queries.ListRunnerKitItemsByEvent(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to list runner kit items: %w", err)
	}
	return items, nil
}

// Update updates an existing runner kit item
func (r *Repository) Update(ctx context.Context, id uuid.UUID, req UpdateRunnerKitItemRequest) (db.RunnerKitItem, error) {
	item, err := r.store.Queries.UpdateRunnerKitItem(ctx, db.UpdateRunnerKitItemParams{
		ID:          id,
		Name:        req.Name,
		Icon:        req.Icon,
	})
	if err != nil {
		return db.RunnerKitItem{}, fmt.Errorf("failed to update runner kit item: %w", err)
	}
	return item, nil
}

// Delete removes a runner kit item
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	err := r.store.Queries.DeleteRunnerKitItem(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete runner kit item: %w", err)
	}
	return nil
}

// ── Converter ──────────────────────────────────────────────────────────────

// toResponse converts a sqlc RunnerKitItem model to an API response
func toResponse(item db.RunnerKitItem) RunnerKitItemResponse {
	return RunnerKitItemResponse{
		ID:          item.ID,
		EventID:     item.EventID,
		Name:        item.Name,
		Icon:        item.Icon,
		CreatedAt:   convert.PgTimeToTime(item.CreatedAt).Format("2006-01-02T15:04:05Z"),
	}
}
