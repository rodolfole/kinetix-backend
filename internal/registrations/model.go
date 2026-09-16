package registrations

import (
	"time"

	"github.com/google/uuid"
)

// Participant represents a participant (simplified for internal use)
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

// Registration represents a registration entity
type Registration struct {
	ID               uuid.UUID  `json:"id"`
	EventID          uuid.UUID  `json:"event_id"`
	ParticipantID    uuid.UUID  `json:"participant_id"`
	DistanceID       uuid.UUID  `json:"distance_id"`
	Category         string     `json:"category"`
	BibNumber        *string    `json:"bib_number,omitempty"`
	QRCode           *string    `json:"qr_code,omitempty"`
	Time             *string    `json:"time,omitempty"`
	Status           string     `json:"status"`
	RegistrationDate time.Time  `json:"registration_date"`
	CheckedInAt      *time.Time `json:"checked_in_at,omitempty"`
}

// CreateRegistrationRequest represents the request body for creating a registration
type CreateRegistrationRequest struct {
	EventID       uuid.UUID `json:"event_id" validate:"required"`
	ParticipantID uuid.UUID `json:"participant_id" validate:"required"`
	DistanceID    uuid.UUID `json:"distance_id" validate:"required"`
	Category      string    `json:"category" validate:"required,max=20"`
	Time          *string   `json:"time" validate:"omitempty,max=20"`
}

// CreateWithParticipantRequest represents the request body for creating a participant and registration together
// If ParticipantID is provided, the existing participant is used and team/size are updated
type CreateWithParticipantRequest struct {
	EventID       uuid.UUID                      `json:"event_id" validate:"required"`
	ParticipantID *uuid.UUID                    `json:"participant_id,omitempty"` // When set, use existing participant
	Participant   CreateParticipantData         `json:"participant,omitempty"`     // Required only if ParticipantID not set
	Registration  CreateWithParticipantRegistration `json:"registration" validate:"required"`
}

// CreateParticipantData represents the participant data in a with-participant request
type CreateParticipantData struct {
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
	WaiverSignedAt        *time.Time `json:"waiver_signed_at"`
	Address               string    `json:"address" validate:"required"`
}

// CreateWithParticipantRegistration represents the registration data in a with-participant request
type CreateWithParticipantRegistration struct {
	DistanceID uuid.UUID `json:"distance_id" validate:"required"`
	Category   string    `json:"category" validate:"required,max=20"`
	Time       *string   `json:"time" validate:"omitempty,max=20"`
	Team       *string   `json:"team" validate:"omitempty,max=255"`
	Size       *string   `json:"size" validate:"omitempty,oneof=XS S M L XL XXL"`
}

// RegistrationResponse represents the response structure
type RegistrationResponse struct {
	Message string       `json:"message"`
	Data    Registration `json:"data"`
}

// RegistrationsListResponse represents a list of registrations
type RegistrationsListResponse struct {
	Message string         `json:"message"`
	Data    []Registration `json:"data"`
}
