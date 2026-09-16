package agecategories

import "github.com/google/uuid"

// AgeCategoryResponse represents an age category for API responses
type AgeCategoryResponse struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	MinAge       *int      `json:"min_age,omitempty"`
	MaxAge       *int      `json:"max_age,omitempty"`
	DisplayOrder int       `json:"display_order"`
}

// GenericResponse wraps a single data payload
type GenericResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}
