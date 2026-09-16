package events

import (
	"context"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// Service handles business logic for events
type Service struct {
	repo      *Repository
	validator *validator.Validate
}

// NewService creates a new events service
func NewService(repo *Repository, validator *validator.Validate) *Service {
	return &Service{
		repo:      repo,
		validator: validator,
	}
}

// EVENT operations

// Create creates a new event
func (s *Service) Create(ctx context.Context, req CreateEventRequest) (Event, error) {
	if err := s.validator.Struct(req); err != nil {
		return Event{}, fmt.Errorf("validation error: %w", err)
	}

	if len(req.Itinerary) == 0 {
		return Event{}, fmt.Errorf("itinerary is required: at least one item must be provided")
	}

	evt, err := s.repo.Create(ctx, req)
	if err != nil {
		return Event{}, fmt.Errorf("failed to create event: %w", err)
	}

	return evt, nil
}

// GetByID retrieves an event by ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (Event, error) {
	if id == uuid.Nil {
		return Event{}, fmt.Errorf("invalid event ID")
	}

	evt, err := s.repo.GetEventByID(ctx, id)
	if err != nil {
		return Event{}, err
	}

	return evt, nil
}

// GetBySlug retrieves an event by slug
func (s *Service) GetBySlug(ctx context.Context, slug string) (Event, error) {
	if slug == "" {
		return Event{}, fmt.Errorf("slug is required")
	}

	evt, err := s.repo.GetEventBySlug(ctx, slug)
	if err != nil {
		return Event{}, err
	}

	return evt, nil
}

// GetFullEventDetail retrieves complete event with all related data
func (s *Service) GetFullEventDetail(ctx context.Context, id uuid.UUID) (EventDetailResponse, error) {
	if id == uuid.Nil {
		return EventDetailResponse{}, fmt.Errorf("invalid event ID")
	}

	evt, distances, awards, pricing, sponsors, locations, tags, participantFields, fieldConfigs, runnerKits, err := s.repo.GetFullEvent(ctx, id)
	if err != nil {
		return EventDetailResponse{}, err
	}

	return EventDetailResponse{
		Message:                 "Event retrieved successfully",
		Data:                    evt,
		Distances:               distances,
		Awards:                  awards,
		PricingStages:           pricing,
		Sponsors:                sponsors,
		RegistrationLocations:   locations,
		Tags:                    tags,
		ParticipantFieldsConfig: participantFields,
		FieldConfig:             fieldConfigs,
		RunnerKits:              runnerKits,
	}, nil
}

// List retrieves a paginated list of events
func (s *Service) List(ctx context.Context, status *string, limit, offset int) ([]Event, int64, error) {
	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	events, total, err := s.repo.ListEvents(ctx, status, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list events: %w", err)
	}

	return events, total, nil
}

// ListByOrganizer retrieves events by organizer
func (s *Service) ListByOrganizer(ctx context.Context, organizerID uuid.UUID) ([]Event, error) {
	if organizerID == uuid.Nil {
		return nil, fmt.Errorf("invalid organizer ID")
	}

	events, err := s.repo.ListEventsByOrganizer(ctx, organizerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list events: %w", err)
	}

	return events, nil
}

// Update updates an existing event
func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdateEventRequest) (Event, error) {
	if id == uuid.Nil {
		return Event{}, fmt.Errorf("invalid event ID")
	}

	if err := s.validator.Struct(req); err != nil {
		return Event{}, fmt.Errorf("validation error: %w", err)
	}

	if len(req.Itinerary) == 0 {
		return Event{}, fmt.Errorf("itinerary is required: at least one item must be provided")
	}

	_, err := s.repo.GetEventByID(ctx, id)
	if err != nil {
		return Event{}, fmt.Errorf("event not found: %w", err)
	}

	if req.Status == "" {
		req.Status = "draft"
	}

	updated, err := s.repo.Update(ctx, id, req)
	if err != nil {
		return Event{}, fmt.Errorf("failed to update event: %w", err)
	}

	return updated, nil
}

// UpdateStatus updates only the event status
func (s *Service) UpdateStatus(ctx context.Context, id uuid.UUID, status string) (Event, error) {
	if id == uuid.Nil {
		return Event{}, fmt.Errorf("invalid event ID")
	}

	validStatuses := map[string]bool{
		"draft":     true,
		"published": true,
		"cancelled": true,
		"completed": true,
	}

	if !validStatuses[status] {
		return Event{}, fmt.Errorf("invalid status: %s", status)
	}

	updated, err := s.repo.UpdateStatus(ctx, id, status)
	if err != nil {
		return Event{}, fmt.Errorf("failed to update status: %w", err)
	}

	return updated, nil
}

