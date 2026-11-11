package orders

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

// fakePaidTx mimics orders.PaidOrderTx semantics: children are scoped to a
// single transaction; writes merge into the parents only when fn succeeds.
type fakePaidTx struct {
	parentOrder  *fakeOrderRepo  // the pre-validated (pending) order view
	parentPayout *fakePayoutRepo // commission + merge target
	childErr     error           // preset error returned by child payout writes

	invoked int
	lastErr error
}

func (f *fakePaidTx) ExecTx(ctx context.Context, fn func(OrderRepository, PayoutRepository) error) error {
	f.invoked++
	orderChild := &fakeOrderRepo{order: f.parentOrder.order, organizerID: f.parentOrder.organizerID}
	payoutChild := &fakePayoutRepo{commissionPct: f.parentPayout.commissionPct, createErr: f.childErr}
	if err := fn(orderChild, payoutChild); err != nil {
		f.lastErr = err
		return err // rollback: nothing is merged
	}
	f.parentOrder.statusUpdates = append(f.parentOrder.statusUpdates, orderChild.statusUpdates...)
	f.parentPayout.created = append(f.parentPayout.created, payoutChild.created...)
	if len(orderChild.statusUpdates) > 0 {
		f.parentOrder.order.Status = orderChild.statusUpdates[len(orderChild.statusUpdates)-1]
	}
	return nil
}

// TestProcessWebhook_PayoutFailureRollsBackPaidTransition pins the T10
// contract: one transaction covers the paid transition and the payout; a
// failing payout leaves the order pending so MercadoPago can retry.
func TestProcessWebhook_PayoutFailureRollsBackPaidTransition(t *testing.T) {
	repo := &fakeOrderRepo{order: pendingOrder(), organizerID: uuid.New()}
	parentPayout := &fakePayoutRepo{commissionPct: 10}
	tx := &fakePaidTx{parentOrder: repo, parentPayout: parentPayout, childErr: errors.New("payout insert failed")}
	var logged []error
	svc := &Service{repo: repo, adminRepo: parentPayout, tx: tx, logPayoutError: func(o Order, err error) { logged = append(logged, err) }}

	if _, err := svc.ProcessWebhook(context.Background(), "mp-1", approvedConfirmation()); err == nil {
		t.Fatal("a failing payout must fail the webhook so MercadoPago retries")
	}
	if len(repo.statusUpdates) != 0 {
		t.Errorf("status rolled back, got updates %v", repo.statusUpdates)
	}
	if len(parentPayout.created) != 0 {
		t.Errorf("payout rolled back, created %d", len(parentPayout.created))
	}
	if len(logged) != 0 {
		t.Errorf("no payout side log when the tx rolled back, got %v", logged)
	}
}

func TestProcessWebhook_PayoutFailureLoggedNotSwallowedIfTxMissing(t *testing.T) {
	// Legacy path without a configured runner: the payout failure is logged,
	// never swallowed, but the order stays paid (pre-T10 semantics).
	repo := &fakeOrderRepo{order: pendingOrder(), organizerErr: errors.New("event lookup failed")}
	payouts := &fakePayoutRepo{commissionPct: 10}
	var logged []error
	svc := &Service{repo: repo, adminRepo: payouts, logPayoutError: func(o Order, err error) { logged = append(logged, err) }}

	got, err := svc.ProcessWebhook(context.Background(), "mp-1", approvedConfirmation())
	if err != nil {
		t.Fatalf("legacy path keeps the order paid; got %v", err)
	}
	if got.Status != "paid" || len(payouts.created) != 0 || len(logged) != 1 {
		t.Fatalf("legacy best-effort results: %+v err=%v logged=%d", got, err, len(logged))
	}
}
