package orders

import (
	"context"
	"fmt"
	"log"
	"math"
	"strings"

	"kinetix-api/internal/mp"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	db "kinetix-api/internal/adapters/postgresql/sqlc"
	sqlc "kinetix-api/internal/adapters/postgresql/sqlc"
	"kinetix-api/internal/admin"
	"kinetix-api/internal/store"
)

// OrderRepository is the data access the orders service needs.
type OrderRepository interface {
	Create(ctx context.Context, req CreateOrderRequest) (Order, error)
	GetByID(ctx context.Context, id uuid.UUID) (Order, error)
	GetByRegistrationID(ctx context.Context, registrationID uuid.UUID) (Order, error)
	GetByMercadoPagoPaymentID(ctx context.Context, paymentID string) (Order, error)
	ListByEvent(ctx context.Context, eventID uuid.UUID) ([]Order, error)
	ListByParticipant(ctx context.Context, participantID uuid.UUID) ([]Order, error)
	ListByStatus(ctx context.Context, status string) ([]Order, error)
	UpdatePaymentInfo(ctx context.Context, id uuid.UUID, req UpdateOrderPaymentInfoRequest) (Order, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) (Order, error)
	// EventOrganizerID returns the organizer that owns the event.
	EventOrganizerID(ctx context.Context, eventID uuid.UUID) (uuid.UUID, error)
}

// PayoutRepository is the slice of the admin repository used to record
// organizer payouts after a paid order.
type PayoutRepository interface {
	GetPlatformConfig(ctx context.Context) (db.PlatformConfig, error)
	CreatePayout(ctx context.Context, params db.CreateOrganizerPayoutParams) (db.OrganizerPayout, error)
}

// PaidOrderTx swaps BOTH repositories for transaction-scoped ones for the
// duration of fn (store.ExecTx, ADR-14): the paid transition and its payout
// land atomically.
type PaidOrderTx interface {
	ExecTx(ctx context.Context, fn func(OrderRepository, PayoutRepository) error) error
}

// PaidTxRunner adapts store.ExecTx when block-scoped repositories.
type PaidTxRunner struct {
	store *store.Store
}

// NewPaidTxRunner builds the transactional runner over the shared pool.
func NewPaidTxRunner(s *store.Store) PaidOrderTx { return PaidTxRunner{store: s} }

// ExecTx ( PaidOrderTx ) runs fn with both repositories scoped to a single transaction.
func (r PaidTxRunner) ExecTx(ctx context.Context, fn func(OrderRepository, PayoutRepository) error) error {
	return r.store.ExecTx(ctx, func(q *sqlc.Queries) error {
		orderRepo := &Repository{store: r.store.WithQueries(q)}
		adminRepo := admin.NewRepository(r.store.WithQueries(q))
		return fn(orderRepo, adminRepo)
	})
}

// Service handles business logic for orders
type Service struct {
	repo      OrderRepository
	adminRepo PayoutRepository
	validator *validator.Validate
	// tx (optional) executes the paid transition and payout atomically.
	// When nil, the legacy non-transactional path applies.
	tx PaidOrderTx
	// logPayoutError reports a payout that could not be recorded. The order
	// stays paid; the failure must be visible so it can be reconciled.
	logPayoutError func(order Order, err error)
}

func defaultLogPayoutError(order Order, err error) {
	log.Printf("orders: payout not recorded for order %s (event %s): %v", order.ID, order.EventID, err)
}

// NewService creates a new orders service.
// adminRepo is the admin module's repo (used to read commission and create
// organizer_payouts rows). Can be nil in tests that don't reach ProcessWebhook.
func NewService(repo OrderRepository, adminRepo *admin.Repository, tx PaidOrderTx, validator *validator.Validate) *Service {
	s := &Service{repo: repo, validator: validator, tx: tx, logPayoutError: defaultLogPayoutError}
	if adminRepo != nil { // keep a nil *admin.Repository as a nil interface
		s.adminRepo = adminRepo
	}
	return s
}

// Create creates a new order
func (s *Service) Create(ctx context.Context, req CreateOrderRequest) (Order, error) {
	if err := s.validator.Struct(req); err != nil {
		return Order{}, fmt.Errorf("validation error: %w", err)
	}

	return s.repo.Create(ctx, req)
}

// GetByID retrieves an order by ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (Order, error) {
	return s.repo.GetByID(ctx, id)
}

// GetByRegistrationID retrieves order by registration ID
func (s *Service) GetByRegistrationID(ctx context.Context, registrationID uuid.UUID) (Order, error) {
	return s.repo.GetByRegistrationID(ctx, registrationID)
}

// GetByMercadoPagoPaymentID retrieves order by Mercado Pago payment ID
func (s *Service) GetByMercadoPagoPaymentID(ctx context.Context, paymentID string) (Order, error) {
	return s.repo.GetByMercadoPagoPaymentID(ctx, paymentID)
}