// Delete removes an event
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return fmt.Errorf("invalid event ID")
	}

	_, err := s.repo.GetEventByID(ctx, id)
	if err != nil {
		return fmt.Errorf("event not found: %w", err)
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete event: %w", err)
	}

	return nil
}

// AWARD operations

// CreateAward creates a new award
func (s *Service) CreateAward(ctx context.Context, req CreateAwardRequest) (Award, error) {
	if err := s.validator.Struct(req); err != nil {
		return Award{}, fmt.Errorf("validation error: %w", err)
	}

	_, err := s.repo.GetDistance(ctx, req.DistanceID)
	if err != nil {
		return Award{}, fmt.Errorf("distance not found: %w", err)
	}

	award, err := s.repo.CreateAward(ctx, req)
	if err != nil {
		return Award{}, fmt.Errorf("failed to create award: %w", err)
	}

	return award, nil
}

// GetAward retrieves an award by ID
func (s *Service) GetAward(ctx context.Context, id uuid.UUID) (Award, error) {
	if id == uuid.Nil {
		return Award{}, fmt.Errorf("invalid award ID")
	}

	award, err := s.repo.GetAward(ctx, id)
	if err != nil {
		return Award{}, err
	}

	return award, nil
}

// ListAwardsByDistance retrieves awards for a distance
func (s *Service) ListAwardsByDistance(ctx context.Context, distanceID uuid.UUID) ([]Award, error) {
	if distanceID == uuid.Nil {
		return nil, fmt.Errorf("invalid distance ID")
	}

	awards, err := s.repo.ListAwardsByDistance(ctx, distanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list awards: %w", err)
	}

	return awards, nil
}

// UpdateAward updates an award
func (s *Service) UpdateAward(ctx context.Context, id uuid.UUID, req UpdateAwardRequest) (Award, error) {
	if id == uuid.Nil {
		return Award{}, fmt.Errorf("invalid award ID")
	}

	if err := s.validator.Struct(req); err != nil {
		return Award{}, fmt.Errorf("validation error: %w", err)
	}

	updated, err := s.repo.UpdateAward(ctx, id, req)
	if err != nil {
		return Award{}, fmt.Errorf("failed to update award: %w", err)
	}

	return updated, nil
}

// DeleteAward removes an award
func (s *Service) DeleteAward(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return fmt.Errorf("invalid award ID")
	}

	if err := s.repo.DeleteAward(ctx, id); err != nil {
		return fmt.Errorf("failed to delete award: %w", err)
	}

	return nil
}

// PRICING STAGE operations

// CreatePricingStage creates a new pricing stage
func (s *Service) CreatePricingStage(ctx context.Context, req CreatePricingStageRequest) (PricingStage, error) {
	if err := s.validator.Struct(req); err != nil {
		return PricingStage{}, fmt.Errorf("validation error: %w", err)
	}

	_, err := s.repo.GetDistance(ctx, req.DistanceID)
	if err != nil {
		return PricingStage{}, fmt.Errorf("distance not found: %w", err)
	}

	if req.EndDate.Before(req.StartDate) {
		return PricingStage{}, fmt.Errorf("end date must be after start date")
	}

	stage, err := s.repo.CreatePricingStage(ctx, req)
	if err != nil {
		return PricingStage{}, fmt.Errorf("failed to create pricing stage: %w", err)
	}

	return stage, nil
}

// GetPricingStage retrieves a pricing stage by ID
func (s *Service) GetPricingStage(ctx context.Context, id uuid.UUID) (PricingStage, error) {
	if id == uuid.Nil {
		return PricingStage{}, fmt.Errorf("invalid pricing stage ID")
	}

	stage, err := s.repo.GetPricingStage(ctx, id)
	if err != nil {
		return PricingStage{}, err
	}

	return stage, nil
}

// ListPricingStagesByDistance retrieves pricing stages for a distance
func (s *Service) ListPricingStagesByDistance(ctx context.Context, distanceID uuid.UUID) ([]PricingStage, error) {
	if distanceID == uuid.Nil {
		return nil, fmt.Errorf("invalid distance ID")
	}

	stages, err := s.repo.ListPricingStagesByDistance(ctx, distanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list pricing stages: %w", err)
	}

	return stages, nil
}

