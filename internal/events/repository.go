package events

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	repo "kinetix-api/internal/adapters/postgresql/sqlc"
	"kinetix-api/internal/database/convert"
	"kinetix-api/internal/store"
)

// Repository handles database operations for events
type Repository struct {
	store *store.Store
}

// NewRepository creates a new events repository
func NewRepository(s *store.Store) *Repository {
	return &Repository{store: s}
}

// Alias methods for compatibility with existing service
func (r *Repository) GetEventByID(ctx context.Context, id uuid.UUID) (Event, error) {
	return r.GetByID(ctx, id)
}

func (r *Repository) GetEventBySlug(ctx context.Context, slug string) (Event, error) {
	return r.GetBySlug(ctx, slug)
}

func (r *Repository) ListEvents(ctx context.Context, status *string, limit, offset int) ([]Event, int64, error) {
	return r.List(ctx, status, limit, offset)
}

func (r *Repository) ListEventsByOrganizer(ctx context.Context, organizerID uuid.UUID) ([]Event, error) {
	return r.ListByOrganizer(ctx, organizerID)
}

func (r *Repository) GetFullEvent(ctx context.Context, id uuid.UUID) (Event, []Distance, []Award, []PricingStage, []Sponsor, []RegistrationLocation, []string, []ParticipantFieldConfig, []EventFieldConfig, []RunnerKitItem, error) {
	return r.GetFullEventDetail(ctx, id)
}

// Create inserts a new event into the database
func (r *Repository) Create(ctx context.Context, req CreateEventRequest) (Event, error) {
	itineraryJSON, err := json.Marshal(req.Itinerary)
	if err != nil {
		return Event{}, fmt.Errorf("failed to marshal itinerary: %w", err)
	}

	evt, err := r.store.Queries.CreateEvent(ctx, repo.CreateEventParams{
		OrganizerID:  req.OrganizerID,
		Name:         req.Name,
		Slug:         req.Slug,
		Description:  convert.ToPgText(req.Description),
		EventDate:    convert.ToPgTime(req.EventDate),
		Deadline:     req.Deadline,
		State:        req.State,
		Municipality: req.Municipality,
		Address:      req.Address,
		LogoUrl:      req.LogoURL,
		Itinerary:    convert.ToPgJSONBFromBytes(itineraryJSON),
		Judges:       convert.ToPgText(req.Judges),
		Rules:        convert.ToPgText(req.Rules),
		Risks:        convert.ToPgText(req.Risks),
		Transit:      convert.ToPgText(req.Transit),
		Status:       convert.ToPgText("draft"),
	})
	if err != nil {
		return Event{}, fmt.Errorf("failed to create event: %w", err)
	}
	return toEvent(evt), nil
}

// GetByID retrieves an event by ID
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (Event, error) {
	evt, err := r.store.Queries.GetEvent(ctx, id)
	if err != nil {
		return Event{}, fmt.Errorf("event not found: %w", err)
	}
	return toEvent(evt), nil
}

// GetBySlug retrieves an event by slug
func (r *Repository) GetBySlug(ctx context.Context, slug string) (Event, error) {
	evt, err := r.store.Queries.GetEventBySlug(ctx, slug)
	if err != nil {
		return Event{}, fmt.Errorf("event not found: %w", err)
	}
	return toEvent(evt), nil
}

