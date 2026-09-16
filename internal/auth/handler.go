package auth

import (
	"net/http"

	jsonlib "kinetix-api/internal/json"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

// Handler handles auth HTTP requests
type Handler struct {
	service  *Service
	validate *validator.Validate
}

// NewHandler creates a new auth handler
func NewHandler(service *Service, validate *validator.Validate) *Handler {
	return &Handler{
		service:  service,
		validate: validate,
	}
}

// Routes registers auth routes
func (h *Handler) Routes(r chi.Router) {
	r.Post("/login", h.Login)
	r.Post("/register/organizer", h.RegisterOrganizer)
	r.Post("/register/participant", h.RegisterParticipant)
	r.Post("/refresh", h.RefreshToken)
}

// Login handles POST /auth/login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := jsonlib.Read(r, &req); err != nil {
		jsonlib.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.validate.Struct(&req); err != nil {
		jsonlib.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	tokens, user, err := h.service.Login(r.Context(), req)
	if err != nil {
		switch err {
		case ErrInvalidCredentials:
			jsonlib.WriteError(w, http.StatusUnauthorized, "invalid email or password")
		case ErrUserInactive:
			jsonlib.WriteError(w, http.StatusForbidden, "account is inactive")
		default:
			jsonlib.WriteError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	jsonlib.Write(w, http.StatusOK, RegisterResponse{
		User: user,
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
	})
}

// RegisterOrganizer handles POST /auth/register/organizer
func (h *Handler) RegisterOrganizer(w http.ResponseWriter, r *http.Request) {
	var req RegisterOrganizerRequest
	if err := jsonlib.Read(r, &req); err != nil {
		jsonlib.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.validate.Struct(&req); err != nil {
		jsonlib.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	response, err := h.service.RegisterOrganizer(r.Context(), req)
	if err != nil {
		switch err {
		case ErrEmailTaken:
			jsonlib.WriteError(w, http.StatusConflict, "email already registered")
		default:
			jsonlib.WriteError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	jsonlib.Write(w, http.StatusCreated, response)
}

// RegisterParticipant handles POST /auth/register/participant
func (h *Handler) RegisterParticipant(w http.ResponseWriter, r *http.Request) {
	var req RegisterParticipantRequest
	if err := jsonlib.Read(r, &req); err != nil {
		jsonlib.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.validate.Struct(&req); err != nil {
		jsonlib.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	response, err := h.service.RegisterParticipant(r.Context(), req)
	if err != nil {
		switch err {
		case ErrEmailTaken:
			jsonlib.WriteError(w, http.StatusConflict, "email already registered")
		default:
			jsonlib.WriteError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	jsonlib.Write(w, http.StatusCreated, response)
}

// RefreshToken handles POST /auth/refresh
func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req RefreshTokenRequest
	if err := jsonlib.Read(r, &req); err != nil {
		jsonlib.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.validate.Struct(&req); err != nil {
		jsonlib.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	tokens, _, err := h.service.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		jsonlib.WriteError(w, http.StatusUnauthorized, "invalid or expired refresh token")
		return
	}

	jsonlib.Write(w, http.StatusOK, AuthResponse{
		Message: "token refreshed",
		Data:    tokens,
	})
}
