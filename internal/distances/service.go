package distances

import (
	"context"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	db "kinetix-api/internal/adapters/postgresql/sqlc"
)

// Service handles business logic for distances
type Service struct {
	repo      *Repository
	validator *validator.Validate
}

// NewService creates a new distances service
func NewService(repo *Repository, validator *validator.Validate) *Service {
	return &Service{
		repo:      repo,
		validator: validator,
	}
}

// ── Distance Types ─────────────────────────────────────────────────────────

// ListTypes returns all distance types from the catalog
func (s *Service) ListTypes(ctx context.Context) ([]db.DistanceType, error) {
	types, err := s.repo.ListTypes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list distance types: %w", err)
	}
	return types, nil
}

// GetTypeByID retrieves a distance type by ID
func (s *Service) GetTypeByID(ctx context.Context, id uuid.UUID) (db.DistanceType, error) {
	if id == uuid.Nil {
		return db.DistanceType{}, fmt.Errorf("invalid distance type ID")
	}

	dt, err := s.repo.GetTypeByID(ctx, id)
	if err != nil {
		return db.DistanceType{}, err
	}
	return dt, nil
}

// ── CRUD Distances ─────────────────────────────────────────────────────────

// Create creates a new distance
func (s *Service) Create(ctx context.Context, req CreateDistanceRequest) (db.Distance, error) {
	if err := s.validator.Struct(req); err != nil {
		return db.Distance{}, fmt.Errorf("validation error: %w", err)
	}

	// Verify distance type exists
	if _, err := s.repo.GetTypeByID(ctx, req.DistanceTypeID); err != nil {
		return db.Distance{}, fmt.Errorf("distance type not found: %w", err)
	}

	dist, err := s.repo.Create(ctx, req)
	if err != nil {
		return db.Distance{}, fmt.Errorf("failed to create distance: %w", err)
	}

	return dist, nil
}

// GetByID retrieves a distance by ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (db.Distance, error) {
	if id == uuid.Nil {
		return db.Distance{}, fmt.Errorf("invalid distance ID")
	}

	dist, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return db.Distance{}, err
	}

	return dist, nil
}

// ListByEvent retrieves all distances for an event
func (s *Service) ListByEvent(ctx context.Context, eventID uuid.UUID) ([]db.Distance, error) {
	if eventID == uuid.Nil {
		return nil, fmt.Errorf("invalid event ID")
	}

	distances, err := s.repo.ListByEvent(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to list distances: %w", err)
	}

	return distances, nil
}

// Update updates a distance
func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdateDistanceRequest) (db.Distance, error) {
	if id == uuid.Nil {
		return db.Distance{}, fmt.Errorf("invalid distance ID")
	}

	if err := s.validator.Struct(req); err != nil {
		return db.Distance{}, fmt.Errorf("validation error: %w", err)
	}

	// Verify distance type exists
	if _, err := s.repo.GetTypeByID(ctx, req.DistanceTypeID); err != nil {
		return db.Distance{}, fmt.Errorf("distance type not found: %w", err)
	}

	updated, err := s.repo.Update(ctx, id, req)
	if err != nil {
		return db.Distance{}, fmt.Errorf("failed to update distance: %w", err)
	}

	return updated, nil
}

// Delete removes a distance
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return fmt.Errorf("invalid distance ID")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete distance: %w", err)
	}

	return nil
}

// ListByEventWithTypes retrieves all distances for an event enriched with type info
func (s *Service) ListByEventWithTypes(ctx context.Context, eventID uuid.UUID) ([]DistanceResponse, error) {
	if eventID == uuid.Nil {
		return nil, fmt.Errorf("invalid event ID")
	}

	distances, err := s.repo.ListByEvent(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to list distances: %w", err)
	}

	result := make([]DistanceResponse, len(distances))
	for i, d := range distances {
		var dt *db.DistanceType
		if d.DistanceTypeID.Valid {
			typeInfo, err := s.repo.GetTypeByID(ctx, d.DistanceTypeID.Bytes)
			if err == nil {
				dt = &typeInfo
			}
		}
		result[i] = toDistanceResponse(d, dt)
	}

	return result, nil
}
