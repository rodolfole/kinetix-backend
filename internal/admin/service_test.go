package admin

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	db "kinetix-api/internal/adapters/postgresql/sqlc"
	"kinetix-api/internal/mp"
)

func numericFromFloat(v float64) pgtype.Numeric {
	var n pgtype.Numeric
	if err := n.Scan(strconv.FormatFloat(v, 'f', 2, 64)); err != nil {
		panic(err)
	}
	return n
}

// fakeAdminRepo implements the admin service seam and records writes.
type fakeAdminRepo struct {
	config       db.PlatformConfig
	configErr    error
	updateCalls  int
	payouts      []db.OrganizerPayout
	payoutErr    error
	listByStatus []string
	payoutByID   map[uuid.UUID]db.OrganizerPayout
	creds        map[uuid.UUID]db.OrganizerMpCredential
	sentIDs      []uuid.UUID
	failedIDs    []uuid.UUID
	failedReason []string
	upserts      []db.UpsertOrganizerMPCredentialsParams
}

func (f *fakeAdminRepo) GetPlatformConfig(ctx context.Context) (db.PlatformConfig, error) {
	return f.config, f.configErr
}

func (f *fakeAdminRepo) UpdatePlatformConfig(ctx context.Context, commissionPct float64, updatedBy string) (db.PlatformConfig, error) {
	f.updateCalls++
	f.config.UpdatedBy = textOf(updatedBy)
	return f.config, nil
}

func (f *fakeAdminRepo) ListPayouts(ctx context.Context, limit int32) ([]db.OrganizerPayout, error) {
	if f.payoutErr != nil {
		return nil, f.payoutErr
	}
	f.listByStatus = append(f.listByStatus, "")
	return f.payouts, nil
}

func (f *fakeAdminRepo) ListPayoutsByStatus(ctx context.Context, status string, limit int32) ([]db.OrganizerPayout, error) {
	if f.payoutErr != nil {
		return nil, f.payoutErr
	}
	f.listByStatus = append(f.listByStatus, status)
	return f.payouts, nil
}

func (f *fakeAdminRepo) GetPayoutByID(ctx context.Context, id uuid.UUID) (db.OrganizerPayout, error) {
	return f.payoutByID[id], nil
}

func (f *fakeAdminRepo) GetMPCredentials(ctx context.Context, id uuid.UUID) (db.OrganizerMpCredential, error) {
	cred, ok := f.creds[id]
	if !ok {
		return db.OrganizerMpCredential{}, errors.New("no credentials")
	}
	return cred, nil
}

func (f *fakeAdminRepo) UpsertMPCredentials(ctx context.Context, p db.UpsertOrganizerMPCredentialsParams) (db.OrganizerMpCredential, error) {
	f.upserts = append(f.upserts, p)
	return db.OrganizerMpCredential{}, nil
}

func (f *fakeAdminRepo) MarkPayoutSent(ctx context.Context, id uuid.UUID, mpPayoutID string) (db.OrganizerPayout, error) {
	f.sentIDs = append(f.sentIDs, id)
	return f.payoutByID[id], nil
}

func (f *fakeAdminRepo) MarkPayoutFailed(ctx context.Context, id uuid.UUID, reason string) (db.OrganizerPayout, error) {
	f.failedIDs = append(f.failedIDs, id)
	f.failedReason = append(f.failedReason, reason)
	return f.payoutByID[id], nil
}

// fakeGateway implements the MP payout gateway the service calls.
type fakeGateway struct {
	resp     *mp.CreatePayoutResponse
	err      error
	requests []mp.PayoutRequest
	tokens   []string
}

func (f *fakeGateway) CreatePayout(token string, req mp.PayoutRequest) (*mp.CreatePayoutResponse, error) {
	f.tokens = append(f.tokens, token)
	f.requests = append(f.requests, req)
	return f.resp, f.err
}

func textOf(v string) pgtype.Text { return pgtype.Text{String: v, Valid: true} }

func TestUpdateConfigBounds(t *testing.T) {
	tests := []struct {
		name    string
		pct     float64
		wantErr bool
	}{
		{"zero allowed", 0, false},
		{"ten allowed", 10, false},
		{"hundred allowed", 100, false},
		{"negative rejected", -1, true},
		{"over-hundred rejected", 100.01, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeAdminRepo{}
			svc := NewService(fake)
			if _, err := svc.UpdateConfig(context.Background(), tt.pct, "u"); (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && fake.updateCalls != 0 {
				t.Error("rejected values must not reach the repo")
			}
		})
	}
}

func TestUpdateConfigDelegates(t *testing.T) {
	fake := &fakeAdminRepo{config: db.PlatformConfig{CommissionPct: numericFromFloat(10)}}
	svc := NewService(fake)

	got, err := svc.UpdateConfig(context.Background(), 15, "admin-1")
	if err != nil {
		t.Fatalf("UpdateConfig: %v", err)
	}
	if fake.updateCalls != 1 {
		t.Fatalf("repo update calls = %d, want 1", fake.updateCalls)
	}
	if got.UpdatedBy.String != "admin-1" {
		t.Errorf("updated_by = %q, want admin-1", got.UpdatedBy.String)
	}
}

