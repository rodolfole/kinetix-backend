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
// EventOrganizer returns the organizer owning the event (ownership support).
func (r *Repository) EventOrganizer(ctx context.Context, eventID uuid.UUID) (uuid.UUID, error) {
	e, err := r.store.Queries.GetEvent(ctx, eventID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("event not found: %w", err)
	}
	return e.OrganizerID, nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (db.EventTag, error) {
	return r.store.Queries.GetEventTag(ctx, id)
}

// EventOrganizerByTag resolves the organizer through the tag's event.
func (r *Repository) EventOrganizerByTag(ctx context.Context, tagID uuid.UUID) (uuid.UUID, error) {
	tag, err := r.GetByID(ctx, tagID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("tag not found: %w", err)
	}
	return r.EventOrganizer(ctx, tag.EventID)
}

func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	err := r.store.Queries.DeleteEventTag(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete event tag: %w", err)
	}
	return nil
}

// DeleteByNames removed — was unused
