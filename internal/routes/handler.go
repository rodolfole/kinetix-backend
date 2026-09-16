package routes

import (
	"net/http"

	"kinetix-api/internal/json"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// Handler handles HTTP requests for routes
type Handler struct {
	service   *Service
	validator *validator.Validate
}

// NewHandler creates a new routes handler
func NewHandler(service *Service, validator *validator.Validate) *Handler {
	return &Handler{
		service:   service,
		validator: validator,
	}
}

// Routes registers all route-related routes (protected, mounted under /events/routes)
func (h *Handler) Routes(r chi.Router) {
	r.Post("/upload", h.UploadGPX)
	r.Get("/{id}", h.GetByID)
	r.Delete("/{id}", h.Delete)
}

// ── Handlers ───────────────────────────────────────────────────────────────

// UploadGPX handles POST /routes/upload — multipart form with GPX file
func (h *Handler) UploadGPX(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form (max 32MB)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		json.WriteError(w, http.StatusBadRequest, "failed to parse multipart form")
		return
	}

	// Get event_id
	eventIDStr := r.FormValue("event_id")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		json.WriteError(w, http.StatusBadRequest, "invalid or missing event_id")
		return
	}

	// Get optional distance_id
	var distanceID *uuid.UUID
	if dIDStr := r.FormValue("distance_id"); dIDStr != "" {
		dID, err := uuid.Parse(dIDStr)
		if err != nil {
			json.WriteError(w, http.StatusBadRequest, "invalid distance_id")
			return
		}
		distanceID = &dID
	}

	// Get optional name
	name := r.FormValue("name")

	// Get the file
	file, _, err := r.FormFile("file")
	if err != nil {
		json.WriteError(w, http.StatusBadRequest, "missing or invalid 'file' field")
		return
	}
	defer file.Close()

	// Build request
	req := UploadRouteRequest{
		EventID:    eventID,
		DistanceID: distanceID,
		Name:       name,
	}

	route, err := h.service.UploadFromGPX(r.Context(), file, req)
	if err != nil {
		json.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	json.Write(w, http.StatusCreated, GenericResponse{
		Message: "Route uploaded successfully",
		Data:    route,
	})
}

// GetByID handles GET /routes/{id}
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		json.WriteError(w, http.StatusBadRequest, "invalid route ID")
		return
	}

	route, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		json.WriteError(w, http.StatusNotFound, "route not found")
		return
	}

	json.Write(w, http.StatusOK, GenericResponse{
		Message: "Route retrieved successfully",
		Data:    route,
	})
}

// ListByEvent handles GET /events/event/{event_id}/routes
func (h *Handler) ListByEvent(w http.ResponseWriter, r *http.Request) {
	eventID, err := uuid.Parse(chi.URLParam(r, "event_id"))
	if err != nil {
		json.WriteError(w, http.StatusBadRequest, "invalid event ID")
		return
	}

	routes, err := h.service.ListByEvent(r.Context(), eventID)
	if err != nil {
		json.WriteError(w, http.StatusInternalServerError, "failed to list routes")
		return
	}

	json.Write(w, http.StatusOK, GenericResponse{
		Message: "Routes retrieved successfully",
		Data:    routes,
	})
}

// ListByDistance handles GET /routes/distance/{distance_id}
func (h *Handler) ListByDistance(w http.ResponseWriter, r *http.Request) {
	distanceID, err := uuid.Parse(chi.URLParam(r, "distance_id"))
	if err != nil {
		json.WriteError(w, http.StatusBadRequest, "invalid distance ID")
		return
	}

	routes, err := h.service.ListByDistance(r.Context(), distanceID)
	if err != nil {
		json.WriteError(w, http.StatusInternalServerError, "failed to list routes by distance")
		return
	}

	json.Write(w, http.StatusOK, GenericResponse{
		Message: "Routes retrieved successfully",
		Data:    routes,
	})
}

// Delete handles DELETE /routes/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		json.WriteError(w, http.StatusBadRequest, "invalid route ID")
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		json.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	json.Write(w, http.StatusOK, map[string]string{
		"message": "Route deleted successfully",
	})
}
