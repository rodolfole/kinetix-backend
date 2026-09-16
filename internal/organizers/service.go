package organizers

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/go-playground/validator/v10"
)

// Service handles business logic for organizers
type Service struct {
	repo      *Repository
	validator *validator.Validate
}

// NewService creates a new organizer service
func NewService(repo *Repository, validator *validator.Validate) *Service {
	return &Service{
		repo:      repo,
		validator: validator,
	}
}

// Create creates a new organizer
func (s *Service) Create(ctx context.Context, req CreateOrganizerRequest) (Organizer, error) {
	// Validate request
	if err := s.validator.Struct(req); err != nil {
		return Organizer{}, fmt.Errorf("validation error: %w", err)
	}

	// Check if RFC already exists
	existing, err := s.repo.GetByRFC(ctx, req.RFC)
	if err == nil && existing.ID != uuid.Nil {
		return Organizer{}, fmt.Errorf("organizer with RFC %s already exists", req.RFC)
	}

	// Create organizer
	org, err := s.repo.Create(ctx, req)
	if err != nil {
		return Organizer{}, fmt.Errorf("failed to create organizer: %w", err)
	}

	return org, nil
}

// GetByID retrieves an organizer by ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (Organizer, error) {
	if id == uuid.Nil {
		return Organizer{}, fmt.Errorf("invalid organizer ID")
	}

	org, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Organizer{}, fmt.Errorf("failed to get organizer: %w", err)
	}

	return org, nil
}

// List retrieves a paginated list of organizers
func (s *Service) List(ctx context.Context, limit, offset int) ([]Organizer, int64, error) {
	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	organizers, total, err := s.repo.List(ctx, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list organizers: %w", err)
	}

	return organizers, total, nil
}

// Update updates an existing organizer
func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdateOrganizerRequest) (Organizer, error) {
	if id == uuid.Nil {
		return Organizer{}, fmt.Errorf("invalid organizer ID")
	}

	// Validate request
	if err := s.validator.Struct(req); err != nil {
		return Organizer{}, fmt.Errorf("validation error: %w", err)
	}

	// Check if organizer exists
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Organizer{}, fmt.Errorf("organizer not found: %w", err)
	}

	// Update organizer
	updated, err := s.repo.Update(ctx, id, req)
	if err != nil {
		return Organizer{}, fmt.Errorf("failed to update organizer: %w", err)
	}

	return updated, nil
}

// Delete removes an organizer
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return fmt.Errorf("invalid organizer ID")
	}

	// Check if organizer exists
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("organizer not found: %w", err)
	}

	// Delete organizer
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete organizer: %w", err)
	}

	return nil
}