// List retrieves a paginated list of events
func (r *Repository) List(ctx context.Context, status *string, limit, offset int) ([]Event, int64, error) {
	var statusPg pgtype.Text
	if status != nil {
		statusPg = convert.ToPgText(*status)
	}

	events, err := r.store.Queries.ListEvents(ctx, repo.ListEventsParams{
		Status: statusPg,
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list events: %w", err)
	}

	result := make([]Event, len(events))
	for i, e := range events {
		result[i] = toEvent(e)
	}

	return result, int64(len(result)), nil
}

// ListByOrganizer retrieves events by organizer ID
func (r *Repository) ListByOrganizer(ctx context.Context, organizerID uuid.UUID) ([]Event, error) {
	events, err := r.store.Queries.ListEventsByOrganizer(ctx, organizerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list events: %w", err)
	}

	result := make([]Event, len(events))
	for i, e := range events {
		result[i] = toEvent(e)
	}
	return result, nil
}

// GetFullEventDetail retrieves event with all related data
func (r *Repository) GetFullEventDetail(ctx context.Context, eventID uuid.UUID) (Event, []Distance, []Award, []PricingStage, []Sponsor, []RegistrationLocation, []string, []ParticipantFieldConfig, []EventFieldConfig, []RunnerKitItem, error) {
	evt, err := r.store.Queries.GetEvent(ctx, eventID)
	if err != nil {
		return Event{}, nil, nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("event not found: %w", err)
	}

	distances, err := r.store.Queries.ListDistancesByEvent(ctx, eventID)
	if err != nil {
		return Event{}, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
	}

	distResult := make([]Distance, len(distances))
	for i, d := range distances {
		distResult[i] = toDistance(d)
	}

	var awards []Award
	var pricing []PricingStage

	for _, dist := range distances {
		distAwards, err := r.store.Queries.ListAwardsByDistance(ctx, dist.ID)
		if err != nil {
			return Event{}, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
		}
		for _, a := range distAwards {
			awards = append(awards, toAward(a))
		}

		distPricing, err := r.store.Queries.ListPricingStagesByDistance(ctx, dist.ID)
		if err != nil {
			return Event{}, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
		}
		for _, p := range distPricing {
			pricing = append(pricing, toPricingStage(p))
		}
	}

	sponsors, err := r.store.Queries.ListSponsorsByEvent(ctx, eventID)
	if err != nil {
		return Event{}, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
	}
	sponsorResult := make([]Sponsor, len(sponsors))
	for i, s := range sponsors {
		sponsorResult[i] = toSponsor(s)
	}

	locations, err := r.store.Queries.ListRegistrationLocationsByEvent(ctx, eventID)
	if err != nil {
		return Event{}, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
	}
	locationResult := make([]RegistrationLocation, len(locations))
	for i, l := range locations {
		locationResult[i] = toRegistrationLocation(l)
	}

	// Get tags
	tagRows, err := r.store.Queries.ListEventTagsByEvent(ctx, eventID)
	if err != nil {
		return Event{}, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
	}
	tags := make([]string, len(tagRows))
	for i, t := range tagRows {
		tags[i] = t.Name
	}

	// Get participant fields config
	participantFields, err := r.store.Queries.ListEventParticipantFieldsByEvent(ctx, eventID)
	if err != nil {
		return Event{}, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
	}
	fieldConfigResult := make([]ParticipantFieldConfig, len(participantFields))
	for i, f := range participantFields {
		fieldConfigResult[i] = ParticipantFieldConfig{
			FieldName:  f.FieldName,
			IsRequired: f.IsRequired,
			IsEnabled:  f.IsEnabled,
		}
	}

	// Get event field config (toggleable sections)
	eventFieldConfigs, err := r.store.Queries.ListEventFieldConfigByEvent(ctx, eventID)
	if err != nil {
		return Event{}, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
	}
	eventFieldConfigResult := make([]EventFieldConfig, len(eventFieldConfigs))
	for i, f := range eventFieldConfigs {
		eventFieldConfigResult[i] = toEventFieldConfig(f)
	}

	// Get runner kit items
	kitRows, err := r.store.Queries.ListRunnerKitItemsByEvent(ctx, eventID)
	if err != nil {
		return Event{}, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
	}
	kitResult := make([]RunnerKitItem, len(kitRows))
	for i, k := range kitRows {
		kitResult[i] = toRunnerKitItem(k)
	}

	return toEvent(evt), distResult, awards, pricing, sponsorResult, locationResult, tags, fieldConfigResult, eventFieldConfigResult, kitResult, nil
}

// Update updates an existing event
func (r *Repository) Update(ctx context.Context, id uuid.UUID, req UpdateEventRequest) (Event, error) {
	itineraryJSON, err := json.Marshal(req.Itinerary)
	if err != nil {
		return Event{}, fmt.Errorf("failed to marshal itinerary: %w", err)
	}

	evt, err := r.store.Queries.UpdateEvent(ctx, repo.UpdateEventParams{
		ID:           id,
		Name:         req.Name,
		Slug:         req.Slug,
		Description:  convert.ToPgText(req.Description),
		EventDate:    convert.ToPgTime(req.EventDate),
		Deadline:     req.Deadline,
		State:        req.State,
		Municipality: req.Municipality,
		Address:      req.Address,
		LogoUrl:      req.LogoURL,
		Itinerary:    convert.ToPgJSONBFromBytes(itineraryJSON),
		Judges:       convert.ToPgText(req.Judges),
		Rules:        convert.ToPgText(req.Rules),
		Risks:        convert.ToPgText(req.Risks),
		Transit:      convert.ToPgText(req.Transit),
		Status:       convert.ToPgText(req.Status),
	})
	if err != nil {
		return Event{}, fmt.Errorf("failed to update event: %w", err)
	}
	return toEvent(evt), nil
}

// UpdateStatus updates only the event status
func (r *Repository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) (Event, error) {
	evt, err := r.store.Queries.UpdateEventStatus(ctx, repo.UpdateEventStatusParams{
		ID:     id,
		Status: convert.ToPgText(status),
	})
	if err != nil {
		return Event{}, fmt.Errorf("failed to update event status: %w", err)
	}
	return toEvent(evt), nil
}

// Delete removes an event from the database
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	err := r.store.Queries.DeleteEvent(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete event: %w", err)
	}
	return nil
}

