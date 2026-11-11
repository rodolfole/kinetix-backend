package runnerkits

import (
	"fmt"
	"kinetix-api/internal/httpx"
	kinemiddleware "kinetix-api/internal/middleware"
	"net/http"
	"strings"

	"kinetix-api/internal/json"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
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
		httpx.WriteErrorCode(w, http.StatusBadRequest, "VALIDATION_FAILED", "invalid request body")
		return
	}

	claims, _ := kinemiddleware.ClaimsFrom(r.Context())
	item, err := h.service.Create(r.Context(), claims, req)
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid input: %w", httpx.ErrInvalidInput))
		return
	}

	json.Write(w, http.StatusCreated, GenericResponse{
		Message: "Runner kit item created successfully",
		Data:    item,
	})
}

// GetByID handles GET /runner-kits/{id}
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.UUIDParam(w, r, "id")
	if !ok {
		return
	}

	item, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("runner kit item not found: %w", httpx.ErrNotFound))
		return
	}

	json.Write(w, http.StatusOK, GenericResponse{
		Message: "Runner kit item retrieved successfully",
		Data:    item,
	})
}

// ListByEvent handles GET /runner-kits/event/{event_id}
func (h *Handler) ListByEvent(w http.ResponseWriter, r *http.Request) {
	eventID, ok := httpx.UUIDParam(w, r, "event_id")
	if !ok {
		return
	}

	items, err := h.service.ListByEvent(r.Context(), eventID)
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("failed to list runner kit items: %w", err))
		return
	}

	json.Write(w, http.StatusOK, GenericResponse{
		Message: "Runner kit items retrieved successfully",
		Data:    items,
	})
}

// Update handles PUT /runner-kits/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.UUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req UpdateRunnerKitItemRequest
	if err := json.Read(r, &req); err != nil {
		httpx.WriteErrorCode(w, http.StatusBadRequest, "VALIDATION_FAILED", "invalid request body")
		return
	}

	claims, _ := kinemiddleware.ClaimsFrom(r.Context())
	item, err := h.service.Update(r.Context(), claims, id, req)
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
	id, ok := httpx.UUIDParam(w, r, "id")
	if !ok {
		return
	}

	claims, _ := kinemiddleware.ClaimsFrom(r.Context())
	if err := h.service.Delete(r.Context(), claims, id); err != nil {
		json.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	json.Write(w, http.StatusOK, map[string]string{
		"message": "Runner kit item deleted successfully",
	})
}
