package routes

import (
	"encoding/json"

	"github.com/google/uuid"
)

// ── Request types ──────────────────────────────────────────────────────────

// UploadRouteRequest represents the multipart form data for uploading a route
type UploadRouteRequest struct {
	EventID    uuid.UUID `json:"event_id"`
	DistanceID *uuid.UUID `json:"distance_id,omitempty"`
	Name       string    `json:"name"`
}

// ── Response types ─────────────────────────────────────────────────────────

// RouteResponse represents a route for API responses
type RouteResponse struct {
	ID                 uuid.UUID        `json:"id"`
	EventID            uuid.UUID        `json:"event_id"`
	DistanceID         *uuid.UUID       `json:"distance_id,omitempty"`
	Name               string           `json:"name"`
	GeoJSON            json.RawMessage  `json:"geojson"`
	ElevationProfile   json.RawMessage  `json:"elevation_profile,omitempty"`
	TotalDistanceKm    *float64         `json:"total_distance_km,omitempty"`
	TotalElevationGain *float64         `json:"total_elevation_gain,omitempty"`
	MinElevation       *float64         `json:"min_elevation,omitempty"`
	MaxElevation       *float64         `json:"max_elevation,omitempty"`
	CreatedAt          string           `json:"created_at"`
}

// RouteListItem represents a route without geojson for list views
type RouteListItem struct {
	ID                 uuid.UUID  `json:"id"`
	EventID            uuid.UUID  `json:"event_id"`
	DistanceID         *uuid.UUID `json:"distance_id,omitempty"`
	Name               string     `json:"name"`
	TotalDistanceKm    *float64   `json:"total_distance_km,omitempty"`
	TotalElevationGain *float64   `json:"total_elevation_gain,omitempty"`
	MinElevation       *float64   `json:"min_elevation,omitempty"`
	MaxElevation       *float64   `json:"max_elevation,omitempty"`
	CreatedAt          string     `json:"created_at"`
}

// GenericResponse wraps a single data payload
type GenericResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}
