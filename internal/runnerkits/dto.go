package runnerkits

import (
	"github.com/google/uuid"
)

// ── Request types ──────────────────────────────────────────────────────────

// CreateRunnerKitItemRequest represents the request body for creating a runner kit item
type CreateRunnerKitItemRequest struct {
	EventID     uuid.UUID `json:"event_id" validate:"required"`
	Name        string    `json:"name" validate:"required,max=100"`
	Icon        string    `json:"icon" validate:"omitempty,max=50"`
}

// UpdateRunnerKitItemRequest represents the request body for updating a runner kit item
type UpdateRunnerKitItemRequest struct {
	Name        string `json:"name" validate:"required,max=100"`
	Icon        string `json:"icon" validate:"omitempty,max=50"`
}

// ── Response types ─────────────────────────────────────────────────────────

// RunnerKitItemResponse represents a runner kit item for API responses
type RunnerKitItemResponse struct {
	ID          uuid.UUID `json:"id"`
	EventID     uuid.UUID `json:"event_id"`
	Name        string    `json:"name"`
	Icon        string    `json:"icon"`
	CreatedAt   string    `json:"created_at"`
}

// GenericResponse wraps a single data payload
type GenericResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}
