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
	"kinetix-api/internal/admin"
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

// Service handles business logic for orders
type Service struct {
	repo      OrderRepository
	adminRepo PayoutRepository
	validator *validator.Validate
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
func NewService(repo OrderRepository, adminRepo *admin.Repository, validator *validator.Validate) *Service {
	s := &Service{repo: repo, validator: validator, logPayoutError: defaultLogPayoutError}
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

	updatedOrder, err := s.repo.UpdateStatus(ctx, order.ID, "paid")
	if err != nil {
		return Order{}, fmt.Errorf("failed to update order status: %w", err)
	}

	// After payment, record a pending organizer_payout (net of commission).
	// The order is already paid, so a payout failure is reported, not returned.
	if s.adminRepo != nil {
		if err := s.recordOrganizerPayout(ctx, updatedOrder); err != nil {
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

// recordOrganizerPayout reads the current commission %, computes the net
// amount in cents, and inserts a pending organizer_payout row for the
// organizer that owns the order's event.
func (s *Service) recordOrganizerPayout(ctx context.Context, order Order) error {
	organizerID, err := s.repo.EventOrganizerID(ctx, order.EventID)
	if err != nil {
		return fmt.Errorf("resolve event organizer: %w", err)
	}
	cfg, err := s.adminRepo.GetPlatformConfig(ctx)
	if err != nil {
		return fmt.Errorf("read platform config: %w", err)
	}
	f8, err := cfg.CommissionPct.Float64Value()
	if err != nil {
		return fmt.Errorf("read commission: %w", err)
	}
	totalCents := int64(math.Round(order.Amount * 100)) // order.Amount is in major units (e.g. MXN)
	netCents := admin.CalculatePayoutAmount(totalCents, f8.Float64)

	if _, err := s.adminRepo.CreatePayout(ctx, db.CreateOrganizerPayoutParams{
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
