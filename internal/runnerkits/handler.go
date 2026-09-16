package runnerkits

import (
	"net/http"
	"strings"

	"kinetix-api/internal/json"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

func sanitize(s string) string {
	// Reemplaza comillas por caracteres seguros para no romper el JSON inline
	return strings.ReplaceAll(s, `"`, `'`)
}

// Handler handles HTTP requests for runner kit items
type Handler struct {
	service   *Service
	validator *validator.Validate
}

// NewHandler creates a new runner kit items handler
func NewHandler(service *Service, validator *validator.Validate) *Handler {
	return &Handler{
		service:   service,
		validator: validator,
	}
}

// Routes registers all runner kit item routes.
// Mounted under /api/v1/runner-kits (protected).
func (h *Handler) Routes(r chi.Router) {
	r.Post("/", h.Create)
	r.Get("/event/{event_id}", h.ListByEvent)
	r.Get("/{id}", h.GetByID)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
}

// ── Handlers ───────────────────────────────────────────────────────────────

// Create handles POST /runner-kits
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRunnerKitItemRequest
	if err := json.Read(r, &req); err != nil {
		http.Error(w, `{"error":"invalid request body: `+sanitize(err.Error())+`"}`, http.StatusBadRequest)
		return
	}

	item, err := h.service.Create(r.Context(), req)
	if err != nil {
		http.Error(w, `{"error":"`+sanitize(err.Error())+`"}`, http.StatusBadRequest)
		return
	}

	json.Write(w, http.StatusCreated, GenericResponse{
		Message: "Runner kit item created successfully",
		Data:    item,
	})
}

// GetByID handles GET /runner-kits/{id}
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid runner kit item ID"}`, http.StatusBadRequest)
		return
	}

	item, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"runner kit item not found"}`, http.StatusNotFound)
		return
	}

	json.Write(w, http.StatusOK, GenericResponse{
		Message: "Runner kit item retrieved successfully",
		Data:    item,
	})
}

// ListByEvent handles GET /runner-kits/event/{event_id}
func (h *Handler) ListByEvent(w http.ResponseWriter, r *http.Request) {
	eventID, err := uuid.Parse(chi.URLParam(r, "event_id"))
	if err != nil {
		http.Error(w, `{"error":"invalid event ID"}`, http.StatusBadRequest)
		return
	}

	items, err := h.service.ListByEvent(r.Context(), eventID)
	if err != nil {
		http.Error(w, `{"error":"failed to list runner kit items"}`, http.StatusInternalServerError)
		return
	}

	json.Write(w, http.StatusOK, GenericResponse{
		Message: "Runner kit items retrieved successfully",
		Data:    items,
	})
}

// Update handles PUT /runner-kits/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid runner kit item ID"}`, http.StatusBadRequest)
		return
	}

	var req UpdateRunnerKitItemRequest
	if err := json.Read(r, &req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	item, err := h.service.Update(r.Context(), id, req)
	if err != nil {
		json.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	json.Write(w, http.StatusOK, GenericResponse{
		Message: "Runner kit item updated successfully",
		Data:    item,
	})
}

// Delete handles DELETE /runner-kits/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid runner kit item ID"}`, http.StatusBadRequest)
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		json.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	json.Write(w, http.StatusOK, map[string]string{
		"message": "Runner kit item deleted successfully",
	})
}
