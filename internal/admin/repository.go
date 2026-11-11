package admin

import (
	"context"

	"github.com/google/uuid"
	db "kinetix-api/internal/adapters/postgresql/sqlc"
	"kinetix-api/internal/database/convert"
	"kinetix-api/internal/store"
)

// Repository wraps sqlc Queries with the admin-specific queries.
type Repository struct {
	store *store.Store
}

func NewRepository(s *store.Store) *Repository {
	return &Repository{store: s}
}

// GetPlatformConfig returns the singleton platform_config row.
func (r *Repository) GetPlatformConfig(ctx context.Context) (db.PlatformConfig, error) {
	return r.store.Queries.GetPlatformConfig(ctx)
}

// UpdatePlatformConfig updates the singleton platform_config row.
func (r *Repository) UpdatePlatformConfig(ctx context.Context, commissionPct float64, updatedBy string) (db.PlatformConfig, error) {
	return r.store.Queries.UpdatePlatformConfig(ctx, db.UpdatePlatformConfigParams{
		CommissionPct: convert.ToPgNumeric(commissionPct),
		UpdatedBy:     convert.ToPgText(updatedBy),
	})
}

// ListPayouts returns the most recent payouts (default 100).
func (r *Repository) ListPayouts(ctx context.Context, limit int32) ([]db.OrganizerPayout, error) {
	if limit <= 0 {
		limit = 100
	}
	return r.store.Queries.ListOrganizerPayouts(ctx, limit)
}

// CreatePayout inserts a new pending payout row.
func (r *Repository) CreatePayout(ctx context.Context, params db.CreateOrganizerPayoutParams) (db.OrganizerPayout, error) {
	return r.store.Queries.CreateOrganizerPayout(ctx, params)
}

// ListPayoutsByStatus returns payouts filtered by status.
func (r *Repository) ListPayoutsByStatus(ctx context.Context, status string, limit int32) ([]db.OrganizerPayout, error) {
	if limit <= 0 {
		limit = 100
	}
	return r.store.Queries.ListOrganizerPayoutsByStatus(ctx, db.ListOrganizerPayoutsByStatusParams{
		Status:     status,
		LimitCount: limit,
	})
}

// MarkPayoutSent transitions a pending payout to sent.
func (r *Repository) MarkPayoutSent(ctx context.Context, id uuid.UUID, mpPayoutID string) (db.OrganizerPayout, error) {
	return r.store.Queries.MarkOrganizerPayoutSent(ctx, db.MarkOrganizerPayoutSentParams{
		ID:         id,
		MpPayoutID: convert.ToPgText(mpPayoutID),
	})
}

// MarkPayoutFailed transitions a pending payout to failed with a reason.
func (r *Repository) MarkPayoutFailed(ctx context.Context, id uuid.UUID, reason string) (db.OrganizerPayout, error) {
	return r.store.Queries.MarkOrganizerPayoutFailed(ctx, db.MarkOrganizerPayoutFailedParams{
		ID:            id,
		FailureReason: convert.ToPgText(reason),
	})
}

// GetPayoutByID returns a single payout row.
func (r *Repository) GetPayoutByID(ctx context.Context, id uuid.UUID) (db.OrganizerPayout, error) {
	return r.store.Queries.GetOrganizerPayoutByID(ctx, id)
}

// GetMPCredentials returns the stored OAuth tokens for an organizer.
func (r *Repository) GetMPCredentials(ctx context.Context, organizerID uuid.UUID) (db.OrganizerMpCredential, error) {
	return r.store.Queries.GetOrganizerMPCredentials(ctx, organizerID)
}

// UpsertMPCredentials stores or updates MP OAuth tokens for an organizer.
func (r *Repository) UpsertMPCredentials(ctx context.Context, params db.UpsertOrganizerMPCredentialsParams) (db.OrganizerMpCredential, error) {
	return r.store.Queries.UpsertOrganizerMPCredentials(ctx, params)
}
