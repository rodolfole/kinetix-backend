package distances

import (
	"github.com/google/uuid"
)

// ── Request types ──────────────────────────────────────────────────────────

// CreateDistanceRequest represents the request body for creating a distance
type CreateDistanceRequest struct {
	EventID        uuid.UUID `json:"event_id" validate:"required"`
	DistanceTypeID uuid.UUID `json:"distance_type_id" validate:"required"`
	Capacity       int       `json:"capacity" validate:"required,gt=0"`
	Surface        string    `json:"surface" validate:"required,max=50"`
	Elevation      *int      `json:"elevation,omitempty"`
	TimeLimit      string    `json:"time_limit,omitempty"`
	StartLat       *float64  `json:"start_lat" validate:"omitempty"`
	StartLng       *float64  `json:"start_lng" validate:"omitempty"`
	EndLat         *float64  `json:"end_lat" validate:"omitempty"`
	EndLng         *float64  `json:"end_lng" validate:"omitempty"`
}

// UpdateDistanceRequest represents the request body for updating a distance
type UpdateDistanceRequest struct {
	DistanceTypeID uuid.UUID `json:"distance_type_id" validate:"required"`
	Capacity       int       `json:"capacity" validate:"required,gt=0"`
	Surface        string    `json:"surface" validate:"required,max=50"`
	Elevation      *int      `json:"elevation,omitempty"`
	TimeLimit      string    `json:"time_limit,omitempty"`
	StartLat       *float64  `json:"start_lat" validate:"omitempty"`
	StartLng       *float64  `json:"start_lng" validate:"omitempty"`
	EndLat         *float64  `json:"end_lat" validate:"omitempty"`
	EndLng         *float64  `json:"end_lng" validate:"omitempty"`
}

// ── Response types ─────────────────────────────────────────────────────────

// DistanceTypeResponse represents a distance type for API responses
type DistanceTypeResponse struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	Km   float64   `json:"km"`
}

// DistanceResponse represents a distance for API responses
type DistanceResponse struct {
	ID             uuid.UUID `json:"id"`
	EventID        uuid.UUID `json:"event_id"`
	DistanceTypeID uuid.UUID `json:"distance_type_id"`
	Km             float64   `json:"km"`
	Capacity       int       `json:"capacity"`
	Surface        string    `json:"surface"`
	Elevation      *int      `json:"elevation,omitempty"`
	TimeLimit      string    `json:"time_limit,omitempty"`
	StartLat       *float64  `json:"start_lat,omitempty"`
	StartLng       *float64  `json:"start_lng,omitempty"`
	EndLat         *float64  `json:"end_lat,omitempty"`
	EndLng         *float64  `json:"end_lng,omitempty"`
}

// GenericResponse wraps a single data payload
type GenericResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// GenericListResponse wraps a list data payload
type GenericListResponse struct {
	Message string        `json:"message"`
	Data    []interface{} `json:"data"`
}
