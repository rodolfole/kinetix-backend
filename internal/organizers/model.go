package organizers

import (
	repo "kinetix-api/internal/adapters/postgresql/sqlc"
)

// Organizer alias to sqlc-generated model
type Organizer = repo.Organizer

// CreateOrganizerRequest represents the request body for creating an organizer
type CreateOrganizerRequest struct {
	BusinessName   string `json:"business_name" validate:"required,min=2,max=255"`
	BrandName      string `json:"brand_name" validate:"omitempty,min=2,max=255"`
	RFC            string `json:"rfc" validate:"required,len=12|len=13"`
	BillingZipCode string `json:"billing_zip_code" validate:"required,max=10"`
	BillingState   string `json:"billing_state" validate:"required,max=100"`
	BillingCity    string `json:"billing_city" validate:"required,max=100"`
	ContactName    string `json:"contact_name" validate:"required,min=2,max=255"`
	ContactEmail   string `json:"contact_email" validate:"required,email"`
	ContactPhone   string `json:"contact_phone" validate:"required,min=10,max=20"`
	LogoUrl        string `json:"logo_url" validate:"omitempty,url,max=500"`
}

// UpdateOrganizerRequest represents the request body for updating an organizer
type UpdateOrganizerRequest struct {
	BusinessName   string `json:"business_name" validate:"required,min=2,max=255"`
	BrandName      string `json:"brand_name" validate:"omitempty,min=2,max=255"`
	BillingZipCode string `json:"billing_zip_code" validate:"required,max=10"`
	BillingState   string `json:"billing_state" validate:"required,max=100"`
	BillingCity    string `json:"billing_city" validate:"required,max=100"`
	ContactName    string `json:"contact_name" validate:"required,min=2,max=255"`
	ContactEmail   string `json:"contact_email" validate:"required,email"`
	ContactPhone   string `json:"contact_phone" validate:"required,min=10,max=20"`
	LogoUrl        string `json:"logo_url" validate:"omitempty,url,max=500"`
}

// OrganizerResponse represents the response structure
type OrganizerResponse struct {
	Message string    `json:"message"`
	Data    Organizer `json:"data"`
}

// OrganizersListResponse represents a list of organizers
type OrganizersListResponse struct {
	Message string      `json:"message"`
	Data    []Organizer `json:"data"`
	Total   int64       `json:"total"`
}
