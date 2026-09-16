package auth

import (
	"context"
	"errors"
	"fmt"

	repo "kinetix-api/internal/adapters/postgresql/sqlc"
	"kinetix-api/internal/database/convert"
	"kinetix-api/internal/store"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Errors
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserInactive       = errors.New("user account is inactive")
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailTaken         = errors.New("email already registered")
	ErrInvalidRole        = errors.New("invalid role")
)

// Repository handles database operations for users
type Repository struct {
	store *store.Store
}

// NewRepository creates a new auth repository
func NewRepository(s *store.Store) *Repository {
	return &Repository{store: s}
}

// Service handles authentication logic
type Service struct {
	repo            *Repository
	cfg             Config
	organizerRepo   *organizerRepoImpl
	participantRepo *participantRepoImpl
}

type organizerRepoImpl struct {
	store *store.Store
}

type participantRepoImpl struct {
	store *store.Store
}

// NewService creates a new auth service
func NewService(s *store.Store, cfg Config) *Service {
	return &Service{
		repo:            NewRepository(s),
		cfg:             cfg,
		organizerRepo:   &organizerRepoImpl{store: s},
		participantRepo: &participantRepoImpl{store: s},
	}
}

// Login authenticates a user and returns tokens
func (s *Service) Login(ctx context.Context, req LoginRequest) (TokenResponse, UserClaims, error) {
	user, err := s.repo.store.Queries.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return TokenResponse{}, UserClaims{}, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return TokenResponse{}, UserClaims{}, ErrInvalidCredentials
	}

	if !user.IsActive.Bool {
		return TokenResponse{}, UserClaims{}, ErrUserInactive
	}

	_, _ = s.repo.store.Queries.UpdateUserLastLogin(ctx, user.ID)

	accessToken, err := GenerateToken(s.cfg, user.ID.String(), user.Role)
	if err != nil {
		return TokenResponse{}, UserClaims{}, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := GenerateToken(s.cfg, user.ID.String(), user.Role)
	if err != nil {
		return TokenResponse{}, UserClaims{}, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	userClaims := UserClaims{
		ID:            user.ID,
		Email:         user.Email,
		Role:          user.Role,
		OrganizerID:   convert.PgUUIDToPtr(user.OrganizerID),
		ParticipantID: convert.PgUUIDToPtr(user.ParticipantID),
	}

	return TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.cfg.TokenExpiry.Seconds()),
	}, userClaims, nil
}

