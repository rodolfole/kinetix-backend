package registrations

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"strings"

	repo "kinetix-api/internal/adapters/postgresql/sqlc"
	"kinetix-api/internal/database/convert"
)

var ErrAlreadyRegistered = errors.New("participant already registered for this event")

// ServiceRepository is the data access the registrations service needs.
// Declared by the consumer (PATTERNS) so tests can hand in fakes.
type ServiceRepository interface {
	Create(ctx context.Context, req CreateRegistrationRequest) (Registration, error)
	GetByID(ctx context.Context, id uuid.UUID) (Registration, error)
	GetByEventParticipant(ctx context.Context, eventID, participantID uuid.UUID) (Registration, error)
	ListByEvent(ctx context.Context, eventID uuid.UUID) ([]Registration, error)
	ListByParticipant(ctx context.Context, participantID uuid.UUID) ([]Registration, error)
	Delete(ctx context.Context, id uuid.UUID) error
	CreateParticipant(ctx context.Context, p CreateParticipantData) (Participant, error)
	GetParticipant(ctx context.Context, id uuid.UUID) (repo.Participant, error)
	UpdateParticipant(ctx context.Context, p repo.UpdateParticipantParams) (repo.Participant, error)
	EventParticipantFields(ctx context.Context, eventID uuid.UUID) ([]repo.EventParticipantField, error)
}

// TxRunner scopes fn to a transaction-backed repository (store.ExecTx,
// ADR-14). Optional: nil keeps the previous non-transactional behavior.
type TxRunner interface {
	ExecTx(ctx context.Context, fn func(ServiceRepository) error) error
}

// Service handles business logic for registrations
type Service struct {
	repo      ServiceRepository
	tx        TxRunner
	validator *validator.Validate
}

// NewService creates a new registrations service. The concrete store-backed
// repository doubles as the transaction runner.
func NewService(r ServiceRepository, tx TxRunner, validator *validator.Validate) *Service {
	return &Service{repo: r, tx: tx, validator: validator}
}

// Create creates a new registration. The bib sequence read, the insert and
// the QR update run inside one transaction; a bib_number collision retries
// on a fresh transaction until it succeeds.
func (s *Service) Create(ctx context.Context, req CreateRegistrationRequest) (Registration, error) {
	if err := s.validator.Struct(req); err != nil {
		return Registration{}, fmt.Errorf("validation error: %w", err)
	}

	if existing, err := s.repo.GetByEventParticipant(ctx, req.EventID, req.ParticipantID); err == nil && existing.ID != uuid.Nil {
		return Registration{}, ErrAlreadyRegistered
	}

	// No runner configured: legacy non-transactional path.
	if s.tx == nil {
		return s.repo.Create(ctx, req)
	}

	const maxRetries = 3
	for attempt := 0; ; attempt++ {
		var reg Registration
		err := s.tx.ExecTx(ctx, func(tx ServiceRepository) error {
			var err error
			reg, err = tx.Create(ctx, req)
			if isBibConflict(err) {
				// Rollback keeps the registration out; a higher bib wins next time.
				return errBibConflict
			}
			return err
		})
		if err == nil {
			return reg, nil
		}
		if errors.Is(err, errBibConflict) && attempt < maxRetries-1 {
			continue
		}
		return Registration{}, err
	}
}

var errBibConflict = errors.New("bib number collision")

// isBibConflict reports a unique-violation on bib_number (concurrent
// registrations picking the same max+1).
func isBibConflict(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" &&
		strings.Contains(pgErr.ConstraintName, "bib_number")
}

