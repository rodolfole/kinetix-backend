package registrations

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handler handles HTTP requests for registrations
type Handler struct {
	service *Service
}

// NewHandler creates a new registrations handler
func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// Routes registers registration routes
func (h *Handler) Routes(r chi.Router) {
	r.Post("/", h.Create)
	r.Post("/with-participant", h.CreateWithParticipant)
	r.Get("/", h.ListByEvent)
	r.Get("/participant/{participant_id}", h.ListByParticipant)
	r.Get("/{id}", h.GetByID)
	r.Delete("/{id}", h.Delete)
}

// Create handles POST /registrations
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	registration, err := h.service.Create(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(RegistrationResponse{
		Message: "Registration created successfully",
		Data:    registration,
	})
}

// CreateWithParticipant handles POST /registrations/with-participant
func (h *Handler) CreateWithParticipant(w http.ResponseWriter, r *http.Request) {
	var req CreateWithParticipantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	registration, err := h.service.CreateWithParticipant(r.Context(), req.EventID, req.Participant, req.Registration, req.ParticipantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(RegistrationResponse{
		Message: "Registration created successfully",
		Data:    registration,
	})
}

// ListByEvent handles GET /registrations?event_id=xxx
func (h *Handler) ListByEvent(w http.ResponseWriter, r *http.Request) {
	eventIDStr := r.URL.Query().Get("event_id")
	if eventIDStr == "" {
		http.Error(w, "event_id is required", http.StatusBadRequest)
		return
	}

	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	registrations, err := h.service.ListByEvent(r.Context(), eventID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(RegistrationsListResponse{
		Message: "Registrations retrieved successfully",
		Data:    registrations,
	})
}

// ListByParticipant handles GET /registrations/participant/{participant_id}
func (h *Handler) ListByParticipant(w http.ResponseWriter, r *http.Request) {
	participantID, err := uuid.Parse(chi.URLParam(r, "participant_id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	registrations, err := h.service.ListByParticipant(r.Context(), participantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(RegistrationsListResponse{
		Message: "Registrations retrieved successfully",
		Data:    registrations,
	})
}

// GetByID handles GET /registrations/{id}
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	registration, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(RegistrationResponse{
		Message: "Registration retrieved successfully",
		Data:    registration,
	})
}

// Delete handles DELETE /registrations/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(GenericResponse{
		Message: "Registration deleted successfully",
	})
}

// GenericResponse for simple responses
type GenericResponse struct {
	Message string `json:"message"`
}
