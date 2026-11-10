package auth

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"

	repo "kinetix-api/internal/adapters/postgresql/sqlc"
	"kinetix-api/internal/database/convert"
	"kinetix-api/internal/store"
)

// Errors
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserInactive       = errors.New("user account is inactive")
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailTaken         = errors.New("email already registered")
	ErrInvalidRole        = errors.New("invalid role")
)

// Repository is the data access the auth service needs. Declared by the
// consumer (PATTERNS) so tests can hand in fakes.
type Repository interface {
	GetUserByEmail(ctx context.Context, email string) (repo.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (repo.User, error)
	CreateOrganizer(ctx context.Context, p repo.CreateOrganizerParams) (repo.Organizer, error)
	CreateParticipant(ctx context.Context, p repo.CreateParticipantParams) (repo.Participant, error)
	CreateUser(ctx context.Context, p repo.CreateUserParams) (repo.User, error)
	UpdateUserLastLogin(ctx context.Context, id uuid.UUID) (repo.User, error)
}

// TxRunner swaps the repository for a transaction-scoped one for the
// duration of fn (store.ExecTx, ADR-14).
type TxRunner interface {
	ExecTx(ctx context.Context, fn func(Repository) error) error
}

// authRepository wraps the store and satisfies both Repository and TxRunner.
type authRepository struct {
	store *store.Store
}

// NewRepository creates a new auth repository
func NewRepository(s *store.Store) Repository { return &authRepository{store: s} }

func (r *authRepository) GetUserByEmail(ctx context.Context, email string) (repo.User, error) {
	return r.store.Queries.GetUserByEmail(ctx, email)
}

func (r *authRepository) GetUserByID(ctx context.Context, id uuid.UUID) (repo.User, error) {
	return r.store.Queries.GetUserByID(ctx, id)
}

func (r *authRepository) CreateOrganizer(ctx context.Context, p repo.CreateOrganizerParams) (repo.Organizer, error) {
	return r.store.Queries.CreateOrganizer(ctx, p)
}

func (r *authRepository) CreateParticipant(ctx context.Context, p repo.CreateParticipantParams) (repo.Participant, error) {
	return r.store.Queries.CreateParticipant(ctx, p)
}

func (r *authRepository) CreateUser(ctx context.Context, p repo.CreateUserParams) (repo.User, error) {
	return r.store.Queries.CreateUser(ctx, p)
}

func (r *authRepository) UpdateUserLastLogin(ctx context.Context, id uuid.UUID) (repo.User, error) {
	return r.store.Queries.UpdateUserLastLogin(ctx, id)
}

// ExecTx adapts store.ExecTx to a transaction-scoped Repository.
func (r *authRepository) ExecTx(ctx context.Context, fn func(Repository) error) error {
	return r.store.ExecTx(ctx, func(q *repo.Queries) error {
		return fn(&authRepository{store: r.store.WithQueries(q)})
	})
}

// Service handles authentication logic
type Service struct {
	repo Repository
	tx   TxRunner
	cfg  Config
}

// NewService creates a new auth service
func NewService(s *store.Store, cfg Config) *Service {
	return &Service{repo: NewRepository(s), tx: &authRepository{store: s}, cfg: cfg}
}

// bcryptHash hashes a password at the standard bcrypt cost.
func bcryptHash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

// Login authenticates a user and returns tokens
func (s *Service) Login(ctx context.Context, req LoginRequest) (TokenResponse, UserClaims, error) {
	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return TokenResponse{}, UserClaims{}, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return TokenResponse{}, UserClaims{}, ErrInvalidCredentials
	}

	if !user.IsActive.Bool {
		return TokenResponse{}, UserClaims{}, ErrUserInactive
	}

	if _, err := s.repo.UpdateUserLastLogin(ctx, user.ID); err != nil {
		log.Printf("auth: could not update last login for user %s: %v", user.ID, err)
	}

	accessToken, err := GenerateToken(s.cfg, user.ID.String(), user.Role)
	if err != nil {
		return TokenResponse{}, UserClaims{}, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := GenerateRefreshToken(s.cfg, user.ID.String(), user.Role)
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

// RegisterOrganizer creates an organizer and their user account in one
// transaction. A failing user insert rolls the organizer back.
func (s *Service) RegisterOrganizer(ctx context.Context, req RegisterOrganizerRequest) (RegisterResponse, error) {
	if err := s.checkEmailAvailable(ctx, req.Email); err != nil {
		return RegisterResponse{}, err
	}

	hashedPassword, err := bcryptHash(req.Password)
	if err != nil {
		return RegisterResponse{}, err
	}

	var user repo.User
	var organizer repo.Organizer
	err = s.tx.ExecTx(ctx, func(tx Repository) error {
		organizer, err = tx.CreateOrganizer(ctx, repo.CreateOrganizerParams{
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
			return fmt.Errorf("failed to create organizer: %w", err)
		}

		user, err = tx.CreateUser(ctx, repo.CreateUserParams{
			Email:        req.Email,
			PasswordHash: hashedPassword,
			Role:         "organizer",
			OrganizerID:  convert.ToPgUUID(organizer.ID),
		})
		if err != nil {
			if isDuplicateKey(err) {
				return fmt.Errorf("user email: %w", ErrEmailTaken)
			}
			return fmt.Errorf("failed to create user: %w", err)
		}
		return nil
	})
	if err != nil {
		return RegisterResponse{}, err
	}

	accessToken, refreshToken, expiresIn, err := s.issueTokensFor(user.ID.String(), user.Role)
	if err != nil {
		return RegisterResponse{}, err
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
		ExpiresIn:    expiresIn,
	}, nil
}

// RegisterParticipant creates a participant and their user account in one
// transaction. A failing user insert rolls the participant back.
func (s *Service) RegisterParticipant(ctx context.Context, req RegisterParticipantRequest) (RegisterResponse, error) {
	if err := s.checkEmailAvailable(ctx, req.Email); err != nil {
		return RegisterResponse{}, err
	}

	hashedPassword, err := bcryptHash(req.Password)
	if err != nil {
		return RegisterResponse{}, err
	}

	var user repo.User
	var participant repo.Participant
	err = s.tx.ExecTx(ctx, func(tx Repository) error {
		participant, err = tx.CreateParticipant(ctx, repo.CreateParticipantParams{
			FirstName:      req.FirstName,
			LastName:       req.LastName,
			SecondLastName: convert.ToPgTextPtr(req.SecondLastName),
			BirthDate:      req.BirthDate,
			Email:          req.Email,
			Phone:          req.Phone,
		})
		if err != nil {
			return fmt.Errorf("failed to create participant: %w", err)
		}

		user, err = tx.CreateUser(ctx, repo.CreateUserParams{
			Email:         req.Email,
			PasswordHash:  hashedPassword,
			Role:          "participant",
			ParticipantID: convert.ToPgUUID(participant.ID),
		})
		if err != nil {
			if isDuplicateKey(err) {
				return fmt.Errorf("user email: %w", ErrEmailTaken)
			}
			return fmt.Errorf("failed to create user: %w", err)
		}
		return nil
	})
	if err != nil {
		return RegisterResponse{}, err
	}

	accessToken, refreshToken, expiresIn, err := s.issueTokensFor(user.ID.String(), user.Role)
	if err != nil {
		return RegisterResponse{}, err
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
		ExpiresIn:    expiresIn,
	}, nil
}

// RefreshToken issues a new token pair for a valid refresh token holder.
func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (TokenResponse, UserClaims, error) {
	claims, err := ValidateRefreshToken(s.cfg, refreshToken)
	if err != nil {
		return TokenResponse{}, UserClaims{}, ErrInvalidCredentials
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return TokenResponse{}, UserClaims{}, ErrInvalidCredentials
	}

	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return TokenResponse{}, UserClaims{}, ErrUserNotFound
	}

	if !user.IsActive.Bool {
		return TokenResponse{}, UserClaims{}, ErrUserInactive
	}

	accessToken, refreshToken, expiresIn, err := s.issueTokensFor(user.ID.String(), user.Role)
	if err != nil {
		return TokenResponse{}, UserClaims{}, err
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
		ExpiresIn:    expiresIn,
	}, userClaims, nil
}

// checkEmailAvailable rejects already registered emails with a cheap read
// before the write. The users_email unique constraint is authoritative.
func (s *Service) checkEmailAvailable(ctx context.Context, email string) error {
	existing, err := s.repo.GetUserByEmail(ctx, email)
	if err == nil && existing.ID != uuid.Nil {
		return ErrEmailTaken
	}
	return nil
}

// isDuplicateKey reports PostgreSQL unique-violation (23505) errors.
func isDuplicateKey(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// issueTokensFor builds the access/refresh token pair for a user.
func (s *Service) issueTokensFor(userID, role string) (string, string, int64, error) {
	accessToken, err := GenerateToken(s.cfg, userID, role)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := GenerateRefreshToken(s.cfg, userID, role)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to generate refresh token: %w", err)
	}
	return accessToken, refreshToken, int64(s.cfg.TokenExpiry.Seconds()), nil
}