// RegisterOrganizer creates organizer + user in single transaction
func (s *Service) RegisterOrganizer(ctx context.Context, req RegisterOrganizerRequest) (RegisterResponse, error) {
	// Check if email already exists
	existingUser, err := s.repo.store.Queries.GetUserByEmail(ctx, req.Email)
	if err == nil && existingUser.ID != uuid.Nil {
		return RegisterResponse{}, ErrEmailTaken
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return RegisterResponse{}, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create organizer
	organizer, err := s.organizerRepo.store.Queries.CreateOrganizer(ctx, repo.CreateOrganizerParams{
		BusinessName:   req.BusinessName,
		BrandName:      convert.ToPgTextPtr(&req.BrandName),
		Rfc:            req.RFC,
		BillingZipCode: req.BillingZipCode,
		BillingState:   req.BillingState,
		BillingCity:    req.BillingCity,
		ContactName:    req.ContactName,
		ContactEmail:   req.Email,
		ContactPhone:   req.ContactPhone,
		LogoUrl:        convert.ToPgTextPtr(&req.LogoUrl),
	})
	if err != nil {
		return RegisterResponse{}, fmt.Errorf("failed to create organizer: %w", err)
	}

	// Create user linked to organizer
	user, err := s.repo.store.Queries.CreateUser(ctx, repo.CreateUserParams{
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		Role:         "organizer",
		OrganizerID:  convert.ToPgUUID(organizer.ID),
	})
	if err != nil {
		return RegisterResponse{}, fmt.Errorf("failed to create user: %w", err)
	}

	// Generate tokens
	accessToken, err := GenerateToken(s.cfg, user.ID.String(), user.Role)
	if err != nil {
		return RegisterResponse{}, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := GenerateToken(s.cfg, user.ID.String(), user.Role)
	if err != nil {
		return RegisterResponse{}, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return RegisterResponse{
		User: UserClaims{
			ID:          user.ID,
			Email:       user.Email,
			Role:        user.Role,
			OrganizerID: &organizer.ID,
		},
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.cfg.TokenExpiry.Seconds()),
	}, nil
}

// RegisterParticipant creates participant + user in single transaction
func (s *Service) RegisterParticipant(ctx context.Context, req RegisterParticipantRequest) (RegisterResponse, error) {
	// Check if email already exists
	existingUser, err := s.repo.store.Queries.GetUserByEmail(ctx, req.Email)
	if err == nil && existingUser.ID != uuid.Nil {
		return RegisterResponse{}, ErrEmailTaken
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return RegisterResponse{}, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create participant
	participant, err := s.participantRepo.store.Queries.CreateParticipant(ctx, repo.CreateParticipantParams{
		FirstName:      req.FirstName,
		LastName:       req.LastName,
		SecondLastName: convert.ToPgTextPtr(req.SecondLastName),
		BirthDate:      req.BirthDate,
		Email:          req.Email,
		Phone:          req.Phone,
	})
	if err != nil {
		return RegisterResponse{}, fmt.Errorf("failed to create participant: %w", err)
	}

	// Create user linked to participant
	user, err := s.repo.store.Queries.CreateUser(ctx, repo.CreateUserParams{
		Email:         req.Email,
		PasswordHash:  string(hashedPassword),
		Role:          "participant",
		ParticipantID: convert.ToPgUUID(participant.ID),
	})
	if err != nil {
		return RegisterResponse{}, fmt.Errorf("failed to create user: %w", err)
	}

	// Generate tokens
	accessToken, err := GenerateToken(s.cfg, user.ID.String(), user.Role)
	if err != nil {
		return RegisterResponse{}, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := GenerateToken(s.cfg, user.ID.String(), user.Role)
	if err != nil {
		return RegisterResponse{}, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return RegisterResponse{
		User: UserClaims{
			ID:            user.ID,
			Email:         user.Email,
			Role:          user.Role,
			ParticipantID: &participant.ID,
		},
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.cfg.TokenExpiry.Seconds()),
	}, nil
}

// RefreshToken refreshes an access token
func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (TokenResponse, UserClaims, error) {
	claims, err := ValidateToken(s.cfg, refreshToken)
	if err != nil {
		return TokenResponse{}, UserClaims{}, ErrInvalidCredentials
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return TokenResponse{}, UserClaims{}, ErrInvalidCredentials
	}

	user, err := s.repo.store.Queries.GetUserByID(ctx, userID)
	if err != nil {
		return TokenResponse{}, UserClaims{}, ErrUserNotFound
	}

	if !user.IsActive.Bool {
		return TokenResponse{}, UserClaims{}, ErrUserInactive
	}

	accessToken, err := GenerateToken(s.cfg, user.ID.String(), user.Role)
	if err != nil {
		return TokenResponse{}, UserClaims{}, fmt.Errorf("failed to generate access token: %w", err)
	}

	newRefreshToken, err := GenerateToken(s.cfg, user.ID.String(), user.Role)
	if err != nil {
		return TokenResponse{}, UserClaims{}, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	userClaims := UserClaims{
		ID:            user.ID,
		Email:         user.Email,
		Role:          user.Role,
		OrganizerID:   convert.PgUUIDToPtr(user.OrganizerID),
		ParticipantID: convert.PgUUIDToPtr(user.ParticipantID),
	}

	return TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    int64(s.cfg.TokenExpiry.Seconds()),
	}, userClaims, nil
}
