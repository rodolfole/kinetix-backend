package tags

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	db "kinetix-api/internal/adapters/postgresql/sqlc"
	"kinetix-api/internal/store"
)

// Repository handles database operations for event tags
type Repository struct {
	store *store.Store
}

// NewRepository creates a new tags repository
func NewRepository(s *store.Store) *Repository {
	return &Repository{store: s}
}

// ListByEvent returns all tags for an event
func (r *Repository) ListByEvent(ctx context.Context, eventID uuid.UUID) ([]db.EventTag, error) {
	tags, err := r.store.Queries.ListEventTagsByEvent(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to list event tags: %w", err)
	}
	return tags, nil
}

// Create inserts a new tag for an event
func (r *Repository) Create(ctx context.Context, eventID uuid.UUID, name string) (db.EventTag, error) {
	tag, err := r.store.Queries.CreateEventTag(ctx, db.CreateEventTagParams{
		EventID: eventID,
		Name:    name,
	})
	if err != nil {
		return db.EventTag{}, fmt.Errorf("failed to create event tag: %w", err)
	}
	return tag, nil
}

// Delete removes a tag by ID
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	err := r.store.Queries.DeleteEventTag(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete event tag: %w", err)
	}
	return nil
}

// DeleteByNames removes tags by names for an event
func (r *Repository) DeleteByNames(ctx context.Context, eventID uuid.UUID, names []string) error {
	err := r.store.Queries.DeleteEventTagsByNames(ctx, db.DeleteEventTagsByNamesParams{
		EventID: eventID,
		Names:   names,
	})
	if err != nil {
		return fmt.Errorf("failed to delete event tags by names: %w", err)
	}
	return nil
}
