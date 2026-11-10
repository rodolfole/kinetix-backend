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

// Service holds the business logic for the admin module.
type Service struct {
	repo *Repository
}

func NewService(r *Repository) *Service {
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
func (s *Service) SendPayout(ctx context.Context, mpClient *mp.Client, payoutID uuid.UUID) error {
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
	resp, err := mpClient.CreatePayout(creds.AccessToken, mp.PayoutRequest{
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