// ListByEvent retrieves orders by event
func (s *Service) ListByEvent(ctx context.Context, eventID uuid.UUID) ([]Order, error) {
	return s.repo.ListByEvent(ctx, eventID)
}

// ListByParticipant retrieves orders by participant
func (s *Service) ListByParticipant(ctx context.Context, participantID uuid.UUID) ([]Order, error) {
	return s.repo.ListByParticipant(ctx, participantID)
}

// ListByStatus retrieves orders by status
func (s *Service) ListByStatus(ctx context.Context, status string) ([]Order, error) {
	return s.repo.ListByStatus(ctx, status)
}

// UpdatePaymentInfo updates order payment info
func (s *Service) UpdatePaymentInfo(ctx context.Context, id uuid.UUID, req UpdateOrderPaymentInfoRequest) (Order, error) {
	if err := s.validator.Struct(req); err != nil {
		return Order{}, fmt.Errorf("validation error: %w", err)
	}

	return s.repo.UpdatePaymentInfo(ctx, id, req)
}

// UpdateStatus updates order status
func (s *Service) UpdateStatus(ctx context.Context, id uuid.UUID, status string) (Order, error) {
	return s.repo.UpdateStatus(ctx, id, status)
}

// ProcessWebhook processes a Mercado Pago payment notification confirmed
// server-side via GetPayment (PLAN-16 T9). It is idempotent: an order that
// is already paid is returned unchanged and no second payout is recorded.
// A nil confirmation or a non-approved payment never marks the order paid.
func (s *Service) ProcessWebhook(ctx context.Context, paymentID string, confirm *mp.PaymentConfirmation) (Order, error) {
	if confirm == nil {
		return Order{}, fmt.Errorf("payment verification required: missing confirmation of %s", paymentID)
	}

	order, err := s.repo.GetByMercadoPagoPaymentID(ctx, paymentID)
	if err != nil {
		return Order{}, fmt.Errorf("order not found: %w", err)
	}
	if order.Status == "paid" {
		return order, nil
	}
	if !strings.EqualFold(confirm.Status, "approved") {
		return order, nil
	}

	// Atomic path: the paid transition and its payout share one transaction
	// so a payout failure leaves the order pending for a later retry.
	if s.tx != nil && s.adminRepo != nil {
		err := s.tx.ExecTx(ctx, func(orderRepo OrderRepository, payoutRepo PayoutRepository) error {
			updated, err := orderRepo.UpdateStatus(ctx, order.ID, "paid")
			if err != nil {
				return fmt.Errorf("failed to update order status: %w", err)
			}
			return s.recordOrganizerPayoutWith(ctx, payoutRepo, updated)
		})
		if err != nil {
			return Order{}, err
		}
		return s.repo.GetByMercadoPagoPaymentID(ctx, paymentID)
	}

	// Legacy best-effort path (kept when no runner is configured).
	updatedOrder, err := s.repo.UpdateStatus(ctx, order.ID, "paid")
	if err != nil {
		return Order{}, fmt.Errorf("failed to update order status: %w", err)
	}

	if s.adminRepo != nil {
		if err := s.recordOrganizerPayoutWith(ctx, s.adminRepo, updatedOrder); err != nil {
			s.reportPayoutError(updatedOrder, err)
		}
	}

	return updatedOrder, nil
}

func (s *Service) reportPayoutError(order Order, err error) {
	if s.logPayoutError != nil {
		s.logPayoutError(order, err)
		return
	}
	defaultLogPayoutError(order, err)
}

// recordOrganizerPayoutWith reads the current commission %, computes the net
// amount in cents, and inserts a pending organizer_payout row for the
// organizer that owns the order's event. It uses the provided
// PayoutRepository (the tx-scoped one when running atomically).
func (s *Service) recordOrganizerPayoutWith(ctx context.Context, payoutRepo PayoutRepository, order Order) error {
	organizerID, err := s.repo.EventOrganizerID(ctx, order.EventID)
	if err != nil {
		return fmt.Errorf("resolve event organizer: %w", err)
	}
	cfg, err := payoutRepo.GetPlatformConfig(ctx)
	if err != nil {
		return fmt.Errorf("read platform config: %w", err)
	}
	f8, err := cfg.CommissionPct.Float64Value()
	if err != nil {
		return fmt.Errorf("read commission: %w", err)
	}
	totalCents := int64(math.Round(order.Amount * 100)) // order.Amount is in major units (e.g. MXN)
	netCents := admin.CalculatePayoutAmount(totalCents, f8.Float64)

	if _, err := payoutRepo.CreatePayout(ctx, db.CreateOrganizerPayoutParams{
		OrganizerID:   organizerID,
		OrderID:       order.ID,
		AmountCents:   netCents,
		Currency:      order.Currency,
		CommissionPct: cfg.CommissionPct,
	}); err != nil {
		return fmt.Errorf("create payout: %w", err)
	}
	return nil
}
