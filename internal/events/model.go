package events

import (
	"time"

	"github.com/google/uuid"
)

// ItineraryItem represents a single item in the event itinerary (JSONB)
type ItineraryItem struct {
	Key  string `json:"key"  validate:"required,min=1,max=50"`
	Time string `json:"time" validate:"required"`
	Icon string `json:"icon" validate:"required,min=1,max=50"`
}

// Event represents an event entity
type Event struct {
	ID           uuid.UUID      `json:"id"`
	OrganizerID  uuid.UUID      `json:"organizer_id"`
	Name         string         `json:"name"`
	Slug         string         `json:"slug"`
	Description  string         `json:"description"`
	EventDate    time.Time      `json:"event_date"`
	Deadline     time.Time      `json:"deadline"`
	State        string         `json:"state"`
	Municipality string         `json:"municipality"`
	Address      string         `json:"address"`
	LogoURL      string         `json:"logo_url,omitempty"`
	Judges       string         `json:"judges"`
	Rules        string         `json:"rules"`
	Risks        string         `json:"risks"`
	Transit      string         `json:"transit"`
	Itinerary    []ItineraryItem `json:"itinerary"`
	Status       string         `json:"status"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// Distance represents a distance entity
type Distance struct {
	ID        uuid.UUID `json:"id"`
	EventID   uuid.UUID `json:"event_id"`
	KM        float64   `json:"km"`
	Capacity  int       `json:"capacity"`
	Surface   string    `json:"surface"`
	Elevation *int      `json:"elevation,omitempty"`
	TimeLimit string    `json:"time_limit,omitempty"`
	StartLat  *float64  `json:"start_lat,omitempty"`
	StartLng  *float64  `json:"start_lng,omitempty"`
	EndLat    *float64  `json:"end_lat,omitempty"`
	EndLng    *float64  `json:"end_lng,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Award represents an award entity
type Award struct {
	ID             uuid.UUID `json:"id"`
	DistanceID     uuid.UUID `json:"distance_id"`
	AgeCategoryID  uuid.UUID `json:"age_category_id"`
	Gender         string    `json:"gender"`
	Place          int       `json:"place"`
	PrizeAmount    float64   `json:"prize_amount"`
	CreatedAt      time.Time `json:"created_at"`
}

// PricingStage represents a pricing stage entity
type PricingStage struct {
	ID          uuid.UUID `json:"id"`
	DistanceID  uuid.UUID `json:"distance_id"`
	Name        string    `json:"name"`
	Price       float64   `json:"price"`
	StartDate   time.Time `json:"start_date"`
	EndDate     time.Time `json:"end_date"`
	IsEarlyBird bool      `json:"is_early_bird"`
	CreatedAt   time.Time `json:"created_at"`
}

// Sponsor represents a sponsor entity
type Sponsor struct {
	ID         uuid.UUID `json:"id"`
	EventID    uuid.UUID `json:"event_id"`
	Name       string    `json:"name"`
	Tier       string    `json:"tier"`
	LogoURL    string    `json:"logo_url,omitempty"`
	WebsiteURL string    `json:"website_url,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// RegistrationLocation represents a registration location entity
type RegistrationLocation struct {
	ID           uuid.UUID `json:"id"`
	EventID      uuid.UUID `json:"event_id"`
	LocationName string    `json:"location_name"`
	Address      string    `json:"address"`
	Schedule     string    `json:"schedule"`
	CreatedAt    time.Time `json:"created_at"`
}

// EventParticipantField represents a participant field configuration for an event
type EventParticipantField struct {
	ID          uuid.UUID `json:"id"`
	EventID     uuid.UUID `json:"event_id"`
	FieldName   string    `json:"field_name"`
	IsRequired  bool      `json:"is_required"`
	IsEnabled   bool      `json:"is_enabled"`
	DisplayOrder int      `json:"display_order"`
	CreatedAt   time.Time `json:"created_at"`
}

// ParticipantFieldConfig represents the config returned in event detail
type ParticipantFieldConfig struct {
	FieldName  string `json:"field_name"`
	IsRequired bool   `json:"is_required"`
	IsEnabled  bool   `json:"is_enabled"`
}

// UpdateParticipantFieldsRequest represents the request to update participant field config
type UpdateParticipantFieldsRequest struct {
	Fields []ParticipantFieldConfigInput `json:"fields" validate:"required,dive"`
}

// ParticipantFieldConfigInput represents a single field config in the update request
type ParticipantFieldConfigInput struct {
	FieldName  string `json:"field_name" validate:"required"`
	IsRequired bool   `json:"is_required"`
	IsEnabled  bool   `json:"is_enabled"`
}

// RunnerKitItem represents a single item in the runner kit (separate table)
type RunnerKitItem struct {
	ID          uuid.UUID `json:"id"`
	EventID     uuid.UUID `json:"event_id"`
	Name        string    `json:"name"`
	Icon        string    `json:"icon"`
	CreatedAt   time.Time `json:"created_at"`
}

// EventFieldConfig represents a toggleable section configuration for an event
type EventFieldConfig struct {
	EventID     uuid.UUID `json:"event_id"`
	SectionName string    `json:"section_name"`
	IsEnabled   bool      `json:"is_enabled"`
	IsRequired  bool      `json:"is_required"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// EventFieldConfigInput represents a single field config in the update request
type EventFieldConfigInput struct {
	SectionName string `json:"section_name" validate:"required"`
	IsEnabled   bool   `json:"is_enabled"`
	IsRequired  bool   `json:"is_required"`
}

// UpdateEventFieldConfigRequest represents the request to update event field config
type UpdateEventFieldConfigRequest struct {
	Fields []EventFieldConfigInput `json:"fields" validate:"required,dive"`
}

// CreateEventRequest represents the request body for creating an event
type CreateEventRequest struct {
	OrganizerID  uuid.UUID       `json:"organizer_id" validate:"required"`
	Name         string          `json:"name" validate:"required,min=2,max=255"`
	Slug         string          `json:"slug" validate:"required,min=2,max=255"`
	Description  string          `json:"description" validate:"omitempty,max=5000"`
	EventDate    time.Time       `json:"event_date" validate:"required"`
	Deadline     time.Time       `json:"deadline" validate:"required"`
	State        string          `json:"state" validate:"omitempty,max=100"`
	Municipality string          `json:"municipality" validate:"omitempty,max=100"`
	Address      string          `json:"address" validate:"required,min=5,max=500"`
	LogoURL      string          `json:"logo_url" validate:"omitempty,url"`
	Judges       string          `json:"judges" validate:"omitempty,max=2000"`
	Rules        string          `json:"rules" validate:"omitempty,max=5000"`
	Risks        string          `json:"risks" validate:"omitempty,max=2000"`
	Transit      string          `json:"transit" validate:"omitempty,max=2000"`
	Itinerary    []ItineraryItem `json:"itinerary" validate:"required,min=1,dive"`
}

// UpdateEventRequest represents the request body for updating an event
type UpdateEventRequest struct {
	Name         string          `json:"name" validate:"required,min=2,max=255"`
	Slug         string          `json:"slug" validate:"required,min=2,max=255"`
	Description  string          `json:"description" validate:"omitempty,max=5000"`
	EventDate    time.Time       `json:"event_date" validate:"required"`
	Deadline     time.Time       `json:"deadline" validate:"required"`
	State        string          `json:"state" validate:"omitempty,max=100"`
	Municipality string          `json:"municipality" validate:"omitempty,max=100"`
	Address      string          `json:"address" validate:"required,min=5,max=500"`
	LogoURL      string          `json:"logo_url" validate:"omitempty,url"`
	Judges       string          `json:"judges" validate:"omitempty,max=2000"`
	Rules        string          `json:"rules" validate:"omitempty,max=5000"`
	Risks        string          `json:"risks" validate:"omitempty,max=2000"`
	Transit      string          `json:"transit" validate:"omitempty,max=2000"`
	Itinerary    []ItineraryItem `json:"itinerary" validate:"required,min=1,dive"`
	Status       string          `json:"status" validate:"omitempty,oneof=draft published cancelled completed"`
}

// CreateAwardRequest represents the request body for creating an award
type CreateAwardRequest struct {
	DistanceID    uuid.UUID `json:"distance_id" validate:"required"`
	AgeCategoryID uuid.UUID `json:"age_category_id" validate:"required"`
	Gender        string    `json:"gender" validate:"required,oneof=male female general"`
	// Posición del lugar (1 = primer lugar). Default 1 si el caller omite.
	Place       int     `json:"place" validate:"omitempty,gte=1,lte=20"`
	PrizeAmount float64 `json:"prize_amount" validate:"required,gte=0"`
}

// UpdateAwardRequest represents the request body for updating an award
type UpdateAwardRequest struct {
	AgeCategoryID uuid.UUID `json:"age_category_id" validate:"required"`
	Gender        string    `json:"gender" validate:"required,oneof=male female general"`
	Place         int       `json:"place" validate:"omitempty,gte=1,lte=20"`
	PrizeAmount   float64   `json:"prize_amount" validate:"required,gte=0"`
}

// CreatePricingStageRequest represents the request body for creating a pricing stage
type CreatePricingStageRequest struct {
	DistanceID  uuid.UUID `json:"distance_id" validate:"required"`
	Name        string    `json:"name" validate:"omitempty,max=100"`
	Price       float64   `json:"price" validate:"required,gte=0"`
	StartDate   time.Time `json:"start_date" validate:"required"`
	EndDate     time.Time `json:"end_date" validate:"required"`
	IsEarlyBird bool      `json:"is_early_bird"`
}

// UpdatePricingStageRequest represents the request body for updating a pricing stage
type UpdatePricingStageRequest struct {
	Name        string    `json:"name" validate:"omitempty,max=100"`
	Price       float64   `json:"price" validate:"required,gte=0"`
	StartDate   time.Time `json:"start_date" validate:"required"`
	EndDate     time.Time `json:"end_date" validate:"required"`
	IsEarlyBird bool      `json:"is_early_bird"`
}

// CreateSponsorRequest represents the request body for creating a sponsor
type CreateSponsorRequest struct {
	EventID    uuid.UUID `json:"event_id" validate:"required"`
	Name       string    `json:"name" validate:"required,max=255"`
	Tier       string    `json:"tier" validate:"required,max=50"`
	LogoURL    string    `json:"logo_url" validate:"omitempty,url"`
	WebsiteURL string    `json:"website_url" validate:"omitempty,url"`
}

// UpdateSponsorRequest represents the request body for updating a sponsor
type UpdateSponsorRequest struct {
	Name       string `json:"name" validate:"required,max=255"`
	Tier       string `json:"tier" validate:"required,max=50"`
	LogoURL    string `json:"logo_url" validate:"omitempty,url"`
	WebsiteURL string `json:"website_url" validate:"omitempty,url"`
}

// CreateRegistrationLocationRequest represents the request body for creating a registration location
type CreateRegistrationLocationRequest struct {
	EventID      uuid.UUID `json:"event_id" validate:"required"`
	LocationName string    `json:"location_name" validate:"required,max=255"`
	Address      string    `json:"address" validate:"required,min=5,max=500"`
	Schedule     string    `json:"schedule" validate:"omitempty,max=100"`
}

// UpdateRegistrationLocationRequest represents the request body for updating a registration location
type UpdateRegistrationLocationRequest struct {
	LocationName string `json:"location_name" validate:"required,max=255"`
	Address      string `json:"address" validate:"required,min=5,max=500"`
	Schedule     string `json:"schedule" validate:"omitempty,max=100"`
}

// EventResponse represents the response structure for events
type EventResponse struct {
	Message string `json:"message"`
	Data    Event  `json:"data"`
}

// EventDetailResponse includes event with all related data
type EventDetailResponse struct {
	Message                 string                   `json:"message"`
	Data                    Event                    `json:"data"`
	Distances               []Distance               `json:"distances"`
	Awards                  []Award                  `json:"awards"`
	PricingStages           []PricingStage           `json:"pricing_stages"`
	Sponsors                []Sponsor                `json:"sponsors"`
	RegistrationLocations   []RegistrationLocation   `json:"registration_locations"`
	Tags                    []string                 `json:"tags"`
	ParticipantFieldsConfig []ParticipantFieldConfig `json:"participant_fields_config"`
	FieldConfig             []EventFieldConfig       `json:"field_config"`
	RunnerKits              []RunnerKitItem          `json:"runner_kits"`
}

// EventsListResponse represents a list of events
type EventsListResponse struct {
	Message string  `json:"message"`
	Data    []Event `json:"data"`
	Total   int64   `json:"total"`
}

// GenericResponse represents a generic response
type GenericResponse struct {
	Message string `json:"message"`
	Data    any    `json:"data"`
}
