package auth

import (
	"time"

	"github.com/google/uuid"
)

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

// RefreshTokenRequest represents a token refresh request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// TokenResponse represents authentication tokens
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

// AuthResponse represents auth response
type AuthResponse struct {
	Message string        `json:"message"`
	Data    TokenResponse `json:"data"`
}

// UserClaims represents authenticated user info
type UserClaims struct {
	ID            uuid.UUID  `json:"id"`
	Email         string     `json:"email"`
	Role          string     `json:"role"`
	OrganizerID   *uuid.UUID `json:"organizer_id,omitempty"`
	ParticipantID *uuid.UUID `json:"participant_id,omitempty"`
}

// RegisterResponse contains created user data
type RegisterResponse struct {
	User         UserClaims `json:"user"`
	AccessToken  string     `json:"access_token"`
	RefreshToken string     `json:"refresh_token"`
	ExpiresIn    int64      `json:"expires_in"`
}

// OrganizerCreatedResponse represents organizer + user creation
type OrganizerCreatedResponse struct {
	OrganizerID uuid.UUID `json:"organizer_id"`
	UserID      uuid.UUID `json:"user_id"`
	Email       string    `json:"email"`
}

// ParticipantCreatedResponse represents participant + user creation
type ParticipantCreatedResponse struct {
	ParticipantID uuid.UUID `json:"participant_id"`
	UserID        uuid.UUID `json:"user_id"`
	Email         string    `json:"email"`
}

// ============ Combined Registration Requests ============

// RegisterOrganizerRequest represents organizer registration with user
type RegisterOrganizerRequest struct {
	Email          string `json:"email" validate:"required,email"`
	Password       string `json:"password" validate:"required,min=8"`
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

// RegisterParticipantRequest represents participant registration with user
type RegisterParticipantRequest struct {
	Email          string    `json:"email" validate:"required,email"`
	Password       string    `json:"password" validate:"required,min=8"`
	Gender         string    `json:"gender" validate:"required,max=20"`
	FirstName      string    `json:"first_name" validate:"required,max=100"`
	LastName       string    `json:"last_name" validate:"required,max=100"`
	SecondLastName *string   `json:"second_last_name" validate:"omitempty,max=100"`
	BirthDate      time.Time `json:"birth_date" validate:"required"`
	Phone          string    `json:"phone" validate:"required,min=10,max=20"`
}
