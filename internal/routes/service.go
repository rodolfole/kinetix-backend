package routes

import (
	"context"
	"fmt"
	"io"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	db "kinetix-api/internal/adapters/postgresql/sqlc"
	"kinetix-api/internal/database/convert"
	"kinetix-api/internal/routeparser"
)

// Service handles business logic for routes
type Service struct {
	repo      *Repository
	validator *validator.Validate
}

// NewService creates a new routes service
func NewService(repo *Repository, validator *validator.Validate) *Service {
	return &Service{
		repo:      repo,
		validator: validator,
	}
}

// UploadFromGPX parses a GPX file and creates a route
func (s *Service) UploadFromGPX(ctx context.Context, file io.Reader, req UploadRouteRequest) (RouteResponse, error) {
	// Parse GPX
	parsed, err := routeparser.ParseGPX(file)
	if err != nil {
		return RouteResponse{}, fmt.Errorf("failed to parse GPX file: %w", err)
	}

	// Use parsed name if not provided
	name := req.Name
	if name == "" && parsed.Name != "" {
		name = parsed.Name
	}
	if name == "" {
		name = "Route"
	}

	// Build distance_id (nullable)
	var distanceID pgtype.UUID
	if req.DistanceID != nil {
		distanceID = convert.ToPgUUID(*req.DistanceID)
	}

	// Create route in database
	params := db.CreateRouteParams{
		EventID:            req.EventID,
		DistanceID:         distanceID,
		Name:               name,
		Geojson:            parsed.GeoJSONLineString(),
		ElevationProfile:   parsed.ElevationProfile(),
		TotalDistance:      convert.ToPgNumeric(parsed.TotalDistanceKm),
		TotalElevationGain: convert.ToPgNumeric(parsed.TotalElevationGain),
		MinElevation:       convert.ToPgNumeric(parsed.MinElevation),
		MaxElevation:       convert.ToPgNumeric(parsed.MaxElevation),
	}

	route, err := s.repo.Create(ctx, params)
	if err != nil {
		return RouteResponse{}, fmt.Errorf("failed to save route: %w", err)
	}

	return toResponse(route), nil
}

// GetByID retrieves a route by ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (RouteResponse, error) {
	if id == uuid.Nil {
		return RouteResponse{}, fmt.Errorf("invalid route ID")
	}

	route, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return RouteResponse{}, err
	}

	return toResponse(route), nil
}

// ListByEvent retrieves all routes for an event (with GeoJSON and elevation profile)
func (s *Service) ListByEvent(ctx context.Context, eventID uuid.UUID) ([]RouteResponse, error) {
	if eventID == uuid.Nil {
		return nil, fmt.Errorf("invalid event ID")
	}

	routes, err := s.repo.ListByEvent(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to list routes: %w", err)
	}

	result := make([]RouteResponse, len(routes))
	for i, route := range routes {
		result[i] = toResponse(route)
	}

	return result, nil
}

// ListByDistance retrieves all routes for a distance (with GeoJSON and elevation profile)
func (s *Service) ListByDistance(ctx context.Context, distanceID uuid.UUID) ([]RouteResponse, error) {
	if distanceID == uuid.Nil {
		return nil, fmt.Errorf("invalid distance ID")
	}

	routes, err := s.repo.ListByDistance(ctx, distanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list routes by distance: %w", err)
	}

	result := make([]RouteResponse, len(routes))
	for i, route := range routes {
		result[i] = toResponse(route)
	}

	return result, nil
}

// Delete removes a route
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return fmt.Errorf("invalid route ID")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete route: %w", err)
	}

	return nil
}
