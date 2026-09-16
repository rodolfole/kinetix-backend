package registrations

import (
	"context"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	repo "kinetix-api/internal/adapters/postgresql/sqlc"
	"kinetix-api/internal/database/convert"
	"kinetix-api/internal/store"
)

// Service handles business logic for registrations
type Service struct {
	repo      *Repository
	store     *store.Store
	validator *validator.Validate
}

// NewService creates a new registrations service
func NewService(repo *Repository, s *store.Store, validator *validator.Validate) *Service {
	return &Service{
		repo:      repo,
		store:     s,
		validator: validator,
	}
}

// Create creates a new registration
func (s *Service) Create(ctx context.Context, req CreateRegistrationRequest) (Registration, error) {
	if err := s.validator.Struct(req); err != nil {
		return Registration{}, fmt.Errorf("validation error: %w", err)
	}

	// Check if participant already registered for this event
	existing, err := s.repo.GetByEventParticipant(ctx, req.EventID, req.ParticipantID)
	if err == nil && existing.ID != uuid.Nil {
		return Registration{}, fmt.Errorf("participant already registered for this event")
	}

	return s.repo.Create(ctx, req)
}

// CreateWithParticipant creates a participant and registration together with dynamic field validation
// If participantID is provided, uses the existing participant and just updates team/size + creates registration
func (s *Service) CreateWithParticipant(ctx context.Context, eventID uuid.UUID, participantData CreateParticipantData, registrationData CreateWithParticipantRegistration, participantID *uuid.UUID) (Registration, error) {
	// If participantID is provided, use existing participant
	if participantID != nil {
		// Get the existing participant to validate it exists
		existingParticipant, err := s.store.Queries.GetParticipant(ctx, *participantID)
		if err != nil {
			return Registration{}, fmt.Errorf("participant not found: %w", err)
		}

		// Update team/size via store.Queries.UpdateParticipant
		_, err = s.store.Queries.UpdateParticipant(ctx, repo.UpdateParticipantParams{
			ID:                    *participantID,
			FirstName:             existingParticipant.FirstName,
			LastName:              existingParticipant.LastName,
			SecondLastName:        existingParticipant.SecondLastName,
			Gender:                existingParticipant.Gender,
			BirthDate:             existingParticipant.BirthDate,
			Email:                 existingParticipant.Email,
			Phone:                 existingParticipant.Phone,
			Country:               existingParticipant.Country,
			Municipality:          existingParticipant.Municipality,
			State:                 existingParticipant.State,
			ZipCode:               existingParticipant.ZipCode,
			Team:                  convert.ToPgTextPtr(registrationData.Team),
			Size:                  convert.ToPgTextPtr(registrationData.Size),
			EmergencyContactName:  existingParticipant.EmergencyContactName,
			EmergencyContactPhone: existingParticipant.EmergencyContactPhone,
			MedicalInfo:           existingParticipant.MedicalInfo,
		})
		if err != nil {
			return Registration{}, fmt.Errorf("failed to update participant team/size: %w", err)
		}

		// Create registration for existing participant
		regReq := CreateRegistrationRequest{
			EventID:       eventID,
			ParticipantID: *participantID,
			DistanceID:    registrationData.DistanceID,
			Category:      registrationData.Category,
			Time:          registrationData.Time,
		}

		return s.repo.Create(ctx, regReq)
	}

	// No participantID provided - create new participant (original flow)
	// Get participant fields config for this event
	fields, err := s.store.Queries.ListEventParticipantFieldsByEvent(ctx, eventID)
	if err != nil {
		// If no config exists, use default validation (all optional)
		fields = []repo.EventParticipantField{}
	}

	// Build list of required field names based on config
	requiredFields := make(map[string]bool)
	enabledFields := make(map[string]bool)
	for _, f := range fields {
		enabledFields[f.FieldName] = f.IsEnabled
		if f.IsRequired {
			requiredFields[f.FieldName] = true
		}
	}

	// address is always required
	requiredFields["address"] = true

	// Validate participant data based on config
	if requiredFields["country"] && participantData.Country == "" {
		return Registration{}, fmt.Errorf("country is required for this event")
	}
	if requiredFields["city"] && (participantData.Municipality == nil || *participantData.Municipality == "") {
		return Registration{}, fmt.Errorf("city is required for this event")
	}
	if requiredFields["state"] && (participantData.State == nil || *participantData.State == "") {
		return Registration{}, fmt.Errorf("state is required for this event")
	}
	if requiredFields["zip_code"] && (participantData.ZipCode == nil || *participantData.ZipCode == "") {
		return Registration{}, fmt.Errorf("zip_code is required for this event")
	}
	if requiredFields["medical_info"] && (participantData.MedicalInfo == nil || *participantData.MedicalInfo == "") {
		return Registration{}, fmt.Errorf("medical_info is required for this event")
	}
	if requiredFields["waiver_signed_at"] && participantData.WaiverSignedAt == nil {
		return Registration{}, fmt.Errorf("waiver_signed_at is required for this event")
	}

	// Create participant
	participant, err := s.repo.CreateParticipant(ctx, participantData)
	if err != nil {
		return Registration{}, fmt.Errorf("failed to create participant: %w", err)
	}

	// Create registration
	regReq := CreateRegistrationRequest{
		EventID:       eventID,
		ParticipantID: participant.ID,
		DistanceID:    registrationData.DistanceID,
		Category:     registrationData.Category,
		Time:         registrationData.Time,
	}

	return s.repo.Create(ctx, regReq)
}

// GetByID retrieves a registration by ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (Registration, error) {
	return s.repo.GetByID(ctx, id)
}

// GetByEventParticipant retrieves registration by event and participant
func (s *Service) GetByEventParticipant(ctx context.Context, eventID, participantID uuid.UUID) (Registration, error) {
	return s.repo.GetByEventParticipant(ctx, eventID, participantID)
}

// ListByEvent retrieves registrations by event
func (s *Service) ListByEvent(ctx context.Context, eventID uuid.UUID) ([]Registration, error) {
	return s.repo.ListByEvent(ctx, eventID)
}

// ListByParticipant retrieves registrations by participant
func (s *Service) ListByParticipant(ctx context.Context, participantID uuid.UUID) ([]Registration, error) {
	return s.repo.ListByParticipant(ctx, participantID)
}

// Delete removes a registration
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	return s.repo.Delete(ctx, id)
}
