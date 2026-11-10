package orders

import (
	"context"
	"errors"
	"testing"
)

func TestProcessWebhook_PayoutFailureIsReportedNotSwallowed(t *testing.T) {
	order := pendingOrder()
	repo := &fakeOrderRepo{order: order, organizerErr: errors.New("event lookup failed")}
	payouts := &fakePayoutRepo{commissionPct: 10}
	var logged []error
	svc := &Service{repo: repo, adminRepo: payouts, logPayoutError: func(o Order, err error) { logged = append(logged, err) }}

	got, err := svc.ProcessWebhook(context.Background(), "mp-1")
	if err != nil {
		t.Fatalf("order is paid even if the payout fails; got error %v", err)
	}
	if got.Status != "paid" {
		t.Errorf("status = %q, want paid", got.Status)
	}
	if len(payouts.created) != 0 {
		t.Errorf("no payout must be written without a resolved organizer, got %d", len(payouts.created))
	}
	if len(logged) != 1 {
		t.Fatalf("payout failures logged = %d, want 1", len(logged))
	}
}
