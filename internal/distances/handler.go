package distances

import (
	"net/http"

	"kinetix-api/internal/json"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// Handler handles HTTP requests for distances
type Handler struct {
	service   *Service
	validator *validator.Validate
}

// NewHandler creates a new distances handler
func NewHandler(service *Service, validator *validator.Validate) *Handler {
	return &Handler{
		service:   service,
		validator: validator,
	}
}

// Routes registers all distance routes.
// Mounted under /api/v1 (public) and /api/v1/events (protected).
func (h *Handler) Routes(r chi.Router) {
	// Public: distance types catalog
	r.Get("/types", h.ListTypes)

	// Protected: list by event
	r.Get("/event/{event_id}", h.ListByEvent)

	// Protected: CRUD
	r.Post("/", h.Create)
	r.Get("/{id}", h.GetByID)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
}

// ── Handlers ───────────────────────────────────────────────────────────────

// ListTypes handles GET /distances/types
func (h *Handler) ListTypes(w http.ResponseWriter, r *http.Request) {
	types, err := h.service.ListTypes(r.Context())
	if err != nil {
		json.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	data := make([]DistanceTypeResponse, len(types))
	for i, dt := range types {
		data[i] = toDistanceTypeResponse(dt)
	}

	json.Write(w, http.StatusOK, GenericResponse{
		Message: "Distance types retrieved successfully",
		Data:    data,
	})
}

// Create handles POST /distances
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateDistanceRequest
	if err := json.Read(r, &req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	dist, err := h.service.Create(r.Context(), req)
	if err != nil {
		json.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	json.Write(w, http.StatusCreated, GenericResponse{
		Message: "Distance created successfully",
		Data:    toDistanceResponse(dist, nil),
	})
}

// GetByID handles GET /distances/{id}
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid distance ID"}`, http.StatusBadRequest)
		return
	}

	dist, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"distance not found"}`, http.StatusNotFound)
		return
	}

	json.Write(w, http.StatusOK, GenericResponse{
		Message: "Distance retrieved successfully",
		Data:    toDistanceResponse(dist, nil),
	})
}

// ListByEvent handles GET /distances/event/{event_id}
func (h *Handler) ListByEvent(w http.ResponseWriter, r *http.Request) {
	eventID, err := uuid.Parse(chi.URLParam(r, "event_id"))
	if err != nil {
		http.Error(w, `{"error":"invalid event ID"}`, http.StatusBadRequest)
		return
	}

	distances, err := h.service.ListByEventWithTypes(r.Context(), eventID)
	if err != nil {
		http.Error(w, `{"error":"failed to list distances"}`, http.StatusInternalServerError)
		return
	}

	json.Write(w, http.StatusOK, GenericResponse{
		Message: "Distances retrieved successfully",
		Data:    distances,
	})
}

// Update handles PUT /distances/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid distance ID"}`, http.StatusBadRequest)
		return
	}

	var req UpdateDistanceRequest
	if err := json.Read(r, &req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	dist, err := h.service.Update(r.Context(), id, req)
	if err != nil {
		json.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	json.Write(w, http.StatusOK, GenericResponse{
		Message: "Distance updated successfully",
		Data:    toDistanceResponse(dist, nil),
	})
}

// Delete handles DELETE /distances/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid distance ID"}`, http.StatusBadRequest)
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		json.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	json.Write(w, http.StatusOK, map[string]string{
		"message": "Distance deleted successfully",
	})
}
