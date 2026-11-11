package participants

import (
	"encoding/json"
	"fmt"
	"kinetix-api/internal/httpx"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// Handler handles HTTP requests for participants
type Handler struct {
	service *Service
}

// NewHandler creates a new participants handler
func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// Routes registers participant routes
func (h *Handler) Routes(r chi.Router) {
	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Get("/{id}", h.GetByID)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
}

// Create handles POST /participants
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateParticipantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid input: %w", httpx.ErrInvalidInput))
		return
	}

	participant, err := h.service.Create(r.Context(), req)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	json.NewEncoder(w).Encode(ParticipantResponse{
		Message: "Participant created successfully",
		Data:    participant,
	})
}

// List handles GET /participants
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	if limit == 0 {
		limit = 10
	}

	participants, total, err := h.service.List(r.Context(), limit, offset)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	json.NewEncoder(w).Encode(ParticipantsListResponse{
		Message: "Participants retrieved successfully",
		Data:    participants,
		Total:   total,
	})
}

// GetByID handles GET /participants/{id}
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.UUIDParam(w, r, "id")
	if !ok {
		return
	}

	participant, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("not found: %w", httpx.ErrNotFound))
		return
	}

	json.NewEncoder(w).Encode(ParticipantResponse{
		Message: "Participant retrieved successfully",
		Data:    participant,
	})
}

// Update handles PUT /participants/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.UUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req UpdateParticipantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid input: %w", httpx.ErrInvalidInput))
		return
	}

	participant, err := h.service.Update(r.Context(), id, req)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	json.NewEncoder(w).Encode(ParticipantResponse{
		Message: "Participant updated successfully",
		Data:    participant,
	})
}

// Delete handles DELETE /participants/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.UUIDParam(w, r, "id")
	if !ok {
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	json.NewEncoder(w).Encode(GenericResponse{
		Message: "Participant deleted successfully",
	})
}
