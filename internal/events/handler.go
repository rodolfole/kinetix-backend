package events

import (
	"net/http"
	"strconv"

	"kinetix-api/internal/json"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// Handler handles HTTP requests for events
type Handler struct {
	service   *Service
	validator *validator.Validate
}

// NewHandler creates a new events handler
func NewHandler(service *Service, validator *validator.Validate) *Handler {
	return &Handler{
		service:   service,
		validator: validator,
	}
}

// Routes registers all event routes
func (h *Handler) Routes(r chi.Router) {
	r.Post("/", h.CreateEvent)
	r.Get("/", h.ListEvents)
	r.Get("/{id}", h.GetEvent)
	r.Get("/{id}/detail", h.GetEventDetail)
	r.Get("/slug/{slug}", h.GetEventBySlug)
	r.Put("/{id}", h.UpdateEvent)
	r.Patch("/{id}/status", h.UpdateEventStatus)
	r.Put("/{id}/participant-fields", h.UpdateParticipantFields)
	r.Get("/{id}/field-config", h.GetFieldConfig)
	r.Put("/{id}/field-config", h.UpdateFieldConfig)
	r.Delete("/{id}", h.DeleteEvent)

	// Awards
	r.Post("/awards", h.CreateAward)
	r.Get("/awards/{id}", h.GetAward)
	r.Get("/distance/{distance_id}/awards", h.ListAwards)
	r.Put("/awards/{id}", h.UpdateAward)
	r.Delete("/awards/{id}", h.DeleteAward)

	// Pricing stages
	r.Post("/pricing", h.CreatePricingStage)
	r.Get("/pricing/{id}", h.GetPricingStage)
	r.Get("/distance/{distance_id}/pricing", h.ListPricingStages)
	r.Get("/distance/{distance_id}/pricing/current", h.GetCurrentPricing)
	r.Put("/pricing/{id}", h.UpdatePricingStage)
	r.Delete("/pricing/{id}", h.DeletePricingStage)

	// Sponsors
	r.Post("/sponsors", h.CreateSponsor)
	r.Get("/sponsors/{id}", h.GetSponsor)
	r.Get("/event/{event_id}/sponsors", h.ListSponsors)
	r.Put("/sponsors/{id}", h.UpdateSponsor)
	r.Delete("/sponsors/{id}", h.DeleteSponsor)

	// Registration locations
	r.Post("/locations", h.CreateRegistrationLocation)
	r.Get("/locations/{id}", h.GetRegistrationLocation)
	r.Get("/event/{event_id}/locations", h.ListRegistrationLocations)
	r.Put("/locations/{id}", h.UpdateRegistrationLocation)
	r.Delete("/locations/{id}", h.DeleteRegistrationLocation)
}

// EVENT handlers

// CreateEvent handles POST /events
func (h *Handler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	var req CreateEventRequest
	if err := json.Read(r, &req); err != nil {
		json.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	evt, err := h.service.Create(r.Context(), req)
	if err != nil {
		// Use json.WriteError so the error string is properly escaped.
		// Previously we did `http.Error(w, `{"error":"`+err.Error()+`"}`, ...)`
		// which broke JSON parsing on the client when err.Error() contained
		// quotes (e.g. Postgres "column does not exist" messages with quotes).
		json.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	json.Write(w, http.StatusCreated, EventResponse{
		Message: "Event created successfully",
		Data:    evt,
	})
}

// GetEvent handles GET /events/{id}
func (h *Handler) GetEvent(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid event ID"}`, http.StatusBadRequest)
		return
	}

	evt, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"event not found"}`, http.StatusNotFound)
		return
	}

	json.Write(w, http.StatusOK, EventResponse{
		Message: "Event retrieved successfully",
		Data:    evt,
	})
}

// GetEventDetail handles GET /events/{id}/detail
func (h *Handler) GetEventDetail(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid event ID"}`, http.StatusBadRequest)
		return
	}

	detail, err := h.service.GetFullEventDetail(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"event not found"}`, http.StatusNotFound)
		return
	}

	json.Write(w, http.StatusOK, detail)
}

// GetEventBySlug handles GET /events/slug/{slug}
func (h *Handler) GetEventBySlug(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	evt, err := h.service.GetBySlug(r.Context(), slug)
	if err != nil {
		http.Error(w, `{"error":"event not found"}`, http.StatusNotFound)
		return
	}

	json.Write(w, http.StatusOK, EventResponse{
		Message: "Event retrieved successfully",
		Data:    evt,
	})
}

// ListEvents handles GET /events
func (h *Handler) ListEvents(w http.ResponseWriter, r *http.Request) {
	var status *string
	if s := r.URL.Query().Get("status"); s != "" {
		status = &s
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	if limit <= 0 {
		limit = 10
	}

	events, total, err := h.service.List(r.Context(), status, limit, offset)
	if err != nil {
		http.Error(w, `{"error":"failed to list events"}`, http.StatusInternalServerError)
		return
	}

	json.Write(w, http.StatusOK, EventsListResponse{
		Message: "Events retrieved successfully",
		Data:    events,
		Total:   total,
	})
}

// UpdateEvent handles PUT /events/{id}
func (h *Handler) UpdateEvent(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid event ID"}`, http.StatusBadRequest)
		return
	}

	var req UpdateEventRequest
	if err := json.Read(r, &req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	evt, err := h.service.Update(r.Context(), id, req)
	if err != nil {
		json.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	json.Write(w, http.StatusOK, EventResponse{
		Message: "Event updated successfully",
		Data:    evt,
	})
}

// UpdateEventStatus handles PATCH /events/{id}/status
func (h *Handler) UpdateEventStatus(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid event ID"}`, http.StatusBadRequest)
		return
	}

	var body struct {
		Status string `json:"status"`
	}
	if err := json.Read(r, &body); err != nil || body.Status == "" {
		http.Error(w, `{"error":"status is required"}`, http.StatusBadRequest)
		return
	}

	evt, err := h.service.UpdateStatus(r.Context(), id, body.Status)
	if err != nil {
		json.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	json.Write(w, http.StatusOK, EventResponse{
		Message: "Event status updated successfully",
		Data:    evt,
	})
}

// UpdateParticipantFields handles PUT /events/{id}/participant-fields
func (h *Handler) UpdateParticipantFields(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid event ID"}`, http.StatusBadRequest)
		return
	}

	var req UpdateParticipantFieldsRequest
	if err := json.Read(r, &req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if err := h.service.UpdateParticipantFields(r.Context(), id, req); err != nil {
		json.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	json.Write(w, http.StatusOK, map[string]string{
		"message": "Participant fields updated successfully",
	})
}

// GetFieldConfig handles GET /events/{id}/field-config
func (h *Handler) GetFieldConfig(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid event ID"}`, http.StatusBadRequest)
		return
	}

	configs, err := h.service.GetFieldConfig(r.Context(), id)
	if err != nil {
		json.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	json.Write(w, http.StatusOK, configs)
}

// UpdateFieldConfig handles PUT /events/{id}/field-config
func (h *Handler) UpdateFieldConfig(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid event ID"}`, http.StatusBadRequest)
		return
	}

	var req UpdateEventFieldConfigRequest
	if err := json.Read(r, &req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if err := h.service.UpdateFieldConfig(r.Context(), id, req); err != nil {
		json.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	json.Write(w, http.StatusOK, map[string]string{
		"message": "Field config updated successfully",
	})
}

// DeleteEvent handles DELETE /events/{id}
func (h *Handler) DeleteEvent(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid event ID"}`, http.StatusBadRequest)
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		json.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	json.Write(w, http.StatusOK, map[string]string{
		"message": "Event deleted successfully",
	})
}

// AWARD handlers

// CreateAward handles POST /events/awards
func (h *Handler) CreateAward(w http.ResponseWriter, r *http.Request) {
	var req CreateAwardRequest
	if err := json.Read(r, &req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	award, err := h.service.CreateAward(r.Context(), req)
	if err != nil {
		json.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	json.Write(w, http.StatusCreated, GenericResponse{
		Message: "Award created successfully",
		Data:    award,
	})
}

// GetAward handles GET /events/awards/{id}
func (h *Handler) GetAward(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid award ID"}`, http.StatusBadRequest)
		return
	}

	award, err := h.service.GetAward(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"award not found"}`, http.StatusNotFound)
		return
	}

	json.Write(w, http.StatusOK, GenericResponse{
		Message: "Award retrieved successfully",
		Data:    award,
	})
}

// ListAwards handles GET /events/distance/{distance_id}/awards
func (h *Handler) ListAwards(w http.ResponseWriter, r *http.Request) {
	distanceID, err := uuid.Parse(chi.URLParam(r, "distance_id"))
	if err != nil {
		http.Error(w, `{"error":"invalid distance ID"}`, http.StatusBadRequest)
		return
	}

	awards, err := h.service.ListAwardsByDistance(r.Context(), distanceID)
	if err != nil {
		http.Error(w, `{"error":"failed to list awards"}`, http.StatusInternalServerError)
		return
	}

	json.Write(w, http.StatusOK, GenericResponse{
		Message: "Awards retrieved successfully",
		Data:    awards,
	})
}

// UpdateAward handles PUT /events/awards/{id}
func (h *Handler) UpdateAward(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid award ID"}`, http.StatusBadRequest)
		return
	}

	var req UpdateAwardRequest
	if err := json.Read(r, &req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	award, err := h.service.UpdateAward(r.Context(), id, req)
	if err != nil {
		json.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	json.Write(w, http.StatusOK, GenericResponse{
		Message: "Award updated successfully",
		Data:    award,
	})
}

// DeleteAward handles DELETE /events/awards/{id}
func (h *Handler) DeleteAward(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid award ID"}`, http.StatusBadRequest)
		return
	}

	if err := h.service.DeleteAward(r.Context(), id); err != nil {
		json.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	json.Write(w, http.StatusOK, map[string]string{
		"message": "Award deleted successfully",
	})
}

// PRICING STAGE handlers

// CreatePricingStage handles POST /events/pricing
func (h *Handler) CreatePricingStage(w http.ResponseWriter, r *http.Request) {
	var req CreatePricingStageRequest
	if err := json.Read(r, &req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	stage, err := h.service.CreatePricingStage(r.Context(), req)
	if err != nil {
		json.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	json.Write(w, http.StatusCreated, GenericResponse{
		Message: "Pricing stage created successfully",
		Data:    stage,
	})
}

// GetPricingStage handles GET /events/pricing/{id}
func (h *Handler) GetPricingStage(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid pricing stage ID"}`, http.StatusBadRequest)
		return
	}

	stage, err := h.service.GetPricingStage(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"pricing stage not found"}`, http.StatusNotFound)
		return
	}

	json.Write(w, http.StatusOK, GenericResponse{
		Message: "Pricing stage retrieved successfully",
		Data:    stage,
	})
}

// ListPricingStages handles GET /events/distance/{distance_id}/pricing
func (h *Handler) ListPricingStages(w http.ResponseWriter, r *http.Request) {
	distanceID, err := uuid.Parse(chi.URLParam(r, "distance_id"))
	if err != nil {
		http.Error(w, `{"error":"invalid distance ID"}`, http.StatusBadRequest)
		return
	}

	stages, err := h.service.ListPricingStagesByDistance(r.Context(), distanceID)
	if err != nil {
		http.Error(w, `{"error":"failed to list pricing stages"}`, http.StatusInternalServerError)
		return
	}

	json.Write(w, http.StatusOK, GenericResponse{
		Message: "Pricing stages retrieved successfully",
		Data:    stages,
	})
}

// GetCurrentPricing handles GET /events/distance/{distance_id}/pricing/current
func (h *Handler) GetCurrentPricing(w http.ResponseWriter, r *http.Request) {
	distanceID, err := uuid.Parse(chi.URLParam(r, "distance_id"))
	if err != nil {
		http.Error(w, `{"error":"invalid distance ID"}`, http.StatusBadRequest)
		return
	}

	stages, err := h.service.GetCurrentPricing(r.Context(), distanceID)
	if err != nil {
		http.Error(w, `{"error":"failed to get current pricing"}`, http.StatusInternalServerError)
		return
	}

	json.Write(w, http.StatusOK, GenericResponse{
		Message: "Current pricing retrieved successfully",
		Data:    stages,
	})
}

// UpdatePricingStage handles PUT /events/pricing/{id}
func (h *Handler) UpdatePricingStage(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid pricing stage ID"}`, http.StatusBadRequest)
		return
	}

	var req UpdatePricingStageRequest
	if err := json.Read(r, &req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	stage, err := h.service.UpdatePricingStage(r.Context(), id, req)
	if err != nil {
		json.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	json.Write(w, http.StatusOK, GenericResponse{
		Message: "Pricing stage updated successfully",
		Data:    stage,
	})
}

// DeletePricingStage handles DELETE /events/pricing/{id}
func (h *Handler) DeletePricingStage(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid pricing stage ID"}`, http.StatusBadRequest)
		return
	}

	if err := h.service.DeletePricingStage(r.Context(), id); err != nil {
		json.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	json.Write(w, http.StatusOK, map[string]string{
		"message": "Pricing stage deleted successfully",
	})
}

// SPONSOR handlers

// CreateSponsor handles POST /events/sponsors
func (h *Handler) CreateSponsor(w http.ResponseWriter, r *http.Request) {
	var req CreateSponsorRequest
	if err := json.Read(r, &req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	sponsor, err := h.service.CreateSponsor(r.Context(), req)
	if err != nil {
		json.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	json.Write(w, http.StatusCreated, GenericResponse{
		Message: "Sponsor created successfully",
		Data:    sponsor,
	})
}

// GetSponsor handles GET /events/sponsors/{id}
func (h *Handler) GetSponsor(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid sponsor ID"}`, http.StatusBadRequest)
		return
	}

	sponsor, err := h.service.GetSponsor(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"sponsor not found"}`, http.StatusNotFound)
		return
	}

	json.Write(w, http.StatusOK, GenericResponse{
		Message: "Sponsor retrieved successfully",
		Data:    sponsor,
	})
}

// ListSponsors handles GET /events/event/{event_id}/sponsors
func (h *Handler) ListSponsors(w http.ResponseWriter, r *http.Request) {
	eventID, err := uuid.Parse(chi.URLParam(r, "event_id"))
	if err != nil {
		http.Error(w, `{"error":"invalid event ID"}`, http.StatusBadRequest)
		return
	}

	sponsors, err := h.service.ListSponsorsByEvent(r.Context(), eventID)
	if err != nil {
		http.Error(w, `{"error":"failed to list sponsors"}`, http.StatusInternalServerError)
		return
	}

	json.Write(w, http.StatusOK, GenericResponse{
		Message: "Sponsors retrieved successfully",
		Data:    sponsors,
	})
}

// UpdateSponsor handles PUT /events/sponsors/{id}
func (h *Handler) UpdateSponsor(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid sponsor ID"}`, http.StatusBadRequest)
		return
	}

	var req UpdateSponsorRequest
	if err := json.Read(r, &req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	sponsor, err := h.service.UpdateSponsor(r.Context(), id, req)
	if err != nil {
		json.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	json.Write(w, http.StatusOK, GenericResponse{
		Message: "Sponsor updated successfully",
		Data:    sponsor,
	})
}

// DeleteSponsor handles DELETE /events/sponsors/{id}
func (h *Handler) DeleteSponsor(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid sponsor ID"}`, http.StatusBadRequest)
		return
	}

	if err := h.service.DeleteSponsor(r.Context(), id); err != nil {
		json.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	json.Write(w, http.StatusOK, map[string]string{
		"message": "Sponsor deleted successfully",
	})
}

// REGISTRATION LOCATION handlers

// CreateRegistrationLocation handles POST /events/locations
func (h *Handler) CreateRegistrationLocation(w http.ResponseWriter, r *http.Request) {
	var req CreateRegistrationLocationRequest
	if err := json.Read(r, &req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	loc, err := h.service.CreateRegistrationLocation(r.Context(), req)
	if err != nil {
		json.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	json.Write(w, http.StatusCreated, GenericResponse{
		Message: "Registration location created successfully",
		Data:    loc,
	})
}

// GetRegistrationLocation handles GET /events/locations/{id}
func (h *Handler) GetRegistrationLocation(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid location ID"}`, http.StatusBadRequest)
		return
	}

	loc, err := h.service.GetRegistrationLocation(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"registration location not found"}`, http.StatusNotFound)
		return
	}

	json.Write(w, http.StatusOK, GenericResponse{
		Message: "Registration location retrieved successfully",
		Data:    loc,
	})
}

// ListRegistrationLocations handles GET /events/event/{event_id}/locations
func (h *Handler) ListRegistrationLocations(w http.ResponseWriter, r *http.Request) {
	eventID, err := uuid.Parse(chi.URLParam(r, "event_id"))
	if err != nil {
		http.Error(w, `{"error":"invalid event ID"}`, http.StatusBadRequest)
		return
	}

	locations, err := h.service.ListRegistrationLocationsByEvent(r.Context(), eventID)
	if err != nil {
		http.Error(w, `{"error":"failed to list registration locations"}`, http.StatusInternalServerError)
		return
	}

	json.Write(w, http.StatusOK, GenericResponse{
		Message: "Registration locations retrieved successfully",
		Data:    locations,
	})
}

// UpdateRegistrationLocation handles PUT /events/locations/{id}
func (h *Handler) UpdateRegistrationLocation(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid location ID"}`, http.StatusBadRequest)
		return
	}

	var req UpdateRegistrationLocationRequest
	if err := json.Read(r, &req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	loc, err := h.service.UpdateRegistrationLocation(r.Context(), id, req)
	if err != nil {
		json.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	json.Write(w, http.StatusOK, GenericResponse{
		Message: "Registration location updated successfully",
		Data:    loc,
	})
}

// DeleteRegistrationLocation handles DELETE /events/locations/{id}
func (h *Handler) DeleteRegistrationLocation(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid location ID"}`, http.StatusBadRequest)
		return
	}

	if err := h.service.DeleteRegistrationLocation(r.Context(), id); err != nil {
		json.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	json.Write(w, http.StatusOK, map[string]string{
		"message": "Registration location deleted successfully",
	})
}
