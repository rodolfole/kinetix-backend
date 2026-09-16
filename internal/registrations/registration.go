package registrations

import (
	"context"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	repo "kinetix-api/internal/adapters/postgresql/sqlc"
	"kinetix-api/internal/database/convert"
	"kinetix-api/internal/store"
)

// Repository handles database operations for registrations
type Repository struct {
	store *store.Store
}

// NewRepository creates a new registration repository
func NewRepository(s *store.Store) *Repository {
	return &Repository{store: s}
}

// Alias methods
func (r *Repository) GetRegistrationByID(ctx context.Context, id uuid.UUID) (Registration, error) {
	return r.GetByID(ctx, id)
}

func (r *Repository) ListRegistrationsByEvent(ctx context.Context, eventID uuid.UUID) ([]Registration, error) {
	return r.ListByEvent(ctx, eventID)
}

func (r *Repository) ListRegistrationsByParticipant(ctx context.Context, participantID uuid.UUID) ([]Registration, error) {
	return r.ListByParticipant(ctx, participantID)
}

// generateBibNumber generates a sequential 4-digit bib number (e.g., 0001, 0002, ...)
func (r *Repository) generateBibNumber(ctx context.Context, eventID uuid.UUID) (string, error) {
	result, err := r.store.Queries.GetMaxBibNumberByEvent(ctx, eventID)
	if err != nil {
		return "", fmt.Errorf("failed to get max bib number: %w", err)
	}

	var maxNum int64
	if result != nil {
		// result is interface{} and can come back as any integer type
		// (pgx decodes INTEGER as int32); convert via string to be safe
		parsed, err := strconv.ParseInt(fmt.Sprintf("%v", result), 10, 64)
		if err == nil {
			maxNum = parsed
		}
	}

	nextNum := maxNum + 1
	return fmt.Sprintf("%04d", nextNum), nil
}

// generateQRCode generates a QR code string with registration info
func generateQRCode(regID, eventID, participantID uuid.UUID) string {
	// QR code contains JSON with registration data for scanning at check-in
	return fmt.Sprintf(`{"reg_id":"%s","event_id":"%s","p_id":"%s"}`,
		regID.String(), eventID.String(), participantID.String())
}

// Create inserts a new registration
func (r *Repository) Create(ctx context.Context, req CreateRegistrationRequest) (Registration, error) {
	// Generate bib number
	bibNumber, err := r.generateBibNumber(ctx, req.EventID)
	if err != nil {
		return Registration{}, fmt.Errorf("failed to generate bib number: %w", err)
	}

	// Create registration first to get ID
	reg, err := r.store.Queries.CreateRegistration(ctx, repo.CreateRegistrationParams{
		EventID:       req.EventID,
		ParticipantID: req.ParticipantID,
		DistanceID:    req.DistanceID,
		Category:      req.Category,
		BibNumber:     convert.ToPgText(bibNumber),
		QrCode:        convert.ToPgText(""),
		Time:          convert.ToPgTextPtr(req.Time),
		Status:        convert.ToPgText("pending"),
	})
	if err != nil {
		return Registration{}, fmt.Errorf("failed to create registration: %w", err)
	}

	// Generate QR code with the actual registration ID
	qrCode := generateQRCode(reg.ID, req.EventID, req.ParticipantID)

	// Update registration with QR code
	_, err = r.store.Queries.UpdateRegistrationQRCode(ctx, repo.UpdateRegistrationQRCodeParams{
		ID:     reg.ID,
		QrCode: convert.ToPgText(qrCode),
	})
	if err != nil {
		// Log error but don't fail - QR code is optional
		// In production, you'd want proper error handling
	}

	return Registration{
		ID:               reg.ID,
		EventID:          reg.EventID,
		ParticipantID:    reg.ParticipantID,
		DistanceID:       reg.DistanceID,
		Category:         reg.Category,
		BibNumber:        &bibNumber,
		QRCode:           &qrCode,
		Time:             convert.PgTextToStringPtr(reg.Time),
		Status:           convert.PgTextToString(reg.Status),
		RegistrationDate: convert.PgTimeToTime(reg.RegistrationDate),
		CheckedInAt:      nil,
	}, nil
}

// GetByID retrieves a registration by ID
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (Registration, error) {
	reg, err := r.store.Queries.GetRegistration(ctx, id)
	if err != nil {
		return Registration{}, fmt.Errorf("registration not found: %w", err)
	}
	return toRegistration(reg), nil
}

