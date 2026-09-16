package organizers

import (
	"net/http"
	"strconv"

	"kinetix-api/internal/json"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/go-playground/validator/v10"
)

// Handler handles HTTP requests for organizers
type Handler struct {
	service   *Service
	validator *validator.Validate
}

// NewHandler creates a new organizer handler
func NewHandler(service *Service, validator *validator.Validate) *Handler {
	return &Handler{
		service:   service,
		validator: validator,
	}
}

// Routes registers all organizer routes
func (h *Handler) Routes(r chi.Router) {
	r.Post("/", h.CreateOrganizer)
	r.Get("/", h.ListOrganizers)
	r.Get("/{id}", h.GetOrganizer)
	r.Put("/{id}", h.UpdateOrganizer)
	r.Delete("/{id}", h.DeleteOrganizer)
}

// CreateOrganizer handles POST /organizers
func (h *Handler) CreateOrganizer(w http.ResponseWriter, r *http.Request) {
	var req CreateOrganizerRequest
	if err := json.Read(r, &req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	org, err := h.service.Create(r.Context(), req)
	if err != nil {
		json.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	json.Write(w, http.StatusCreated, OrganizerResponse{
		Message: "Organizer created successfully",
		Data:    org,
	})
}

// GetOrganizer handles GET /organizers/{id}
func (h *Handler) GetOrganizer(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid organizer ID"}`, http.StatusBadRequest)
		return
	}

	org, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"organizer not found"}`, http.StatusNotFound)
		return
	}

	json.Write(w, http.StatusOK, OrganizerResponse{
		Message: "Organizer retrieved successfully",
		Data:    org,
	})
}

// ListOrganizers handles GET /organizers
func (h *Handler) ListOrganizers(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	if limit <= 0 {
		limit = 10
	}

	organizers, total, err := h.service.List(r.Context(), limit, offset)
	if err != nil {
		http.Error(w, `{"error":"failed to list organizers"}`, http.StatusInternalServerError)
		return
	}

	json.Write(w, http.StatusOK, OrganizersListResponse{
		Message: "Organizers retrieved successfully",
		Data:    organizers,
		Total:   total,
	})
}

// UpdateOrganizer handles PUT /organizers/{id}
func (h *Handler) UpdateOrganizer(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid organizer ID"}`, http.StatusBadRequest)
		return
	}

	var req UpdateOrganizerRequest
	if err := json.Read(r, &req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	org, err := h.service.Update(r.Context(), id, req)
	if err != nil {
		json.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	json.Write(w, http.StatusOK, OrganizerResponse{
		Message: "Organizer updated successfully",
		Data:    org,
	})
}

// DeleteOrganizer handles DELETE /organizers/{id}
func (h *Handler) DeleteOrganizer(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid organizer ID"}`, http.StatusBadRequest)
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		json.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	json.Write(w, http.StatusOK, map[string]string{
		"message": "Organizer deleted successfully",
	})
}
