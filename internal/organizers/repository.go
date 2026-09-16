package organizers

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	repo "kinetix-api/internal/adapters/postgresql/sqlc"
	"kinetix-api/internal/database/convert"
	"kinetix-api/internal/store"
)

// Repository handles database operations for organizers
type Repository struct {
	store *store.Store
}

// NewRepository creates a new organizer repository
func NewRepository(s *store.Store) *Repository {
	return &Repository{store: s}
}

// Create inserts a new organizer into the database
func (r *Repository) Create(ctx context.Context, req CreateOrganizerRequest) (Organizer, error) {
	org, err := r.store.Queries.CreateOrganizer(ctx, repo.CreateOrganizerParams{
		BusinessName:   req.BusinessName,
		BrandName:      convert.ToPgText(req.BrandName),
		Rfc:            req.RFC,
		BillingZipCode: req.BillingZipCode,
		BillingState:   req.BillingState,
		BillingCity:    req.BillingCity,
		ContactEmail:   req.ContactEmail,
		ContactPhone:   req.ContactPhone,
	})
	if err != nil {
		return Organizer{}, fmt.Errorf("failed to create organizer: %w", err)
	}

	return org, nil
}

// GetByID retrieves an organizer by ID
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (Organizer, error) {
	org, err := r.store.Queries.GetOrganizer(ctx, id)
	if err != nil {
		return Organizer{}, fmt.Errorf("organizer not found: %w", err)
	}

	return org, nil
}

// GetByRFC retrieves an organizer by RFC
func (r *Repository) GetByRFC(ctx context.Context, rfc string) (Organizer, error) {
	org, err := r.store.Queries.GetOrganizerByRFC(ctx, rfc)
	if err != nil {
		return Organizer{}, fmt.Errorf("organizer not found: %w", err)
	}

	return org, nil
}

// List retrieves a paginated list of organizers
func (r *Repository) List(ctx context.Context, limit, offset int) ([]Organizer, int64, error) {
	orgs, err := r.store.Queries.ListOrganizers(ctx, repo.ListOrganizersParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list organizers: %w", err)
	}

	return orgs, int64(len(orgs)), nil
}

// Update updates an existing organizer
func (r *Repository) Update(ctx context.Context, id uuid.UUID, req UpdateOrganizerRequest) (Organizer, error) {
	org, err := r.store.Queries.UpdateOrganizer(ctx, repo.UpdateOrganizerParams{
		ID:             id,
		BusinessName:   req.BusinessName,
		BrandName:      convert.ToPgText(req.BrandName),
		BillingZipCode: req.BillingZipCode,
		BillingState:   req.BillingState,
		BillingCity:    req.BillingCity,
		ContactEmail:   req.ContactEmail,
		ContactPhone:   req.ContactPhone,
	})
	if err != nil {
		return Organizer{}, fmt.Errorf("failed to update organizer: %w", err)
	}

	return org, nil
}

// Delete removes an organizer from the database
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	err := r.store.Queries.DeleteOrganizer(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete organizer: %w", err)
	}

	return nil
}
