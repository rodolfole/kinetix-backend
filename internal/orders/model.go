package orders

import (
	"time"

	"github.com/google/uuid"
)

// Order represents an order entity
type Order struct {
	ID                      uuid.UUID  `json:"id"`
	RegistrationID          uuid.UUID  `json:"registration_id"`
	ParticipantID           uuid.UUID  `json:"participant_id"`
	EventID                 uuid.UUID  `json:"event_id"`
	Amount                  float64    `json:"amount"`
	Currency                string     `json:"currency"`
	Status                  string     `json:"status"`
	PaymentMethod           *string    `json:"payment_method,omitempty"`
	MercadoPagoPaymentID    *string    `json:"mercado_pago_payment_id,omitempty"`
	MercadoPagoPreferenceID *string    `json:"mercado_pago_preference_id,omitempty"`
	TransactionDate         *time.Time `json:"transaction_date,omitempty"`
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`
}

// CreateOrderRequest represents the request body for creating an order
type CreateOrderRequest struct {
	RegistrationID uuid.UUID `json:"registration_id" validate:"required"`
	ParticipantID  uuid.UUID `json:"participant_id" validate:"required"`
	EventID        uuid.UUID `json:"event_id" validate:"required"`
	Amount         float64   `json:"amount" validate:"required,gt=0"`
	Currency       string    `json:"currency" validate:"required,len=3"`
}

// UpdateOrderPaymentInfoRequest represents the request body for updating payment info
type UpdateOrderPaymentInfoRequest struct {
	PaymentMethod        string    `json:"payment_method"`
	MercadoPagoPaymentID string    `json:"mercado_pago_payment_id"`
	TransactionDate      time.Time `json:"transaction_date"`
	Status               string    `json:"status"`
}

// PaymentWebhookRequest represents MercadoPago webhook payload
type PaymentWebhookRequest struct {
	Action        string `json:"action"`
	ApplicationID string `json:"application_id"`
	Data          struct {
		ID string `json:"id"`
	} `json:"data"`
	Type string `json:"type"`
}

// PaymentResponse represents payment initiation response
type PaymentResponse struct {
	Message        string `json:"message"`
	OrderID        string `json:"order_id"`
	MercadoPagoURL string `json:"mercado_pago_url"`
	PreferenceID   string `json:"preference_id"`
}

// OrderResponse represents the response structure
type OrderResponse struct {
	Message string `json:"message"`
	Data    Order  `json:"data"`
}

// OrdersListResponse represents a list of orders
type OrdersListResponse struct {
	Message string  `json:"message"`
	Data    []Order `json:"data"`
}