// GetDistance retrieves a distance by ID (used by awards/pricing validation)
func (r *Repository) GetDistance(ctx context.Context, id uuid.UUID) (Distance, error) {
	dist, err := r.store.Queries.GetDistance(ctx, id)
	if err != nil {
		return Distance{}, fmt.Errorf("distance not found: %w", err)
	}
	return toDistance(dist), nil
}

// AWARD operations

// CreateAward inserts a new award
func (r *Repository) CreateAward(ctx context.Context, req CreateAwardRequest) (Award, error) {
	place := int32(req.Place)
	if place == 0 {
		// Default = 1er lugar si el cliente no envía el campo
		// (compatibilidad con payloads viejos).
		place = 1
	}
	award, err := r.store.Queries.CreateAward(ctx, repo.CreateAwardParams{
		DistanceID:    req.DistanceID,
		AgeCategoryID: req.AgeCategoryID,
		Gender:        req.Gender,
		Place:         place,
		PrizeAmount:   convert.ToPgNumeric(req.PrizeAmount),
	})
	if err != nil {
		return Award{}, fmt.Errorf("failed to create award: %w", err)
	}
	return toAward(award), nil
}

// GetAward retrieves an award by ID
func (r *Repository) GetAward(ctx context.Context, id uuid.UUID) (Award, error) {
	award, err := r.store.Queries.GetAward(ctx, id)
	if err != nil {
		return Award{}, fmt.Errorf("award not found: %w", err)
	}
	return toAward(award), nil
}

// ListAwardsByDistance retrieves all awards for a distance
func (r *Repository) ListAwardsByDistance(ctx context.Context, distanceID uuid.UUID) ([]Award, error) {
	awards, err := r.store.Queries.ListAwardsByDistance(ctx, distanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list awards: %w", err)
	}

	result := make([]Award, len(awards))
	for i, a := range awards {
		result[i] = toAward(a)
	}
	return result, nil
}

