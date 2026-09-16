package orders

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handler handles HTTP requests for orders
type Handler struct {
	service       *Service
	webhookSecret string
}

// NewHandler creates a new orders handler
func NewHandler(service *Service, webhookSecret string) *Handler {
	return &Handler{
		service:       service,
		webhookSecret: webhookSecret,
	}
}

// Routes registers order routes
func (h *Handler) Routes(r chi.Router) {
	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Get("/{id}", h.GetByID)
	r.Put("/{id}/payment", h.UpdatePaymentInfo)
	r.Put("/{id}/status", h.UpdateStatus)
}

// Create handles POST /orders
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	order, err := h.service.Create(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		eventID, _ := uuid.Parse(eventIDStr)
		orders, err = h.service.ListByEvent(r.Context(), eventID)
	} else if participantIDStr != "" {
		participantID, _ := uuid.Parse(participantIDStr)
		orders, err = h.service.ListByParticipant(r.Context(), participantID)
	} else if status != "" {
		orders, err = h.service.ListByStatus(r.Context(), status)
	} else {
		http.Error(w, "event_id, participant_id, or status is required", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(OrdersListResponse{
		Message: "Orders retrieved successfully",
		Data:    orders,
	})
}

// GetByID handles GET /orders/{id}
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	order, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
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
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var req UpdateOrderPaymentInfoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	order, err := h.service.UpdatePaymentInfo(r.Context(), id, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(OrderResponse{
		Message: "Order payment updated successfully",
		Data:    order,
	})
}

// UpdateStatus handles PUT /orders/{id}/status
func (h *Handler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	order, err := h.service.UpdateStatus(r.Context(), id, req.Status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	// Validate webhook signature
	if err := ValidateWebhookRequest(h.webhookSecret, r, body); err != nil {
		if err == ErrMissingSignature {
			http.Error(w, "missing webhook signature", http.StatusUnauthorized)
			return
		}
		http.Error(w, "invalid webhook signature", http.StatusForbidden)
		return
	}

	// Parse payload
	var webhook PaymentWebhookRequest
	if err := json.Unmarshal(body, &webhook); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if webhook.Data.ID == "" {
		http.Error(w, "payment ID not found", http.StatusBadRequest)
		return
	}

	order, err := h.service.ProcessWebhook(r.Context(), webhook.Data.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
