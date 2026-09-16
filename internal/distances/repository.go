package distances

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	db "kinetix-api/internal/adapters/postgresql/sqlc"
	"kinetix-api/internal/database/convert"
	"kinetix-api/internal/store"
)

// Repository handles database operations for distances
type Repository struct {
	store *store.Store
}

// NewRepository creates a new distances repository
func NewRepository(s *store.Store) *Repository {
	return &Repository{store: s}
}

// ── Distance Types ─────────────────────────────────────────────────────────

// ListTypes returns all distance types from the catalog
func (r *Repository) ListTypes(ctx context.Context) ([]db.DistanceType, error) {
	types, err := r.store.Queries.ListDistanceTypes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list distance types: %w", err)
	}
	return types, nil
}

// GetTypeByID retrieves a distance type by ID
func (r *Repository) GetTypeByID(ctx context.Context, id uuid.UUID) (db.DistanceType, error) {
	dt, err := r.store.Queries.GetDistanceType(ctx, id)
	if err != nil {
		return db.DistanceType{}, fmt.Errorf("distance type not found: %w", err)
	}
	return dt, nil
}

// ── CRUD Distances ─────────────────────────────────────────────────────────

// Create inserts a new distance
func (r *Repository) Create(ctx context.Context, req CreateDistanceRequest) (db.Distance, error) {
	dist, err := r.store.Queries.CreateDistance(ctx, db.CreateDistanceParams{
		EventID:        req.EventID,
		DistanceTypeID: convert.ToPgUUID(req.DistanceTypeID),
		Km:             convert.ToPgNumeric(0), // will be overridden if needed
		Capacity:       int32(req.Capacity),
		Surface:        req.Surface,
		Elevation:      convert.ToPgInt4Ptr(req.Elevation),
		TimeLimit:      convert.ToPgText(req.TimeLimit),
		StartLat:       convert.ToPgNumericPtr(req.StartLat),
		StartLng:       convert.ToPgNumericPtr(req.StartLng),
		EndLat:         convert.ToPgNumericPtr(req.EndLat),
		EndLng:         convert.ToPgNumericPtr(req.EndLng),
	})
	if err != nil {
		return db.Distance{}, fmt.Errorf("failed to create distance: %w", err)
	}
	return dist, nil
}

// GetByID retrieves a distance by ID
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (db.Distance, error) {
	dist, err := r.store.Queries.GetDistance(ctx, id)
	if err != nil {
		return db.Distance{}, fmt.Errorf("distance not found: %w", err)
	}
	return dist, nil
}

// ListByEvent retrieves all distances for an event
func (r *Repository) ListByEvent(ctx context.Context, eventID uuid.UUID) ([]db.Distance, error) {
	distances, err := r.store.Queries.ListDistancesByEvent(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to list distances: %w", err)
	}
	return distances, nil
}

// Update updates an existing distance
func (r *Repository) Update(ctx context.Context, id uuid.UUID, req UpdateDistanceRequest) (db.Distance, error) {
	dist, err := r.store.Queries.UpdateDistance(ctx, db.UpdateDistanceParams{
		ID:             id,
		DistanceTypeID: convert.ToPgUUID(req.DistanceTypeID),
		Km:             convert.ToPgNumeric(0),
		Capacity:       int32(req.Capacity),
		Surface:        req.Surface,
		Elevation:      convert.ToPgInt4Ptr(req.Elevation),
		TimeLimit:      convert.ToPgText(req.TimeLimit),
		StartLat:       convert.ToPgNumericPtr(req.StartLat),
		StartLng:       convert.ToPgNumericPtr(req.StartLng),
		EndLat:         convert.ToPgNumericPtr(req.EndLat),
		EndLng:         convert.ToPgNumericPtr(req.EndLng),
	})
	if err != nil {
		return db.Distance{}, fmt.Errorf("failed to update distance: %w", err)
	}
	return dist, nil
}

// Delete removes a distance
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	err := r.store.Queries.DeleteDistance(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete distance: %w", err)
	}
	return nil
}

// ── Converters ─────────────────────────────────────────────────────────────

// toDistanceResponse converts a sqlc Distance model to an API response
func toDistanceResponse(d db.Distance, dt *db.DistanceType) DistanceResponse {
	km := convert.PgNumericToFloat64(d.Km)
	if dt != nil {
		km = convert.PgNumericToFloat64(dt.Km)
	}

	resp := DistanceResponse{
		ID:        d.ID,
		EventID:   d.EventID,
		Km:        km,
		Capacity:  int(d.Capacity),
		Surface:   d.Surface,
		Elevation: convert.PgInt4ToIntPtr(d.Elevation),
		TimeLimit: convert.PgTextToString(d.TimeLimit),
		StartLat:  convert.PgNumericToFloat64Ptr(d.StartLat),
		StartLng:  convert.PgNumericToFloat64Ptr(d.StartLng),
		EndLat:    convert.PgNumericToFloat64Ptr(d.EndLat),
		EndLng:    convert.PgNumericToFloat64Ptr(d.EndLng),
	}

	if d.DistanceTypeID.Valid {
		resp.DistanceTypeID = d.DistanceTypeID.Bytes
	}

	return resp
}

// toDistanceTypeResponse converts a sqlc DistanceType model to an API response
func toDistanceTypeResponse(dt db.DistanceType) DistanceTypeResponse {
	return DistanceTypeResponse{
		ID:   dt.ID,
		Name: dt.Name,
		Km:   convert.PgNumericToFloat64(dt.Km),
	}
}