// UpdateAward updates an award
func (r *Repository) UpdateAward(ctx context.Context, id uuid.UUID, req UpdateAwardRequest) (Award, error) {
	place := int32(req.Place)
	if place == 0 {
		place = 1
	}
	award, err := r.store.Queries.UpdateAward(ctx, repo.UpdateAwardParams{
		ID:            id,
		AgeCategoryID: req.AgeCategoryID,
		Gender:        req.Gender,
		Place:         place,
		PrizeAmount:   convert.ToPgNumeric(req.PrizeAmount),
	})
	if err != nil {
		return Award{}, fmt.Errorf("failed to update award: %w", err)
	}
	return toAward(award), nil
}

// DeleteAward removes an award
func (r *Repository) DeleteAward(ctx context.Context, id uuid.UUID) error {
	err := r.store.Queries.DeleteAward(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete award: %w", err)
	}
	return nil
}

// PRICING STAGE operations

// CreatePricingStage inserts a new pricing stage
func (r *Repository) CreatePricingStage(ctx context.Context, req CreatePricingStageRequest) (PricingStage, error) {
	stage, err := r.store.Queries.CreatePricingStage(ctx, repo.CreatePricingStageParams{
		DistanceID: req.DistanceID,
		Name:       convert.ToPgText(req.Name),
		Price:      convert.ToPgNumeric(req.Price),
		StartDate:  req.StartDate,
		EndDate:    req.EndDate,
	})
	if err != nil {
		return PricingStage{}, fmt.Errorf("failed to create pricing stage: %w", err)
	}
	return toPricingStage(stage), nil
}

// GetPricingStage retrieves a pricing stage by ID
func (r *Repository) GetPricingStage(ctx context.Context, id uuid.UUID) (PricingStage, error) {
	stage, err := r.store.Queries.GetPricingStage(ctx, id)
	if err != nil {
		return PricingStage{}, fmt.Errorf("pricing stage not found: %w", err)
	}
	return toPricingStage(stage), nil
}

// ListPricingStagesByDistance retrieves all pricing stages for a distance
func (r *Repository) ListPricingStagesByDistance(ctx context.Context, distanceID uuid.UUID) ([]PricingStage, error) {
	stages, err := r.store.Queries.ListPricingStagesByDistance(ctx, distanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list pricing stages: %w", err)
	}

	result := make([]PricingStage, len(stages))
	for i, s := range stages {
		result[i] = toPricingStage(s)
	}
	return result, nil
}

// GetCurrentPricing retrieves current active pricing stages
func (r *Repository) GetCurrentPricing(ctx context.Context, distanceID uuid.UUID) ([]PricingStage, error) {
	stages, err := r.store.Queries.GetCurrentPricing(ctx, distanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current pricing: %w", err)
	}

	result := make([]PricingStage, len(stages))
	for i, s := range stages {
		result[i] = toPricingStage(s)
	}
	return result, nil
}

// UpdatePricingStage updates a pricing stage
func (r *Repository) UpdatePricingStage(ctx context.Context, id uuid.UUID, req UpdatePricingStageRequest) (PricingStage, error) {
	stage, err := r.store.Queries.UpdatePricingStage(ctx, repo.UpdatePricingStageParams{
		ID:        id,
		Name:      convert.ToPgText(req.Name),
		Price:     convert.ToPgNumeric(req.Price),
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
	})
	if err != nil {
		return PricingStage{}, fmt.Errorf("failed to update pricing stage: %w", err)
	}
	return toPricingStage(stage), nil
}

// DeletePricingStage removes a pricing stage
func (r *Repository) DeletePricingStage(ctx context.Context, id uuid.UUID) error {
	err := r.store.Queries.DeletePricingStage(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete pricing stage: %w", err)
	}
	return nil
}

// SPONSOR operations

