package orders

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	repo "kinetix-api/internal/adapters/postgresql/sqlc"
	"kinetix-api/internal/database/convert"
	"kinetix-api/internal/store"
)

// Repository handles database operations for orders
type Repository struct {
	store *store.Store
}

// NewRepository creates a new order repository
func NewRepository(s *store.Store) *Repository {
	return &Repository{store: s}
}

// Alias methods
func (r *Repository) GetOrderByID(ctx context.Context, id uuid.UUID) (Order, error) {
	return r.GetByID(ctx, id)
}

func (r *Repository) ListOrdersByEventID(ctx context.Context, eventID uuid.UUID) ([]Order, error) {
	return r.ListByEvent(ctx, eventID)
}

func (r *Repository) ListOrdersByParticipantID(ctx context.Context, participantID uuid.UUID) ([]Order, error) {
	return r.ListByParticipant(ctx, participantID)
}

func (r *Repository) ListOrdersByStatus(ctx context.Context, status string) ([]Order, error) {
	return r.ListByStatus(ctx, status)
}

// Create inserts a new order
func (r *Repository) Create(ctx context.Context, req CreateOrderRequest) (Order, error) {
	order, err := r.store.Queries.CreateOrder(ctx, repo.CreateOrderParams{
		RegistrationID:          req.RegistrationID,
		ParticipantID:           req.ParticipantID,
		EventID:                 req.EventID,
		Amount:                  convert.ToPgNumeric(req.Amount),
		Currency:                req.Currency,
		Status:                  convert.ToPgText("pending"),
		PaymentMethod:           convert.ToPgText(""),
		MercadoPagoPaymentID:    convert.ToPgText(""),
		MercadoPagoPreferenceID: convert.ToPgText(""),
		TransactionDate:         convert.ToPgTime(time.Now()),
	})
	if err != nil {
		return Order{}, fmt.Errorf("failed to create order: %w", err)
	}
	return toOrder(order), nil
}

// GetByID retrieves an order by ID
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (Order, error) {
	order, err := r.store.Queries.GetOrder(ctx, id)
	if err != nil {
		return Order{}, fmt.Errorf("order not found: %w", err)
	}
	return toOrder(order), nil
}

// GetByRegistrationID retrieves order by registration ID
func (r *Repository) GetByRegistrationID(ctx context.Context, registrationID uuid.UUID) (Order, error) {
	order, err := r.store.Queries.GetOrderByRegistrationId(ctx, registrationID)
	if err != nil {
		return Order{}, fmt.Errorf("order not found: %w", err)
	}
	return toOrder(order), nil
}

// GetByMercadoPagoPaymentID retrieves order by Mercado Pago payment ID
func (r *Repository) GetByMercadoPagoPaymentID(ctx context.Context, paymentID string) (Order, error) {
	order, err := r.store.Queries.GetOrderByMercadoPagoPaymentId(ctx, convert.ToPgText(paymentID))
	if err != nil {
		return Order{}, fmt.Errorf("order not found: %w", err)
	}
	return toOrder(order), nil
}

// ListByEvent retrieves orders by event
func (r *Repository) ListByEvent(ctx context.Context, eventID uuid.UUID) ([]Order, error) {
	orders, err := r.store.Queries.ListOrdersByEvent(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to list orders: %w", err)
	}

	result := make([]Order, len(orders))
	for i, o := range orders {
		result[i] = toOrder(o)
	}
	return result, nil
}

// ListByParticipant retrieves orders by participant
func (r *Repository) ListByParticipant(ctx context.Context, participantID uuid.UUID) ([]Order, error) {
	orders, err := r.store.Queries.ListOrdersByParticipant(ctx, participantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list orders: %w", err)
	}

	result := make([]Order, len(orders))
	for i, o := range orders {
		result[i] = toOrder(o)
	}
	return result, nil
}

// ListByStatus retrieves orders by status
func (r *Repository) ListByStatus(ctx context.Context, status string) ([]Order, error) {
	orders, err := r.store.Queries.ListOrdersByStatus(ctx, convert.ToPgText(status))
	if err != nil {
		return nil, fmt.Errorf("failed to list orders: %w", err)
	}

	result := make([]Order, len(orders))
	for i, o := range orders {
		result[i] = toOrder(o)
	}
	return result, nil
}

// UpdatePaymentInfo updates order payment info
func (r *Repository) UpdatePaymentInfo(ctx context.Context, id uuid.UUID, req UpdateOrderPaymentInfoRequest) (Order, error) {
	order, err := r.store.Queries.UpdateOrderPaymentInfo(ctx, repo.UpdateOrderPaymentInfoParams{
		ID:                   id,
		PaymentMethod:        convert.ToPgText(req.PaymentMethod),
		MercadoPagoPaymentID: convert.ToPgText(req.MercadoPagoPaymentID),
		TransactionDate:      convert.ToPgTime(req.TransactionDate),
		Status:               convert.ToPgText(req.Status),
	})
	if err != nil {
		return Order{}, fmt.Errorf("failed to update order: %w", err)
	}
	return toOrder(order), nil
}

// UpdateStatus updates order status
func (r *Repository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) (Order, error) {
	order, err := r.store.Queries.UpdateOrderStatus(ctx, repo.UpdateOrderStatusParams{
		ID:     id,
		Status: convert.ToPgText(status),
	})
	if err != nil {
		return Order{}, fmt.Errorf("failed to update order status: %w", err)
	}
	return toOrder(order), nil
}

// Delete removes an order
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	err := r.store.Queries.DeleteOrder(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete order: %w", err)
	}
	return nil
}

// ============ Model conversion ============

func toOrder(o repo.Order) Order {
	return Order{
		ID:                      o.ID,
		RegistrationID:          o.RegistrationID,
		ParticipantID:           o.ParticipantID,
		EventID:                 o.EventID,
		Amount:                  convert.PgNumericToFloat64(o.Amount),
		Currency:                o.Currency,
		Status:                  convert.PgTextToString(o.Status),
		PaymentMethod:           convert.ToStringPtr(convert.PgTextToString(o.PaymentMethod)),
		MercadoPagoPaymentID:    convert.ToStringPtr(convert.PgTextToString(o.MercadoPagoPaymentID)),
		MercadoPagoPreferenceID: convert.ToStringPtr(convert.PgTextToString(o.MercadoPagoPreferenceID)),
		TransactionDate:         convert.ToTimePtr(convert.PgTimeToTime(o.TransactionDate)),
		CreatedAt:               convert.PgTimeToTime(o.CreatedAt),
		UpdatedAt:               convert.PgTimeToTime(o.UpdatedAt),
	}
}
