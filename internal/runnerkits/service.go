package runnerkits

import (
	"context"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// Service handles business logic for runner kit items
type Service struct {
	repo      *Repository
	validator *validator.Validate
}

// NewService creates a new runner kit items service
func NewService(repo *Repository, validator *validator.Validate) *Service {
	return &Service{
		repo:      repo,
		validator: validator,
	}
}

// Create creates a new runner kit item
func (s *Service) Create(ctx context.Context, req CreateRunnerKitItemRequest) (RunnerKitItemResponse, error) {
	if err := s.validator.Struct(req); err != nil {
		return RunnerKitItemResponse{}, fmt.Errorf("validation error: %w", err)
	}

	item, err := s.repo.Create(ctx, req)
	if err != nil {
		return RunnerKitItemResponse{}, fmt.Errorf("failed to create runner kit item: %w", err)
	}

	return toResponse(item), nil
}

// GetByID retrieves a runner kit item by ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (RunnerKitItemResponse, error) {
	if id == uuid.Nil {
		return RunnerKitItemResponse{}, fmt.Errorf("invalid runner kit item ID")
	}

	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return RunnerKitItemResponse{}, err
	}

	return toResponse(item), nil
}

// ListByEvent retrieves all runner kit items for an event
func (s *Service) ListByEvent(ctx context.Context, eventID uuid.UUID) ([]RunnerKitItemResponse, error) {
	if eventID == uuid.Nil {
		return nil, fmt.Errorf("invalid event ID")
	}

	items, err := s.repo.ListByEvent(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to list runner kit items: %w", err)
	}

	result := make([]RunnerKitItemResponse, len(items))
	for i, item := range items {
		result[i] = toResponse(item)
	}

	return result, nil
}

// Update updates a runner kit item
func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdateRunnerKitItemRequest) (RunnerKitItemResponse, error) {
	if id == uuid.Nil {
		return RunnerKitItemResponse{}, fmt.Errorf("invalid runner kit item ID")
	}

	if err := s.validator.Struct(req); err != nil {
		return RunnerKitItemResponse{}, fmt.Errorf("validation error: %w", err)
	}

	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return RunnerKitItemResponse{}, fmt.Errorf("runner kit item not found")
	}

	item, err := s.repo.Update(ctx, id, req)
	if err != nil {
		return RunnerKitItemResponse{}, fmt.Errorf("failed to update runner kit item: %w", err)
	}

	return toResponse(item), nil
}

// Delete removes a runner kit item
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return fmt.Errorf("invalid runner kit item ID")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete runner kit item: %w", err)
	}

	return nil
}
