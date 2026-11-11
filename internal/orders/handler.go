package orders

import (
	"fmt"
	"strings"

	"encoding/json"
	"errors"
	"io"
	"net/http"

	jsonlib "kinetix-api/internal/json"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"kinetix-api/internal/httpx"
	kinemiddleware "kinetix-api/internal/middleware"

	"kinetix-api/internal/mp"
)

// PaymentVerifier confirms payment status with the MercadoPago API before
// the webhook trusts the notification (PLAN-16 T9).
type PaymentVerifier interface {
	GetPayment(paymentID string) (*mp.PaymentConfirmation, error)
}

// Handler handles HTTP requests for orders
type Handler struct {
	service       *Service
	webhookSecret string
	verifier      PaymentVerifier
}

// NewHandler creates a new orders handler
func NewHandler(service *Service, webhookSecret string, verifier PaymentVerifier) *Handler {
	return &Handler{
		service:       service,
		webhookSecret: webhookSecret,
		verifier:      verifier,
	}
}

// verifyPayment asks the PaymentVerifier for the payment status. It writes
// the error response itself and returns the confirmación only on success.
func (h *Handler) verifyPayment(w http.ResponseWriter, r *http.Request, paymentID string) (*mp.PaymentConfirmation, error) {
	if h.verifier == nil {
		jsonlib.WriteError(w, http.StatusServiceUnavailable, "payment verifier not configured")
		return nil, fmt.Errorf("payment verifier not configured")
	}
	if confirm, err := h.verifier.GetPayment(paymentID); err != nil {
		jsonlib.WriteError(w, http.StatusServiceUnavailable, "payment verification failed")
		return nil, err
	} else if !strings.EqualFold(confirm.Status, "approved") {
		jsonlib.Write(w, http.StatusOK, map[string]any{
			"message":        "payment not approved; the order stays unchanged",
			"payment_status": confirm.Status,
		})
		return nil, nil
	} else {
		return confirm, nil
	}
}

// Routes registers order routes. Mutating a payment belongs to the
// platform: those two routes require the admin role (PLAN-16 T8).
func (h *Handler) Routes(r chi.Router) {
	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Get("/{id}", h.GetByID)
	r.With(kinemiddleware.RequireRole("admin")).Put("/{id}/payment", h.UpdatePaymentInfo)
	r.With(kinemiddleware.RequireRole("admin")).Put("/{id}/status", h.UpdateStatus)
}

// Create handles POST /orders
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteValidation(w, err)
		return
	}

	order, err := h.service.Create(r.Context(), req)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	json.NewEncoder(w).Encode(OrderResponse{
		Message: "Order created successfully",
		Data:    order,
	})
}

// List handles GET /orders?event_id=xxx&participant_id=xxx&status=xxx
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	eventIDStr := r.URL.Query().Get("event_id")
	participantIDStr := r.URL.Query().Get("participant_id")
	status := r.URL.Query().Get("status")

	var orders []Order
	var err error

	if eventIDStr != "" {
		eventID, parseErr := uuid.Parse(eventIDStr)
		if parseErr != nil {
			httpx.WriteErrorCode(w, http.StatusBadRequest, "INVALID_ID", "invalid event_id")
			return
		}
		orders, err = h.service.ListByEvent(r.Context(), eventID)
	} else if participantIDStr != "" {
		participantID, parseErr := uuid.Parse(participantIDStr)
		if parseErr != nil {
			httpx.WriteErrorCode(w, http.StatusBadRequest, "INVALID_ID", "invalid participant_id")
			return
		}
		orders, err = h.service.ListByParticipant(r.Context(), participantID)
	} else if status != "" {
		orders, err = h.service.ListByStatus(r.Context(), status)
	} else {
		httpx.WriteErrorCode(w, http.StatusBadRequest, "VALIDATION_FAILED", "event_id, participant_id, or status is required")
		return
	}

	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	json.NewEncoder(w).Encode(OrdersListResponse{
		Message: "Orders retrieved successfully",
		Data:    orders,
	})
}

// GetByID handles GET /orders/{id}
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.UUIDParam(w, r, "id")
	if !ok {
		return
	}

	order, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	json.NewEncoder(w).Encode(OrderResponse{
		Message: "Order retrieved successfully",
		Data:    order,
	})
}

// UpdatePaymentInfo handles PUT /orders/{id}/payment
func (h *Handler) UpdatePaymentInfo(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid input: %w", httpx.ErrInvalidInput))
		return
	}

	var req UpdateOrderPaymentInfoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid input: %w", httpx.ErrInvalidInput))
		return
	}

	order, err := h.service.UpdatePaymentInfo(r.Context(), id, req)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	json.NewEncoder(w).Encode(OrderResponse{
		Message: "Order payment updated successfully",
		Data:    order,
	})
}

// UpdateStatus handles PUT /orders/{id}/status
func (h *Handler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.UUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req struct {
		Status string `json:"status"`
	}
	if !httpx.Read(w, r, &req) {
		return
	}

	order, err := h.service.UpdateStatus(r.Context(), id, req.Status)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	json.NewEncoder(w).Encode(OrderResponse{
		Message: "Order status updated successfully",
		Data:    order,
	})
}

// PaymentWebhook handles POST /orders/webhook
func (h *Handler) PaymentWebhook(w http.ResponseWriter, r *http.Request) {
	// Read body for signature validation
	body, err := io.ReadAll(r.Body)
	if err != nil {
		httpx.WriteErrorCode(w, http.StatusBadRequest, "VALIDATION_FAILED", "failed to read request body")
		return
	}

	// Parse payload first: MercadoPago signs over the notification data id.
	var webhook PaymentWebhookRequest
	if err := json.Unmarshal(body, &webhook); err != nil {
		httpx.WriteErrorCode(w, http.StatusBadRequest, "VALIDATION_FAILED", "invalid request body")
		return
	}

	if webhook.Data.ID == "" {
		httpx.WriteErrorCode(w, http.StatusBadRequest, "VALIDATION_FAILED", "payment id not found")
		return
	}

	// Validate webhook signature against the parsed notification id
	if err := ValidateWebhookRequest(h.webhookSecret, r, body, webhook.Data.ID); err != nil {
		if errors.Is(err, ErrWebhookNotConfigured) {
			jsonlib.WriteError(w, http.StatusServiceUnavailable, ErrWebhookNotConfigured.Error())
			return
		}
		if err == ErrMissingSignature {
			jsonlib.WriteError(w, http.StatusUnauthorized, "missing webhook signature")
			return
		}
		jsonlib.WriteError(w, http.StatusForbidden, "invalid webhook signature")
		return
	}

	// Confirm the payment with MercadoPago: the notification alone is untrusted.
	confirm, vErr := h.verifyPayment(w, r, webhook.Data.ID)
	if vErr != nil {
		return
	}

	order, err := h.service.ProcessWebhook(r.Context(), webhook.Data.ID, confirm)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	json.NewEncoder(w).Encode(OrderResponse{
		Message: "Webhook processed successfully",
		Data:    order,
	})
}

// GenericResponse for simple responses
type GenericResponse struct {
	Message string `json:"message"`
}
