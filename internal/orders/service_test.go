package orders

import (
	"context"
	"strconv"
	"testing"

	"github.com/google/uuid"

	"github.com/jackc/pgx/v5/pgtype"
	"kinetix-api/internal/mp"

	db "kinetix-api/internal/adapters/postgresql/sqlc"
)

// fakeOrderRepo implements OrderRepository and records writes.
type fakeOrderRepo struct {
	OrderRepository // unused methods panic if reached
	order           Order
	organizerID     uuid.UUID
	organizerErr    error
	statusUpdates   []string
}

func (f *fakeOrderRepo) GetByMercadoPagoPaymentID(ctx context.Context, paymentID string) (Order, error) {
	return f.order, nil
}

func (f *fakeOrderRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) (Order, error) {
	f.statusUpdates = append(f.statusUpdates, status)
	o := f.order
	o.Status = status
	return o, nil
}

func (f *fakeOrderRepo) EventOrganizerID(ctx context.Context, eventID uuid.UUID) (uuid.UUID, error) {
	return f.organizerID, f.organizerErr
}

// fakePayoutRepo implements PayoutRepository and records created payouts.
type fakePayoutRepo struct {
	commissionPct float64
	createErr     error
	created       []db.CreateOrganizerPayoutParams
}

func (f *fakePayoutRepo) GetPlatformConfig(ctx context.Context) (db.PlatformConfig, error) {
	var n pgtype.Numeric
	if err := n.Scan(fmtFloat(f.commissionPct)); err != nil {
		return db.PlatformConfig{}, err
	}
	return db.PlatformConfig{CommissionPct: n}, nil
}

func (f *fakePayoutRepo) CreatePayout(ctx context.Context, p db.CreateOrganizerPayoutParams) (db.OrganizerPayout, error) {
	f.created = append(f.created, p)
	return db.OrganizerPayout{}, f.createErr
}

func fmtFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', 2, 64)
}

func pendingOrder() Order {
	return Order{
		ID:            uuid.New(),
		ParticipantID: uuid.New(),
		EventID:       uuid.New(),
		Amount:        500,
		Currency:      "MXN",
		Status:        "pending",
	}
}

func TestProcessWebhook_PayoutGoesToEventOrganizer(t *testing.T) {
	order := pendingOrder()
	organizerID := uuid.New()
	repo := &fakeOrderRepo{order: order, organizerID: organizerID}
	parentPayout := &fakePayoutRepo{commissionPct: 10}
	tx := &fakePaidTx{parentOrder: repo, parentPayout: parentPayout}
	svc := &Service{repo: repo, adminRepo: parentPayout, tx: tx}

	got, err := svc.ProcessWebhook(context.Background(), "mp-1", approvedConfirmation())
	if err != nil {
		t.Fatalf("ProcessWebhook: %v", err)
	}
	if got.Status != "paid" {
		t.Errorf("status = %q, want paid", got.Status)
	}
	if len(parentPayout.created) != 1 {
		t.Fatalf("payouts created = %d, want 1", len(parentPayout.created))
	}
	p := parentPayout.created[0]
	if p.OrganizerID != organizerID {
		t.Errorf("payout organizer = %s, want event organizer %s (participant was %s)", p.OrganizerID, organizerID, order.ParticipantID)
	}
	if p.OrderID != order.ID || p.AmountCents != 45000 || p.Currency != "MXN" {
		t.Errorf("payout = %+v, want order %s, 45000 cents, MXN", p, order.ID)
	}
}

func TestProcessWebhook_AlreadyPaidIsIdempotent(t *testing.T) {
	order := pendingOrder()
	order.Status = "paid"
	repo := &fakeOrderRepo{order: order, organizerID: uuid.New()}
	parentPayout := &fakePayoutRepo{commissionPct: 10}
	tx := &fakePaidTx{parentOrder: repo, parentPayout: parentPayout}
	svc := &Service{repo: repo, adminRepo: parentPayout, tx: tx}

	got, err := svc.ProcessWebhook(context.Background(), "mp-1", approvedConfirmation())
	if err != nil {
		t.Fatalf("ProcessWebhook: %v", err)
	}
	if got.ID != order.ID || got.Status != "paid" {
		t.Errorf("got %+v, want the existing paid order", got)
	}
	if len(repo.statusUpdates) != 0 {
		t.Errorf("status updates = %v, want none", repo.statusUpdates)
	}
	if len(parentPayout.created) != 0 {
		t.Errorf("payouts created = %d, want 0", len(parentPayout.created))
	}
	if tx.invoked != 0 {
		t.Errorf("an already-paid order must not enter the transaction, invoked %d", tx.invoked)
	}
}

func TestProcessWebhook_PayoutCentsAreRoundedNotTruncated(t *testing.T) {
	order := pendingOrder()
	order.Amount = 19.99 // 19.99*100 == 1998.9999... in float64
	repo := &fakeOrderRepo{order: order, organizerID: uuid.New()}
	parentPayout := &fakePayoutRepo{commissionPct: 0}
	tx := &fakePaidTx{parentOrder: repo, parentPayout: parentPayout}
	svc := &Service{repo: repo, adminRepo: parentPayout, tx: tx}

	if _, err := svc.ProcessWebhook(context.Background(), "mp-1", approvedConfirmation()); err != nil {
		t.Fatalf("ProcessWebhook: %v", err)
	}
	if len(parentPayout.created) != 1 || parentPayout.created[0].AmountCents != 1999 {
		t.Fatalf("payouts = %+v, want one payout of 1999 cents", parentPayout.created)
	}
}

func approvedConfirmation() *mp.PaymentConfirmation {
	return &mp.PaymentConfirmation{ID: 123, Status: "approved"}
}
