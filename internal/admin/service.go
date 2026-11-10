package admin

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/google/uuid"
	db "kinetix-api/internal/adapters/postgresql/sqlc"
	"kinetix-api/internal/mp"
)

// ServiceRepository is the data access the admin service needs (service seam).
type ServiceRepository interface {
	GetPlatformConfig(ctx context.Context) (db.PlatformConfig, error)
	UpdatePlatformConfig(ctx context.Context, commissionPct float64, updatedBy string) (db.PlatformConfig, error)
	ListPayouts(ctx context.Context, limit int32) ([]db.OrganizerPayout, error)
	ListPayoutsByStatus(ctx context.Context, status string, limit int32) ([]db.OrganizerPayout, error)
	GetPayoutByID(ctx context.Context, id uuid.UUID) (db.OrganizerPayout, error)
	GetMPCredentials(ctx context.Context, id uuid.UUID) (db.OrganizerMpCredential, error)
	UpsertMPCredentials(ctx context.Context, p db.UpsertOrganizerMPCredentialsParams) (db.OrganizerMpCredential, error)
	MarkPayoutSent(ctx context.Context, id uuid.UUID, mpPayoutID string) (db.OrganizerPayout, error)
	MarkPayoutFailed(ctx context.Context, id uuid.UUID, reason string) (db.OrganizerPayout, error)
}

// PayoutGateway is the MP payout API the service uses (test seam).
type PayoutGateway interface {
	CreatePayout(organizerAccessToken string, req mp.PayoutRequest) (*mp.CreatePayoutResponse, error)
}

// Service holds the business logic for the admin module.
type Service struct {
	repo ServiceRepository
}

func NewService(r ServiceRepository) *Service {
	return &Service{repo: r}
}

// GetConfig returns the current commission config.
func (s *Service) GetConfig(ctx context.Context) (db.PlatformConfig, error) {
	return s.repo.GetPlatformConfig(ctx)
}

// UpdateConfig updates the commission %.
func (s *Service) UpdateConfig(ctx context.Context, commissionPct float64, updatedBy string) (db.PlatformConfig, error) {
	if commissionPct < 0 || commissionPct > 100 {
		return db.PlatformConfig{}, ErrInvalidCommission
	}
	return s.repo.UpdatePlatformConfig(ctx, commissionPct, updatedBy)
}

// ListPayouts returns recent payouts (default all statuses, limit 100).
func (s *Service) ListPayouts(ctx context.Context, status string, limit int32) ([]db.OrganizerPayout, error) {
	if status == "" || status == "all" {
		return s.repo.ListPayouts(ctx, limit)
	}
	return s.repo.ListPayoutsByStatus(ctx, status, limit)
}

// CalculatePayoutAmount computes the net amount for an order given the
// total amount in cents and the platform commission percentage. Rounded down
// to the nearest cent.
func CalculatePayoutAmount(totalCents int64, commissionPct float64) int64 {
	if totalCents <= 0 {
		return 0
	}
	if commissionPct < 0 {
		commissionPct = 0
	}
	if commissionPct > 100 {
		commissionPct = 100
	}
	multiplier := 1.0 - (commissionPct / 100.0)
	return int64(math.Floor(float64(totalCents) * multiplier))
}

// UpsertMPCredentials stores the OAuth tokens for an organizer after they
// authorize our MP Marketplace app.
func (s *Service) UpsertMPCredentials(ctx context.Context, organizerID uuid.UUID, tok *mp.OAuthTokenResponse) (db.OrganizerMpCredential, error) {
	expiresAt := time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)
	return s.repo.UpsertMPCredentials(ctx, db.UpsertOrganizerMPCredentialsParams{
		OrganizerID:  organizerID,
		MpUserID:     strconv.FormatInt(tok.UserID, 10),
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		ExpiresAt:    expiresAt,
	})
}

// SendPayout looks up the organizer's MP access_token, calls MP Payouts API
// to disburse the amount, and marks the row sent (or failed with reason).
func (s *Service) SendPayout(ctx context.Context, gateway PayoutGateway, payoutID uuid.UUID) error {
	payout, err := s.repo.GetPayoutByID(ctx, payoutID)
	if err != nil {
		return err
	}
	if payout.Status != "pending" {
		return fmt.Errorf("payout %s is not pending (current: %s)", payoutID, payout.Status)
	}
	creds, err := s.repo.GetMPCredentials(ctx, payout.OrganizerID)
	if err != nil {
		return fmt.Errorf("organizer %s has no MP credentials: %w", payout.OrganizerID, err)
	}
	if creds.AccessToken == "" {
		return fmt.Errorf("organizer %s has empty MP access_token", payout.OrganizerID)
	}
	resp, err := gateway.CreatePayout(creds.AccessToken, mp.PayoutRequest{
		Amount:            float64(payout.AmountCents) / 100.0,
		ExternalReference: payoutID.String(),
	})
	if err != nil {
		_, _ = s.repo.MarkPayoutFailed(ctx, payoutID, err.Error())
		return err
	}
	_, err = s.repo.MarkPayoutSent(ctx, payoutID, strconv.FormatInt(resp.ID, 10))
	return err
}