// CreateSponsor inserts a new sponsor
func (r *Repository) CreateSponsor(ctx context.Context, req CreateSponsorRequest) (Sponsor, error) {
	sponsor, err := r.store.Queries.CreateSponsor(ctx, repo.CreateSponsorParams{
		EventID:    req.EventID,
		Name:       req.Name,
		Tier:       req.Tier,
		LogoUrl:    convert.ToPgText(req.LogoURL),
		WebsiteUrl: convert.ToPgText(req.WebsiteURL),
	})
	if err != nil {
		return Sponsor{}, fmt.Errorf("failed to create sponsor: %w", err)
	}
	return toSponsor(sponsor), nil
}

// GetSponsor retrieves a sponsor by ID
func (r *Repository) GetSponsor(ctx context.Context, id uuid.UUID) (Sponsor, error) {
	sponsor, err := r.store.Queries.GetSponsor(ctx, id)
	if err != nil {
		return Sponsor{}, fmt.Errorf("sponsor not found: %w", err)
	}
	return toSponsor(sponsor), nil
}

// ListSponsorsByEvent retrieves all sponsors for an event
func (r *Repository) ListSponsorsByEvent(ctx context.Context, eventID uuid.UUID) ([]Sponsor, error) {
	sponsors, err := r.store.Queries.ListSponsorsByEvent(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to list sponsors: %w", err)
	}

	result := make([]Sponsor, len(sponsors))
	for i, s := range sponsors {
		result[i] = toSponsor(s)
	}
	return result, nil
}

// UpdateSponsor updates a sponsor
func (r *Repository) UpdateSponsor(ctx context.Context, id uuid.UUID, req UpdateSponsorRequest) (Sponsor, error) {
	sponsor, err := r.store.Queries.UpdateSponsor(ctx, repo.UpdateSponsorParams{
		ID:         id,
		Name:       req.Name,
		Tier:       req.Tier,
		LogoUrl:    convert.ToPgText(req.LogoURL),
		WebsiteUrl: convert.ToPgText(req.WebsiteURL),
	})
	if err != nil {
		return Sponsor{}, fmt.Errorf("failed to update sponsor: %w", err)
	}
	return toSponsor(sponsor), nil
}

// DeleteSponsor removes a sponsor
func (r *Repository) DeleteSponsor(ctx context.Context, id uuid.UUID) error {
	err := r.store.Queries.DeleteSponsor(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete sponsor: %w", err)
	}
	return nil
}

// REGISTRATION LOCATION operations

// CreateRegistrationLocation inserts a new registration location
func (r *Repository) CreateRegistrationLocation(ctx context.Context, req CreateRegistrationLocationRequest) (RegistrationLocation, error) {
	loc, err := r.store.Queries.CreateRegistrationLocation(ctx, repo.CreateRegistrationLocationParams{
		EventID:      req.EventID,
		LocationName: req.LocationName,
		Address:      req.Address,
		Schedule:     convert.ToPgText(req.Schedule),
	})
	if err != nil {
		return RegistrationLocation{}, fmt.Errorf("failed to create registration location: %w", err)
	}
	return toRegistrationLocation(loc), nil
}

// GetRegistrationLocation retrieves a registration location by ID
func (r *Repository) GetRegistrationLocation(ctx context.Context, id uuid.UUID) (RegistrationLocation, error) {
	loc, err := r.store.Queries.GetRegistrationLocation(ctx, id)
	if err != nil {
		return RegistrationLocation{}, fmt.Errorf("registration location not found: %w", err)
	}
	return toRegistrationLocation(loc), nil
}

// ListRegistrationLocationsByEvent retrieves all registration locations for an event
func (r *Repository) ListRegistrationLocationsByEvent(ctx context.Context, eventID uuid.UUID) ([]RegistrationLocation, error) {
	locations, err := r.store.Queries.ListRegistrationLocationsByEvent(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to list registration locations: %w", err)
	}

	result := make([]RegistrationLocation, len(locations))
	for i, l := range locations {
		result[i] = toRegistrationLocation(l)
	}
	return result, nil
}