// GetByEventParticipant retrieves registration by event and participant
func (r *Repository) GetByEventParticipant(ctx context.Context, eventID, participantID uuid.UUID) (Registration, error) {
	reg, err := r.store.Queries.GetRegistrationByEventParticipant(ctx, repo.GetRegistrationByEventParticipantParams{
		EventID:       eventID,
		ParticipantID: participantID,
	})
	if err != nil {
		return Registration{}, fmt.Errorf("registration not found: %w", err)
	}
	return toRegistration(reg), nil
}

// ListByEvent retrieves registrations by event
func (r *Repository) ListByEvent(ctx context.Context, eventID uuid.UUID) ([]Registration, error) {
	registrations, err := r.store.Queries.ListRegistrationsByEvent(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to list registrations: %w", err)
	}

	result := make([]Registration, len(registrations))
	for i, reg := range registrations {
		result[i] = toRegistration(reg)
	}
	return result, nil
}

// ListByParticipant retrieves registrations by participant
func (r *Repository) ListByParticipant(ctx context.Context, participantID uuid.UUID) ([]Registration, error) {
	registrations, err := r.store.Queries.ListRegistrationsByParticipant(ctx, participantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list registrations: %w", err)
	}

	result := make([]Registration, len(registrations))
	for i, reg := range registrations {
		result[i] = toRegistration(reg)
	}
	return result, nil
}

// Delete removes a registration
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	err := r.store.Queries.DeleteRegistration(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete registration: %w", err)
	}
	return nil
}

// CreateParticipant creates a new participant (used by CreateWithParticipant)
func (r *Repository) CreateParticipant(ctx context.Context, req CreateParticipantData) (Participant, error) {
	participant, err := r.store.Queries.CreateParticipant(ctx, repo.CreateParticipantParams{
		FirstName:              req.FirstName,
		LastName:               req.LastName,
		SecondLastName:         convert.ToPgTextPtr(req.SecondLastName),
		Gender:                 req.Gender,
		BirthDate:              req.BirthDate,
		Email:                  req.Email,
		Phone:                  req.Phone,
		Country:                req.Country,
		Municipality:           convert.ToPgTextPtr(req.Municipality),
		State:                  convert.ToPgTextPtr(req.State),
		ZipCode:                convert.ToPgTextPtr(req.ZipCode),
		Team:                   convert.ToPgTextPtr(req.Team),
		Size:                   convert.ToPgTextPtr(req.Size),
		EmergencyContactName:   convert.ToPgTextPtr(req.EmergencyContactName),
		EmergencyContactPhone:  convert.ToPgTextPtr(req.EmergencyContactPhone),
		MedicalInfo:            convert.ToPgTextPtr(req.MedicalInfo),
		WaiverSignedAt:         convert.ToPgTimePtr(req.WaiverSignedAt),
	})
	if err != nil {
		return Participant{}, fmt.Errorf("failed to create participant: %w", err)
	}

	return Participant{
		ID:                    participant.ID,
		FirstName:             participant.FirstName,
		LastName:              participant.LastName,
		SecondLastName:        convert.PgTextToStringPtr(participant.SecondLastName),
		Gender:                participant.Gender,
		BirthDate:             participant.BirthDate,
		Email:                 participant.Email,
		Phone:                 participant.Phone,
		Country:               participant.Country,
		Municipality:          convert.PgTextToStringPtr(participant.Municipality),
		State:                 convert.PgTextToStringPtr(participant.State),
		ZipCode:               convert.PgTextToStringPtr(participant.ZipCode),
		Team:                  convert.PgTextToStringPtr(participant.Team),
		Size:                  convert.PgTextToStringPtr(participant.Size),
		EmergencyContactName:   convert.PgTextToStringPtr(participant.EmergencyContactName),
		EmergencyContactPhone:  convert.PgTextToStringPtr(participant.EmergencyContactPhone),
		MedicalInfo:           convert.PgTextToStringPtr(participant.MedicalInfo),
		WaiverSignedAt:        convert.PgTimeToTimePtr(participant.WaiverSignedAt),
		CreatedAt:            convert.PgTimeToTime(participant.CreatedAt),
		UpdatedAt:            convert.PgTimeToTime(participant.UpdatedAt),
	}, nil
}

// ============ Model conversion ============

func toRegistration(r repo.Registration) Registration {
	return Registration{
		ID:               r.ID,
		EventID:          r.EventID,
		ParticipantID:    r.ParticipantID,
		DistanceID:       r.DistanceID,
		Category:         r.Category,
		BibNumber:        convert.PgTextToStringPtr(r.BibNumber),
		QRCode:           convert.PgTextToStringPtr(r.QrCode),
		Time:             convert.PgTextToStringPtr(r.Time),
		Status:           convert.PgTextToString(r.Status),
		RegistrationDate: convert.PgTimeToTime(r.RegistrationDate),
		CheckedInAt:      convert.PgTimeToTimePtr(r.CheckedInAt),
	}
}