func TestListPayoutsRoutesByStatus(t *testing.T) {
	fake := &fakeAdminRepo{payouts: []db.OrganizerPayout{{}}}
	svc := NewService(fake)

	for _, status := range []string{"", "all"} {
		if _, err := svc.ListPayouts(context.Background(), status, 10); err != nil {
			t.Fatalf("status %q: %v", status, err)
		}
	}
	got := fake.listByStatus[len(fake.listByStatus)-1]
	if got != "" {
		t.Errorf("empty/all status must use width=100 list, got %q", got)
	}

	fake.listByStatus = nil
	if _, err := svc.ListPayouts(context.Background(), "failed", 10); err != nil {
		t.Fatalf("failed filter: %v", err)
	}
	if got := fake.listByStatus[0]; got != "failed" {
		t.Errorf("status %q must use the filtered list, got %q", "failed", got)
	}
}

func TestSendPayoutHappy(t *testing.T) {
	id := uuid.New()
	fake := &fakeAdminRepo{payoutByID: map[uuid.UUID]db.OrganizerPayout{
		id: {ID: id, OrganizerID: uuid.New(), AmountCents: 9050, Status: "pending"},
	}}
	credOrg := fake.payoutByID[id].OrganizerID
	fake.creds = map[uuid.UUID]db.OrganizerMpCredential{credOrg: {OrganizerID: credOrg, AccessToken: "tok-1"}}
	gw := &fakeGateway{resp: &mp.CreatePayoutResponse{ID: 55, Status: "approved", ExternalReference: id.String()}}
	svc := NewService(fake)

	if err := svc.SendPayout(context.Background(), gw, id); err != nil {
		t.Fatalf("SendPayout: %v", err)
	}
	if len(fake.sentIDs) != 1 || fake.sentIDs[0] != id {
		t.Errorf("sent = %v, want the payout id", fake.sentIDs)
	}
	if len(gw.requests) != 1 || gw.requests[0].Amount != 90.5 {
		t.Errorf("gateway request = %+v, want 90.5 MXN from 9050 cents", gw.requests[0])
	}
	if len(gw.tokens) != 1 || gw.tokens[0] != "tok-1" {
		t.Errorf("gateway tokens = %v, want the organizer token", gw.tokens)
	}
}

func TestSendPayoutNotPending(t *testing.T) {
	id := uuid.New()
	fake := &fakeAdminRepo{payoutByID: map[uuid.UUID]db.OrganizerPayout{
		id: {ID: id, AmountCents: 100, Status: "sent"},
	}}
	gw := &fakeGateway{resp: &mp.CreatePayoutResponse{ID: 1}}
	svc := NewService(fake)

	if err := svc.SendPayout(context.Background(), gw, id); err == nil || !strings.Contains(err.Error(), "not pending") {
		t.Fatalf("err = %v, want not pending", err)
	}
	if len(gw.requests) != 0 || len(fake.sentIDs) != 0 || len(fake.failedIDs) != 0 {
		t.Error("no provider call or status writes for a non-pending payout")
	}
}

func TestSendPayoutGatewayFailureMarkedFailed(t *testing.T) {
	id := uuid.New()
	fake := &fakeAdminRepo{payoutByID: map[uuid.UUID]db.OrganizerPayout{
		id: {ID: id, AmountCents: 9050, Status: "pending"},
	}}
	org := fake.payoutByID[id].OrganizerID
	fake.creds = map[uuid.UUID]db.OrganizerMpCredential{org: {AccessToken: "tok-1"}}
	gw := &fakeGateway{err: errors.New("payout HTTP 502")}
	svc := NewService(fake)

	if err := svc.SendPayout(context.Background(), gw, id); err == nil || !strings.Contains(err.Error(), "502") {
		t.Fatalf("err = %v, want provider failure", err)
	}
	if len(fake.failedIDs) != 1 || fake.failedIDs[0] != id {
		t.Errorf("failed marks = %v, want the payout id", fake.failedIDs)
	}
	if len(fake.failedReason) != 1 || fake.failedReason[0] != "payout HTTP 502" {
		t.Errorf("reason = %q, want the provider error", fake.failedReason)
	}
}

func TestSendPayoutMissingCredentials(t *testing.T) {
	id := uuid.New()
	fake := &fakeAdminRepo{payoutByID: map[uuid.UUID]db.OrganizerPayout{
		id: {ID: id, OrganizerID: uuid.New(), AmountCents: 100, Status: "pending"},
	}}
	gw := &fakeGateway{resp: &mp.CreatePayoutResponse{ID: 1}}
	svc := NewService(fake)

	if err := svc.SendPayout(context.Background(), gw, id); err == nil || !strings.Contains(err.Error(), "no MP credentials") {
		t.Fatalf("err = %v, want missing credentials", err)
	}
	if len(gw.requests) != 0 {
		t.Error("no provider call without credentials")
	}
}

func TestUpsertMPCredentialsMapsParams(t *testing.T) {
	fake := &fakeAdminRepo{}
	svc := NewService(fake)
	org := uuid.New()
	tok := &mp.OAuthTokenResponse{AccessToken: "at", RefreshToken: "rt", UserID: 42, ExpiresIn: 3600}

	if _, err := svc.UpsertMPCredentials(context.Background(), org, tok); err != nil {
		t.Fatalf("UpsertMPCredentials: %v", err)
	}
	if len(fake.upserts) != 1 {
		t.Fatal("upsert not recorded")
	}
	got := fake.upserts[0]
	if got.OrganizerID != org || got.MpUserID != "42" || got.AccessToken != "at" || got.RefreshToken != "rt" {
		t.Errorf("params = %+v", got)
	}
	if time.Until(got.ExpiresAt) < 3500*time.Second {
		t.Errorf("expires_at = %v, want now+3600s", got.ExpiresAt)
	}
}
