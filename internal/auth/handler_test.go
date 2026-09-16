package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
)

// helper to build an HTTP request with a Bearer header
func newAuthRequest(headerValue string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	if headerValue != "" {
		r.Header.Set("Authorization", headerValue)
	}
	return r
}

var validate = validator.New()

func validateStruct(v any) error {
	return validate.Struct(v)
}

func TestLoginRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     LoginRequest
		wantErr bool
	}{
		{name: "valid", req: LoginRequest{Email: "user@example.com", Password: "longenoughpw"}, wantErr: false},
		{name: "missing email", req: LoginRequest{Email: "", Password: "longenoughpw"}, wantErr: true},
		{name: "missing password", req: LoginRequest{Email: "user@example.com", Password: ""}, wantErr: true},
		{name: "short password", req: LoginRequest{Email: "user@example.com", Password: "short"}, wantErr: true},
		{name: "bad email", req: LoginRequest{Email: "no-at-sign", Password: "longenoughpw"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateStruct(tt.req)
			if tt.wantErr && err == nil {
				t.Fatal("expected validation error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

func TestRefreshTokenRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     RefreshTokenRequest
		wantErr bool
	}{
		{name: "valid", req: RefreshTokenRequest{RefreshToken: "some.token.value"}, wantErr: false},
		{name: "empty", req: RefreshTokenRequest{RefreshToken: ""}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateStruct(tt.req)
			if tt.wantErr && err == nil {
				t.Fatal("expected validation error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestRegisterParticipantRequest_Validate(t *testing.T) {
	birth := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		req     RegisterParticipantRequest
		wantErr bool
	}{
		{
			name: "valid",
			req: RegisterParticipantRequest{
				Email:     "user@example.com",
				Password:  "longenoughpw",
				Gender:    "male",
				FirstName: "John",
				LastName:  "Doe",
				Phone:     "+5255555555",
				BirthDate: birth,
			},
			wantErr: false,
		},
		{
			name: "missing email",
			req: RegisterParticipantRequest{
				Email: "", Password: "longenoughpw", Gender: "male",
				FirstName: "John", LastName: "Doe",
				Phone:     "+5255555555",
				BirthDate: birth,
			},
			wantErr: true,
		},
		{
			name: "short password",
			req: RegisterParticipantRequest{
				Email: "user@example.com", Password: "short", Gender: "male",
				FirstName: "John", LastName: "Doe",
				Phone:     "+5255555555",
				BirthDate: birth,
			},
			wantErr: true,
		},
		{
			name: "missing first name",
			req: RegisterParticipantRequest{
				Email: "user@example.com", Password: "longenoughpw", Gender: "male",
				FirstName: "", LastName: "Doe",
				Phone:     "+5255555555",
				BirthDate: birth,
			},
			wantErr: true,
		},
		{
			name: "missing last name",
			req: RegisterParticipantRequest{
				Email: "user@example.com", Password: "longenoughpw", Gender: "male",
				FirstName: "John", LastName: "",
				Phone:     "+5255555555",
				BirthDate: birth,
			},
			wantErr: true,
		},
		{
			name: "missing phone",
			req: RegisterParticipantRequest{
				Email: "user@example.com", Password: "longenoughpw", Gender: "male",
				FirstName: "John", LastName: "Doe",
				Phone:     "",
				BirthDate: birth,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateStruct(tt.req)
			if tt.wantErr && err == nil {
				t.Fatal("expected validation error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestRegisterOrganizerRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     RegisterOrganizerRequest
		wantErr bool
	}{
		{
			name: "valid",
			req: RegisterOrganizerRequest{
				Email:          "owner@example.com",
				Password:       "longenoughpw",
				BusinessName:   "Acme Corp",
				BrandName:      "Acme",
				RFC:            "ABC010101ABC",
				BillingZipCode: "44100",
				BillingState:   "Jalisco",
				BillingCity:    "Guadalajara",
				ContactName:    "John Doe",
				ContactEmail:   "owner@example.com",
				ContactPhone:   "+5255555555",
			},
			wantErr: false,
		},
		{
			name: "bad rfc length",
			req: RegisterOrganizerRequest{
				Email: "owner@example.com", Password: "longenoughpw",
				BusinessName: "Acme Corp", BrandName: "Acme",
				RFC: "ABC", BillingZipCode: "44100", BillingState: "Jalisco", BillingCity: "Guadalajara",
				ContactName: "John Doe", ContactEmail: "owner@example.com", ContactPhone: "+5255555555",
			},
			wantErr: true,
		},
		{
			name: "missing business name",
			req: RegisterOrganizerRequest{
				Email: "owner@example.com", Password: "longenoughpw",
				BusinessName: "", BrandName: "Acme",
				RFC: "ABC010101ABC", BillingZipCode: "44100", BillingState: "Jalisco", BillingCity: "Guadalajara",
				ContactName: "John Doe", ContactEmail: "owner@example.com", ContactPhone: "+5255555555",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateStruct(tt.req)
			if tt.wantErr && err == nil {
				t.Fatal("expected validation error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestExtractTokenFromHeader_BearerVariations(t *testing.T) {
	tests := []struct {
		name      string
		header    string
		wantToken string
		wantErr   bool
	}{
		{name: "lowercase bearer rejected (case-sensitive)", header: "bearer abc", wantErr: true},
		{name: "double space returns second token", header: "Bearer  abc", wantToken: " abc"},
		{name: "trailing space is preserved", header: "Bearer abc ", wantToken: "abc "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newAuthRequest(tt.header)
			tok, err := ExtractTokenFromHeader(r)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got token %q", tok)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tok != tt.wantToken {
				t.Errorf("want %q, got %q", tt.wantToken, tok)
			}
		})
	}
}

// keep strings import used to avoid unused import error
var _ = strings.TrimSpace