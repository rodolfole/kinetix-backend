package participants

import (
	"time"

	"github.com/google/uuid"
)

// Participant represents a participant entity
type Participant struct {
	ID                    uuid.UUID  `json:"id"`
	FirstName             string     `json:"first_name"`
	LastName              string     `json:"last_name"`
	SecondLastName        *string    `json:"second_last_name,omitempty"`
	Gender                string     `json:"gender"`
	BirthDate             time.Time  `json:"birth_date"`
	Email                 string     `json:"email"`
	Phone                 string     `json:"phone"`
	Country               string     `json:"country"`
	Municipality          *string    `json:"municipality,omitempty"`
	State                 *string    `json:"state,omitempty"`
	ZipCode               *string    `json:"zip_code,omitempty"`
	Team                  *string    `json:"team,omitempty"`
	Size                  *string    `json:"size,omitempty"`
	EmergencyContactName  *string    `json:"emergency_contact_name,omitempty"`
	EmergencyContactPhone *string    `json:"emergency_contact_phone,omitempty"`
	MedicalInfo           *string    `json:"medical_info,omitempty"`
	WaiverSignedAt        *time.Time `json:"waiver_signed_at,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

// CreateParticipantRequest represents the request body for creating a participant
type CreateParticipantRequest struct {
	FirstName             string    `json:"first_name" validate:"required,max=100"`
	LastName              string    `json:"last_name" validate:"required,max=100"`
	SecondLastName        *string   `json:"second_last_name" validate:"omitempty,max=100"`
	Gender                string    `json:"gender" validate:"required,max=20"`
	BirthDate             time.Time `json:"birth_date" validate:"required"`
	Email                 string    `json:"email" validate:"required,email"`
	Phone                 string    `json:"phone" validate:"required,min=10,max=20"`
	Country               string    `json:"country" validate:"required,len=2"`
	Municipality          *string   `json:"municipality" validate:"omitempty,max=100"`
	State                 *string   `json:"state" validate:"omitempty,max=100"`
	ZipCode               *string   `json:"zip_code" validate:"omitempty,max=10"`
	Team                  *string   `json:"team" validate:"omitempty,max=255"`
	Size                  *string   `json:"size" validate:"omitempty,oneof=XS S M L XL XXL"`
	EmergencyContactName  *string   `json:"emergency_contact_name" validate:"omitempty,max=255"`
	EmergencyContactPhone *string   `json:"emergency_contact_phone" validate:"omitempty,max=20"`
	MedicalInfo           *string   `json:"medical_info" validate:"omitempty,max=1000"`
}

// UpdateParticipantRequest represents the request body for updating a participant
type UpdateParticipantRequest struct {
	FirstName             string    `json:"first_name" validate:"required,max=100"`
	LastName              string    `json:"last_name" validate:"required,max=100"`
	SecondLastName        *string   `json:"second_last_name" validate:"omitempty,max=100"`
	Gender                string    `json:"gender" validate:"required,max=20"`
	BirthDate             time.Time `json:"birth_date" validate:"required"`
	Email                 string    `json:"email" validate:"required,email"`
	Phone                 string    `json:"phone" validate:"required,min=10,max=20"`
	Country               string    `json:"country" validate:"required,len=2"`
	Municipality          *string   `json:"municipality" validate:"omitempty,max=100"`
	State                 *string   `json:"state" validate:"omitempty,max=100"`
	ZipCode               *string   `json:"zip_code" validate:"omitempty,max=10"`
	Team                  *string   `json:"team" validate:"omitempty,max=255"`
	Size                  *string   `json:"size" validate:"omitempty,oneof=XS S M L XL XXL"`
	EmergencyContactName  *string   `json:"emergency_contact_name" validate:"omitempty,max=255"`
	EmergencyContactPhone *string   `json:"emergency_contact_phone" validate:"omitempty,max=20"`
	MedicalInfo           *string   `json:"medical_info" validate:"omitempty,max=1000"`
}

// UpdateParticipantTeamSizeRequest represents the request body for updating only team and size
type UpdateParticipantTeamSizeRequest struct {
	Team *string `json:"team" validate:"omitempty,max=255"`
	Size *string `json:"size" validate:"omitempty,oneof=XS S M L XL XXL"`
}

// ParticipantResponse represents the response structure
type ParticipantResponse struct {
	Message string      `json:"message"`
	Data    Participant `json:"data"`
}

// ParticipantsListResponse represents a list of participants
type ParticipantsListResponse struct {
	Message string        `json:"message"`
	Data    []Participant `json:"data"`
	Total   int64         `json:"total"`
}

// GenericResponse represents a generic response
type GenericResponse struct {
	Message string `json:"message"`
}
