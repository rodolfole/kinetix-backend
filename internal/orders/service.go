package orders

import (
	"context"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// Service handles business logic for orders
type Service struct {
	repo      *Repository
	validator *validator.Validate
}

// NewService creates a new orders service
func NewService(repo *Repository, validator *validator.Validate) *Service {
	return &Service{
		repo:      repo,
		validator: validator,
	}
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

// ProcessWebhook processes Mercado Pago webhook
func (s *Service) ProcessWebhook(ctx context.Context, paymentID string) (Order, error) {
	order, err := s.repo.GetByMercadoPagoPaymentID(ctx, paymentID)
	if err != nil {
		return Order{}, fmt.Errorf("order not found: %w", err)
	}

	// Update order status based on payment
	updatedOrder, err := s.repo.UpdateStatus(ctx, order.ID, "paid")
	if err != nil {
		return Order{}, fmt.Errorf("failed to update order status: %w", err)
	}

	return updatedOrder, nil
}