// UpdateRegistrationLocation updates a registration location
func (r *Repository) UpdateRegistrationLocation(ctx context.Context, id uuid.UUID, req UpdateRegistrationLocationRequest) (RegistrationLocation, error) {
	loc, err := r.store.Queries.UpdateRegistrationLocation(ctx, repo.UpdateRegistrationLocationParams{
		ID:           id,
		LocationName: req.LocationName,
		Address:      req.Address,
		Schedule:     convert.ToPgText(req.Schedule),
	})
	if err != nil {
		return RegistrationLocation{}, fmt.Errorf("failed to update registration location: %w", err)
	}
	return toRegistrationLocation(loc), nil
}

// DeleteRegistrationLocation removes a registration location
func (r *Repository) DeleteRegistrationLocation(ctx context.Context, id uuid.UUID) error {
	err := r.store.Queries.DeleteRegistrationLocation(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete registration location: %w", err)
	}
	return nil
}

// EVENT PARTICIPANT FIELD operations

// UpsertEventParticipantFields upserts participant field configurations for an event
func (r *Repository) UpsertEventParticipantFields(ctx context.Context, eventID uuid.UUID, fields []ParticipantFieldConfigInput) error {
	// First delete existing fields
	err := r.store.Queries.DeleteEventParticipantFieldsByEvent(ctx, eventID)
	if err != nil {
		return fmt.Errorf("failed to delete existing participant fields: %w", err)
	}

	// Then insert new fields
	for i, f := range fields {
		_, err := r.store.Queries.UpsertEventParticipantField(ctx, repo.UpsertEventParticipantFieldParams{
			EventID:     eventID,
			FieldName:   f.FieldName,
			IsRequired:  f.IsRequired,
			IsEnabled:   f.IsEnabled,
			DisplayOrder: pgtype.Int4{Int32: int32(i), Valid: true},
		})
		if err != nil {
			return fmt.Errorf("failed to upsert participant field %s: %w", f.FieldName, err)
		}
	}

	return nil
}

// ============ EVENT FIELD CONFIG operations ============

// ListEventFieldConfigByEvent retrieves all field configs for an event
func (r *Repository) ListEventFieldConfigByEvent(ctx context.Context, eventID uuid.UUID) ([]EventFieldConfig, error) {
	configs, err := r.store.Queries.ListEventFieldConfigByEvent(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to list event field configs: %w", err)
	}

	result := make([]EventFieldConfig, len(configs))
	for i, c := range configs {
		result[i] = toEventFieldConfig(c)
	}
	return result, nil
}

// UpsertEventFieldConfigs upserts field configurations for an event (replaces all)
func (r *Repository) UpsertEventFieldConfigs(ctx context.Context, eventID uuid.UUID, configs []EventFieldConfigInput) error {
	// First delete existing configs
	err := r.store.Queries.DeleteEventFieldConfigByEvent(ctx, eventID)
	if err != nil {
		return fmt.Errorf("failed to delete existing event field configs: %w", err)
	}

	// Then insert new configs
	for _, c := range configs {
		_, err := r.store.Queries.UpsertEventFieldConfig(ctx, repo.UpsertEventFieldConfigParams{
			EventID:     eventID,
			SectionName: c.SectionName,
			IsEnabled:   c.IsEnabled,
			IsRequired:  c.IsRequired,
		})
		if err != nil {
			return fmt.Errorf("failed to upsert event field config %s: %w", c.SectionName, err)
		}
	}

	return nil
}

// ============ Model conversion functions ============