// CreateWithParticipant creates a participant and registration together with dynamic field validation.
// If participantID is provided, uses the existing participant and just updates team/size + creates registration.
// Both writes run in one transaction (the participant update and the registration insert
// must not leave an updated participant without a registration).
func (s *Service) CreateWithParticipant(ctx context.Context, eventID uuid.UUID, participantData CreateParticipantData, registrationData CreateWithParticipantRegistration, participantID *uuid.UUID) (Registration, error) {
	// Existing participant: update team/size and create the registration together.
	if participantID != nil {
		regReq := CreateRegistrationRequest{
			EventID:       eventID,
			ParticipantID: *participantID,
			DistanceID:    registrationData.DistanceID,
			Category:      registrationData.Category,
			Time:          registrationData.Time,
		}
		if s.tx == nil {
			return s.applyExistingParticipant(ctx, *participantID, regReq, registrationData)
		}
		var reg Registration
		err := s.tx.ExecTx(ctx, func(tx ServiceRepository) error {
			r, err := s.applyExistingParticipantWith(ctx, tx, *participantID, regReq, registrationData)
			reg = r
			return err
		})
		if err != nil {
			return Registration{}, err
		}
		return reg, nil
	}

	// No participantID provided - create new participant (original flow)
	fields, err := s.repo.EventParticipantFields(ctx, eventID)
	if err != nil {
		// If no config exists, use default validation (all optional)
		fields = []repo.EventParticipantField{}
	}

	if err := s.validateParticipantFields(fields, participantData); err != nil {
		return Registration{}, err
	}

	regReq := CreateRegistrationRequest{
		EventID:    eventID,
		DistanceID: registrationData.DistanceID,
		Category:   registrationData.Category,
		Time:       registrationData.Time,
	}

	// New participant + registration must land atomically.
	if s.tx == nil {
		participant, err := s.repo.CreateParticipant(ctx, participantData)
		if err != nil {
			return Registration{}, fmt.Errorf("failed to create participant: %w", err)
		}
		regReq.ParticipantID = participant.ID
		return s.repo.Create(ctx, regReq)
	}

	var reg Registration
	err = s.tx.ExecTx(ctx, func(tx ServiceRepository) error {
		participant, err := tx.CreateParticipant(ctx, participantData)
		if err != nil {
			return fmt.Errorf("failed to create participant: %w", err)
		}
		regReq.ParticipantID = participant.ID
		reg, err = tx.Create(ctx, regReq)
		return err
	})
	if err != nil {
		return Registration{}, err
	}
	return reg, nil
}

// applyExistingParticipant keeps the non-transactional legacy flow.
func (s *Service) applyExistingParticipant(ctx context.Context, participantID uuid.UUID, regReq CreateRegistrationRequest, registrationData CreateWithParticipantRegistration) (Registration, error) {
	return s.applyExistingParticipantWith(ctx, s.repo, participantID, regReq, registrationData)
}

func (s *Service) applyExistingParticipantWith(ctx context.Context, tx ServiceRepository, participantID uuid.UUID, regReq CreateRegistrationRequest, registrationData CreateWithParticipantRegistration) (Registration, error) {
	existing, err := tx.GetParticipant(ctx, participantID)
	if err != nil {
		return Registration{}, fmt.Errorf("participant not found: %w", err)
	}

	params := repo.UpdateParticipantParams{
		ID:                    participantID,
		FirstName:             existing.FirstName,
		LastName:              existing.LastName,
		SecondLastName:        existing.SecondLastName,
		Gender:                existing.Gender,
		BirthDate:             existing.BirthDate,
		Email:                 existing.Email,
		Phone:                 existing.Phone,
		Country:               existing.Country,
		Municipality:          existing.Municipality,
		State:                 existing.State,
		ZipCode:               existing.ZipCode,
		Team:                  convert.ToPgTextPtr(registrationData.Team),
		Size:                  convert.ToPgTextPtr(registrationData.Size),
		EmergencyContactName:  existing.EmergencyContactName,
		EmergencyContactPhone: existing.EmergencyContactPhone,
		MedicalInfo:           existing.MedicalInfo,
	}
	if _, err := tx.UpdateParticipant(ctx, params); err != nil {
		return Registration{}, fmt.Errorf("failed to update participant team/size: %w", err)
	}

	return tx.Create(ctx, regReq)
}

// validateParticipantFields enforces the event-specific required fields.
func (s *Service) validateParticipantFields(fields []repo.EventParticipantField, p CreateParticipantData) error {
	requiredFields := make(map[string]bool)
	for _, f := range fields {
		if f.IsRequired {
			requiredFields[f.FieldName] = true
		}
	}
	requiredFields["address"] = true

	for field, err := range map[string]error{
		"country":          errStr("country is required for this event", p.Country == ""),
		"city":             errStr("city is required for this event", p.Municipality == nil || *p.Municipality == ""),
		"state":            errStr("state is required for this event", p.State == nil || *p.State == ""),
		"zip_code":         errStr("zip_code is required for this event", p.ZipCode == nil || *p.ZipCode == ""),
		"medical_info":     errStr("medical_info is required for this event", p.MedicalInfo == nil || *p.MedicalInfo == ""),
		"waiver_signed_at": errStr("waiver_signed_at is required for this event", p.WaiverSignedAt == nil),
	} {
		if requiredFields[field] && err != nil {
			return err
		}
	}
	return nil
}

// errStr carries a conditional error message for the required-field table.
func errStr(msg string, missing bool) error {
	if missing {
		return errors.New(msg)
	}
	return nil
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
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return err
	}

	return s.repo.Delete(ctx, id)
}