// GetCurrentPricing retrieves current active pricing
func (s *Service) GetCurrentPricing(ctx context.Context, distanceID uuid.UUID) ([]PricingStage, error) {
	if distanceID == uuid.Nil {
		return nil, fmt.Errorf("invalid distance ID")
	}

	stages, err := s.repo.GetCurrentPricing(ctx, distanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current pricing: %w", err)
	}

	return stages, nil
}

// UpdatePricingStage updates a pricing stage
func (s *Service) UpdatePricingStage(ctx context.Context, id uuid.UUID, req UpdatePricingStageRequest) (PricingStage, error) {
	if id == uuid.Nil {
		return PricingStage{}, fmt.Errorf("invalid pricing stage ID")
	}

	if err := s.validator.Struct(req); err != nil {
		return PricingStage{}, fmt.Errorf("validation error: %w", err)
	}

	if req.EndDate.Before(req.StartDate) {
		return PricingStage{}, fmt.Errorf("end date must be after start date")
	}

	updated, err := s.repo.UpdatePricingStage(ctx, id, req)
	if err != nil {
		return PricingStage{}, fmt.Errorf("failed to update pricing stage: %w", err)
	}

	return updated, nil
}

// DeletePricingStage removes a pricing stage
func (s *Service) DeletePricingStage(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return fmt.Errorf("invalid pricing stage ID")
	}

	if err := s.repo.DeletePricingStage(ctx, id); err != nil {
		return fmt.Errorf("failed to delete pricing stage: %w", err)
	}

	return nil
}

// SPONSOR operations

// CreateSponsor creates a new sponsor
func (s *Service) CreateSponsor(ctx context.Context, req CreateSponsorRequest) (Sponsor, error) {
	if err := s.validator.Struct(req); err != nil {
		return Sponsor{}, fmt.Errorf("validation error: %w", err)
	}

	_, err := s.repo.GetEventByID(ctx, req.EventID)
	if err != nil {
		return Sponsor{}, fmt.Errorf("event not found: %w", err)
	}

	sponsor, err := s.repo.CreateSponsor(ctx, req)
	if err != nil {
		return Sponsor{}, fmt.Errorf("failed to create sponsor: %w", err)
	}

	return sponsor, nil
}

// GetSponsor retrieves a sponsor by ID
func (s *Service) GetSponsor(ctx context.Context, id uuid.UUID) (Sponsor, error) {
	if id == uuid.Nil {
		return Sponsor{}, fmt.Errorf("invalid sponsor ID")
	}

	sponsor, err := s.repo.GetSponsor(ctx, id)
	if err != nil {
		return Sponsor{}, err
	}

	return sponsor, nil
}

// ListSponsorsByEvent retrieves sponsors for an event
func (s *Service) ListSponsorsByEvent(ctx context.Context, eventID uuid.UUID) ([]Sponsor, error) {
	if eventID == uuid.Nil {
		return nil, fmt.Errorf("invalid event ID")
	}

	sponsors, err := s.repo.ListSponsorsByEvent(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to list sponsors: %w", err)
	}

	return sponsors, nil
}

// UpdateSponsor updates a sponsor
func (s *Service) UpdateSponsor(ctx context.Context, id uuid.UUID, req UpdateSponsorRequest) (Sponsor, error) {
	if id == uuid.Nil {
		return Sponsor{}, fmt.Errorf("invalid sponsor ID")
	}

	if err := s.validator.Struct(req); err != nil {
		return Sponsor{}, fmt.Errorf("validation error: %w", err)
	}

	updated, err := s.repo.UpdateSponsor(ctx, id, req)
	if err != nil {
		return Sponsor{}, fmt.Errorf("failed to update sponsor: %w", err)
	}

	return updated, nil
}

// DeleteSponsor removes a sponsor
func (s *Service) DeleteSponsor(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return fmt.Errorf("invalid sponsor ID")
	}

	if err := s.repo.DeleteSponsor(ctx, id); err != nil {
		return fmt.Errorf("failed to delete sponsor: %w", err)
	}

	return nil
}

// REGISTRATION LOCATION operations

// CreateRegistrationLocation creates a new registration location
func (s *Service) CreateRegistrationLocation(ctx context.Context, req CreateRegistrationLocationRequest) (RegistrationLocation, error) {
	if err := s.validator.Struct(req); err != nil {
		return RegistrationLocation{}, fmt.Errorf("validation error: %w", err)
	}

	_, err := s.repo.GetEventByID(ctx, req.EventID)
	if err != nil {
		return RegistrationLocation{}, fmt.Errorf("event not found: %w", err)
	}

	loc, err := s.repo.CreateRegistrationLocation(ctx, req)
	if err != nil {
		return RegistrationLocation{}, fmt.Errorf("failed to create registration location: %w", err)
	}

	return loc, nil
}