func toEvent(e repo.Event) Event {
	var itinerary []ItineraryItem
	if len(e.Itinerary) > 0 {
		if err := json.Unmarshal(e.Itinerary, &itinerary); err != nil {
			itinerary = []ItineraryItem{}
		}
	}

	return Event{
		ID:           e.ID,
		OrganizerID:  e.OrganizerID,
		Name:         e.Name,
		Slug:         e.Slug,
		Description:  convert.PgTextToString(e.Description),
		EventDate:    convert.PgTimeToTime(e.EventDate),
		Deadline:     e.Deadline,
		State:        e.State,
		Municipality: e.Municipality,
		Address:      e.Address,
		LogoURL:      e.LogoUrl,
		Itinerary:    itinerary,
		Judges:       convert.PgTextToString(e.Judges),
		Rules:        convert.PgTextToString(e.Rules),
		Risks:        convert.PgTextToString(e.Risks),
		Transit:      convert.PgTextToString(e.Transit),
		Status:       convert.PgTextToString(e.Status),
		CreatedAt:    convert.PgTimeToTime(e.CreatedAt),
		UpdatedAt:    convert.PgTimeToTime(e.UpdatedAt),
	}
}

func toEventFieldConfig(f repo.EventFieldConfig) EventFieldConfig {
	return EventFieldConfig{
		EventID:     f.EventID,
		SectionName: f.SectionName,
		IsEnabled:   f.IsEnabled,
		IsRequired:  f.IsRequired,
	}
}

func toDistance(d repo.Distance) Distance {
	return Distance{
		ID:        d.ID,
		EventID:   d.EventID,
		KM:        convert.PgNumericToFloat64(d.Km),
		Capacity:  int(d.Capacity),
		Surface:   d.Surface,
		Elevation: convert.PgInt4ToIntPtr(d.Elevation),
		TimeLimit: convert.PgTextToString(d.TimeLimit),
		StartLat:  convert.PgNumericToFloat64Ptr(d.StartLat),
		StartLng:  convert.PgNumericToFloat64Ptr(d.StartLng),
		EndLat:    convert.PgNumericToFloat64Ptr(d.EndLat),
		EndLng:    convert.PgNumericToFloat64Ptr(d.EndLng),
		CreatedAt: convert.PgTimeToTime(d.CreatedAt),
	}
}

func toAward(a repo.Award) Award {
	return Award{
		ID:            a.ID,
		DistanceID:    a.DistanceID,
		AgeCategoryID: a.AgeCategoryID,
		Gender:        a.Gender,
		Place:         int(a.Place),
		PrizeAmount:   convert.PgNumericToFloat64(a.PrizeAmount),
		CreatedAt:     convert.PgTimeToTime(a.CreatedAt),
	}
}

func toPricingStage(p repo.PricingStage) PricingStage {
	return PricingStage{
		ID:         p.ID,
		DistanceID: p.DistanceID,
		Name:       convert.PgTextToString(p.Name),
		Price:      convert.PgNumericToFloat64(p.Price),
		StartDate:  p.StartDate,
		EndDate:    p.EndDate,
		CreatedAt:  convert.PgTimeToTime(p.CreatedAt),
	}
}

func toSponsor(s repo.Sponsor) Sponsor {
	return Sponsor{
		ID:         s.ID,
		EventID:    s.EventID,
		Name:       s.Name,
		Tier:       s.Tier,
		LogoURL:    convert.PgTextToString(s.LogoUrl),
		WebsiteURL: convert.PgTextToString(s.WebsiteUrl),
		CreatedAt:  convert.PgTimeToTime(s.CreatedAt),
	}
}

func toRunnerKitItem(k repo.RunnerKitItem) RunnerKitItem {
	return RunnerKitItem{
		ID:          k.ID,
		EventID:     k.EventID,
		Name:        k.Name,
		Icon:        k.Icon,
		CreatedAt:   convert.PgTimeToTime(k.CreatedAt),
	}
}

func toRegistrationLocation(r repo.RegistrationLocation) RegistrationLocation {
	return RegistrationLocation{
		ID:           r.ID,
		EventID:      r.EventID,
		LocationName: r.LocationName,
		Address:      r.Address,
		Schedule:     convert.PgTextToString(r.Schedule),
		CreatedAt:    convert.PgTimeToTime(r.CreatedAt),
	}
}
