package participants

import (
	"context"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// Service handles business logic for participants
type Service struct {
	repo      *Repository
	validator *validator.Validate
}

// NewService creates a new participants service
func NewService(repo *Repository, validator *validator.Validate) *Service {
	return &Service{
		repo:      repo,
		validator: validator,
	}
}

// Create creates a new participant
func (s *Service) Create(ctx context.Context, req CreateParticipantRequest) (Participant, error) {
	if err := s.validator.Struct(req); err != nil {
		return Participant{}, fmt.Errorf("validation error: %w", err)
	}

	return s.repo.Create(ctx, req)
}

// GetByID retrieves a participant by ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (Participant, error) {
	return s.repo.GetByID(ctx, id)
}

// GetByEmail retrieves a participant by email
func (s *Service) GetByEmail(ctx context.Context, email string) (Participant, error) {
	return s.repo.GetByEmail(ctx, email)
}

// List retrieves a paginated list of participants
func (s *Service) List(ctx context.Context, limit, offset int) ([]Participant, int64, error) {
	return s.repo.List(ctx, limit, offset)
}

// Update updates a participant
func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdateParticipantRequest) (Participant, error) {
	if err := s.validator.Struct(req); err != nil {
		return Participant{}, fmt.Errorf("validation error: %w", err)
	}

	return s.repo.Update(ctx, id, req)
}

// Delete removes a participant
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	return s.repo.Delete(ctx, id)
}

// UpdateTeamSize updates only team and size fields for a participant
func (s *Service) UpdateTeamSize(ctx context.Context, id uuid.UUID, req UpdateParticipantTeamSizeRequest) (Participant, error) {
	return s.repo.UpdateTeamSize(ctx, id, req)
}
