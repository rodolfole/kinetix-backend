package participants

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	repo "kinetix-api/internal/adapters/postgresql/sqlc"
	"kinetix-api/internal/database/convert"
	"kinetix-api/internal/store"
)

// Repository handles database operations for participants
type Repository struct {
	store *store.Store
}

// NewRepository creates a new participant repository
func NewRepository(s *store.Store) *Repository {
	return &Repository{store: s}
}

// Alias methods for compatibility
func (r *Repository) GetParticipantByID(ctx context.Context, id uuid.UUID) (Participant, error) {
	return r.GetByID(ctx, id)
}

func (r *Repository) ListAllParticipants(ctx context.Context, limit, offset int) ([]Participant, int64, error) {
	return r.List(ctx, limit, offset)
}

// Create inserts a new participant
func (r *Repository) Create(ctx context.Context, req CreateParticipantRequest) (Participant, error) {
	p, err := r.store.Queries.CreateParticipant(ctx, repo.CreateParticipantParams{
		FirstName:             req.FirstName,
		LastName:              req.LastName,
		SecondLastName:        convert.ToPgTextPtr(req.SecondLastName),
		Gender:                req.Gender,
		BirthDate:             req.BirthDate,
		Email:                 req.Email,
		Phone:                 req.Phone,
		Country:               req.Country,
		Municipality:          convert.ToPgTextPtr(req.Municipality),
		State:                 convert.ToPgTextPtr(req.State),
		ZipCode:               convert.ToPgTextPtr(req.ZipCode),
		Team:                  convert.ToPgTextPtr(req.Team),
		Size:                  convert.ToPgTextPtr(req.Size),
		EmergencyContactName:  convert.ToPgTextPtr(req.EmergencyContactName),
		EmergencyContactPhone: convert.ToPgTextPtr(req.EmergencyContactPhone),
		MedicalInfo:           convert.ToPgTextPtr(req.MedicalInfo),
	})
	if err != nil {
		return Participant{}, fmt.Errorf("failed to create participant: %w", err)
	}
	return toParticipant(p), nil
}

// GetByID retrieves a participant by ID
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (Participant, error) {
	p, err := r.store.Queries.GetParticipant(ctx, id)
	if err != nil {
		return Participant{}, fmt.Errorf("participant not found: %w", err)
	}
	return toParticipant(p), nil
}

// GetByEmail retrieves a participant by email
func (r *Repository) GetByEmail(ctx context.Context, email string) (Participant, error) {
	p, err := r.store.Queries.GetParticipantByEmail(ctx, email)
	if err != nil {
		return Participant{}, fmt.Errorf("participant not found: %w", err)
	}
	return toParticipant(p), nil
}

// List retrieves a paginated list of participants
func (r *Repository) List(ctx context.Context, limit, offset int) ([]Participant, int64, error) {
	participants, err := r.store.Queries.ListParticipants(ctx, repo.ListParticipantsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list participants: %w", err)
	}

	result := make([]Participant, len(participants))
	for i, p := range participants {
		result[i] = toParticipant(p)
	}

	return result, int64(len(result)), nil
}

// Update updates a participant
func (r *Repository) Update(ctx context.Context, id uuid.UUID, req UpdateParticipantRequest) (Participant, error) {
	p, err := r.store.Queries.UpdateParticipant(ctx, repo.UpdateParticipantParams{
		ID:                    id,
		FirstName:             req.FirstName,
		LastName:              req.LastName,
		SecondLastName:        convert.ToPgTextPtr(req.SecondLastName),
		Gender:                req.Gender,
		BirthDate:             req.BirthDate,
		Email:                 req.Email,
		Phone:                 req.Phone,
		Country:               req.Country,
		Municipality:          convert.ToPgTextPtr(req.Municipality),
		State:                 convert.ToPgTextPtr(req.State),
		ZipCode:               convert.ToPgTextPtr(req.ZipCode),
		Team:                  convert.ToPgTextPtr(req.Team),
		Size:                  convert.ToPgTextPtr(req.Size),
		EmergencyContactName:  convert.ToPgTextPtr(req.EmergencyContactName),
		EmergencyContactPhone: convert.ToPgTextPtr(req.EmergencyContactPhone),
		MedicalInfo:           convert.ToPgTextPtr(req.MedicalInfo),
	})
	if err != nil {
		return Participant{}, fmt.Errorf("failed to update participant: %w", err)
	}
	return toParticipant(p), nil
}

// Delete removes a participant
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	err := r.store.Queries.DeleteParticipant(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete participant: %w", err)
	}
	return nil
}

// UpdateTeamSize updates only team and size fields for a participant
func (r *Repository) UpdateTeamSize(ctx context.Context, id uuid.UUID, req UpdateParticipantTeamSizeRequest) (Participant, error) {
	// Get current participant to preserve other fields
	current, err := r.GetByID(ctx, id)
	if err != nil {
		return Participant{}, fmt.Errorf("participant not found: %w", err)
	}

	p, err := r.store.Queries.UpdateParticipant(ctx, repo.UpdateParticipantParams{
		ID:                    id,
		FirstName:             current.FirstName,
		LastName:              current.LastName,
		SecondLastName:        convert.ToPgTextPtr(current.SecondLastName),
		Gender:                current.Gender,
		BirthDate:             current.BirthDate,
		Email:                 current.Email,
		Phone:                 current.Phone,
		Country:               current.Country,
		Municipality:          convert.ToPgTextPtr(current.Municipality),
		State:                 convert.ToPgTextPtr(current.State),
		ZipCode:               convert.ToPgTextPtr(current.ZipCode),
		Team:                  convert.ToPgTextPtr(req.Team),
		Size:                  convert.ToPgTextPtr(req.Size),
		EmergencyContactName:  convert.ToPgTextPtr(current.EmergencyContactName),
		EmergencyContactPhone: convert.ToPgTextPtr(current.EmergencyContactPhone),
		MedicalInfo:           convert.ToPgTextPtr(current.MedicalInfo),
	})
	if err != nil {
		return Participant{}, fmt.Errorf("failed to update participant team/size: %w", err)
	}
	return toParticipant(p), nil
}

// ============ Model conversion ============

func toParticipant(p repo.Participant) Participant {
	return Participant{
		ID:                    p.ID,
		FirstName:             p.FirstName,
		LastName:              p.LastName,
		SecondLastName:        convert.PgTextToStringPtr(p.SecondLastName),
		Gender:                p.Gender,
		BirthDate:             p.BirthDate,
		Email:                 p.Email,
		Phone:                 p.Phone,
		Country:               p.Country,
		Municipality:          convert.PgTextToStringPtr(p.Municipality),
		State:                 convert.PgTextToStringPtr(p.State),
		ZipCode:               convert.PgTextToStringPtr(p.ZipCode),
		Team:                  convert.PgTextToStringPtr(p.Team),
		Size:                  convert.PgTextToStringPtr(p.Size),
		EmergencyContactName:  convert.PgTextToStringPtr(p.EmergencyContactName),
		EmergencyContactPhone: convert.PgTextToStringPtr(p.EmergencyContactPhone),
		MedicalInfo:           convert.PgTextToStringPtr(p.MedicalInfo),
		WaiverSignedAt:        convert.PgTimeToTimePtr(p.WaiverSignedAt),
		CreatedAt:             convert.PgTimeToTime(p.CreatedAt),
		UpdatedAt:             convert.PgTimeToTime(p.UpdatedAt),
	}
}
