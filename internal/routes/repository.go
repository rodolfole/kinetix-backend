package routes

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	db "kinetix-api/internal/adapters/postgresql/sqlc"
	"kinetix-api/internal/database/convert"
	"kinetix-api/internal/store"
)

// Repository handles database operations for routes
type Repository struct {
	store *store.Store
}

// NewRepository creates a new routes repository
func NewRepository(s *store.Store) *Repository {
	return &Repository{store: s}
}

// Create inserts a new route
func (r *Repository) Create(ctx context.Context, params db.CreateRouteParams) (db.Route, error) {
	route, err := r.store.Queries.CreateRoute(ctx, params)
	if err != nil {
		return db.Route{}, fmt.Errorf("failed to create route: %w", err)
	}
	return route, nil
}

// GetByID retrieves a route by ID
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (db.Route, error) {
	route, err := r.store.Queries.GetRoute(ctx, id)
	if err != nil {
		return db.Route{}, fmt.Errorf("route not found: %w", err)
	}
	return route, nil
}

// ListByEvent retrieves all routes for an event
func (r *Repository) ListByEvent(ctx context.Context, eventID uuid.UUID) ([]db.Route, error) {
	routes, err := r.store.Queries.ListRoutesByEvent(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to list routes: %w", err)
	}
	return routes, nil
}

// ListByDistance retrieves all routes for a distance
func (r *Repository) ListByDistance(ctx context.Context, distanceID uuid.UUID) ([]db.Route, error) {
	routes, err := r.store.Queries.ListRoutesByDistance(ctx, convert.ToPgUUID(distanceID))
	if err != nil {
		return nil, fmt.Errorf("failed to list routes by distance: %w", err)
	}
	return routes, nil
}

// Delete removes a route
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	err := r.store.Queries.DeleteRoute(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete route: %w", err)
	}
	return nil
}

// ── Converters ─────────────────────────────────────────────────────────────

// toResponse converts a sqlc Route model to an API response
func toResponse(route db.Route) RouteResponse {
	distanceID := convert.PgUUIDToPtr(route.DistanceID)

	var totalDistanceKm *float64
	if route.TotalDistance.Valid {
		v := convert.PgNumericToFloat64(route.TotalDistance)
		totalDistanceKm = &v
	}

	var totalElevationGain *float64
	if route.TotalElevationGain.Valid {
		v := convert.PgNumericToFloat64(route.TotalElevationGain)
		totalElevationGain = &v
	}

	var minElevation *float64
	if route.MinElevation.Valid {
		v := convert.PgNumericToFloat64(route.MinElevation)
		minElevation = &v
	}

	var maxElevation *float64
	if route.MaxElevation.Valid {
		v := convert.PgNumericToFloat64(route.MaxElevation)
		maxElevation = &v
	}

	return RouteResponse{
		ID:                 route.ID,
		EventID:            route.EventID,
		DistanceID:         distanceID,
		Name:               route.Name,
		GeoJSON:            json.RawMessage(route.Geojson),
		ElevationProfile:   json.RawMessage(route.ElevationProfile),
		TotalDistanceKm:    totalDistanceKm,
		TotalElevationGain: totalElevationGain,
		MinElevation:       minElevation,
		MaxElevation:       maxElevation,
		CreatedAt:          convert.PgTimeToTime(route.CreatedAt).Format("2006-01-02T15:04:05Z"),
	}
}

// toListItem converts a sqlc Route model to a list item response (without GeoJSON)
func toListItem(route db.Route) RouteListItem {
	distanceID := convert.PgUUIDToPtr(route.DistanceID)

	var totalDistanceKm *float64
	if route.TotalDistance.Valid {
		v := convert.PgNumericToFloat64(route.TotalDistance)
		totalDistanceKm = &v
	}

	var totalElevationGain *float64
	if route.TotalElevationGain.Valid {
		v := convert.PgNumericToFloat64(route.TotalElevationGain)
		totalElevationGain = &v
	}

	var minElevation *float64
	if route.MinElevation.Valid {
		v := convert.PgNumericToFloat64(route.MinElevation)
		minElevation = &v
	}

	var maxElevation *float64
	if route.MaxElevation.Valid {
		v := convert.PgNumericToFloat64(route.MaxElevation)
		maxElevation = &v
	}

	return RouteListItem{
		ID:                 route.ID,
		EventID:            route.EventID,
		DistanceID:         distanceID,
		Name:               route.Name,
		TotalDistanceKm:    totalDistanceKm,
		TotalElevationGain: totalElevationGain,
		MinElevation:       minElevation,
		MaxElevation:       maxElevation,
		CreatedAt:          convert.PgTimeToTime(route.CreatedAt).Format("2006-01-02T15:04:05Z"),
	}
}