// GetRegistrationLocation retrieves a registration location by ID
func (s *Service) GetRegistrationLocation(ctx context.Context, id uuid.UUID) (RegistrationLocation, error) {
	if id == uuid.Nil {
		return RegistrationLocation{}, fmt.Errorf("invalid registration location ID")
	}

	loc, err := s.repo.GetRegistrationLocation(ctx, id)
	if err != nil {
		return RegistrationLocation{}, err
	}

	return loc, nil
}

// ListRegistrationLocationsByEvent retrieves registration locations for an event
func (s *Service) ListRegistrationLocationsByEvent(ctx context.Context, eventID uuid.UUID) ([]RegistrationLocation, error) {
	if eventID == uuid.Nil {
		return nil, fmt.Errorf("invalid event ID")
	}

	locations, err := s.repo.ListRegistrationLocationsByEvent(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to list registration locations: %w", err)
	}

	return locations, nil
}

// UpdateRegistrationLocation updates a registration location
func (s *Service) UpdateRegistrationLocation(ctx context.Context, id uuid.UUID, req UpdateRegistrationLocationRequest) (RegistrationLocation, error) {
	if id == uuid.Nil {
		return RegistrationLocation{}, fmt.Errorf("invalid registration location ID")
	}

	if err := s.validator.Struct(req); err != nil {
		return RegistrationLocation{}, fmt.Errorf("validation error: %w", err)
	}

	updated, err := s.repo.UpdateRegistrationLocation(ctx, id, req)
	if err != nil {
		return RegistrationLocation{}, fmt.Errorf("failed to update registration location: %w", err)
	}

	return updated, nil
}

// DeleteRegistrationLocation removes a registration location
func (s *Service) DeleteRegistrationLocation(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return fmt.Errorf("invalid registration location ID")
	}

	if err := s.repo.DeleteRegistrationLocation(ctx, id); err != nil {
		return fmt.Errorf("failed to delete registration location: %w", err)
	}

	return nil
}

// UpdateParticipantFields updates participant field configuration for an event
func (s *Service) UpdateParticipantFields(ctx context.Context, eventID uuid.UUID, req UpdateParticipantFieldsRequest) error {
	if eventID == uuid.Nil {
		return fmt.Errorf("invalid event ID")
	}

	// Verify event exists
	_, err := s.repo.GetEventByID(ctx, eventID)
	if err != nil {
		return fmt.Errorf("event not found: %w", err)
	}

	// Validate field names
	validFields := map[string]bool{
		"country":            true,
		"city":               true,
		"state":              true,
		"zip_code":           true,
		"medical_info":       true,
		"waiver_signed_at":   true,
		"emergency_contact":  true,
	}

	for _, f := range req.Fields {
		if !validFields[f.FieldName] {
			return fmt.Errorf("invalid field name: %s", f.FieldName)
		}
	}

	return s.repo.UpsertEventParticipantFields(ctx, eventID, req.Fields)
}

// GetFieldConfig retrieves the field configuration for an event
func (s *Service) GetFieldConfig(ctx context.Context, eventID uuid.UUID) ([]EventFieldConfig, error) {
	if eventID == uuid.Nil {
		return nil, fmt.Errorf("invalid event ID")
	}

	return s.repo.ListEventFieldConfigByEvent(ctx, eventID)
}

// UpdateFieldConfig updates the field configuration for an event
func (s *Service) UpdateFieldConfig(ctx context.Context, eventID uuid.UUID, req UpdateEventFieldConfigRequest) error {
	if eventID == uuid.Nil {
		return fmt.Errorf("invalid event ID")
	}

	// Verify event exists
	_, err := s.repo.GetEventByID(ctx, eventID)
	if err != nil {
		return fmt.Errorf("event not found: %w", err)
	}

	// Validate section names
	validSections := map[string]bool{
		"categories":    true,
		"sponsors":      true,
		"registrations": true,
		"judges":        true,
		"rules":         true,
		"risks":         true,
		"transit":       true,
	}

	for _, f := range req.Fields {
		if !validSections[f.SectionName] {
			return fmt.Errorf("invalid section name: %s", f.SectionName)
		}
	}

	return s.repo.UpsertEventFieldConfigs(ctx, eventID, req.Fields)
}
