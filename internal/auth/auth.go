package auth

import (
	"errors"
	"github.com/google/uuid"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Token types carried in the "typ" claim. Access and refresh tokens are not
// interchangeable: JWTAuth accepts only access tokens and /auth/refresh only
// refresh tokens.
const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

// Claims represents JWT claims. OrganizerID/ParticipantID link the token to
// the profile so ownership checks are pure (PLAN-16 T7). Tokens issued
// before the claim existed carry nil for both.
type Claims struct {
	UserID        string     `json:"user_id"`
	Role          string     `json:"role"`
	TokenType     string     `json:"typ,omitempty"`
	OrganizerID   *uuid.UUID `json:"organizer_id,omitempty"`
	ParticipantID *uuid.UUID `json:"participant_id,omitempty"`
	jwt.RegisteredClaims
}

// Config holds JWT configuration
type Config struct {
	SecretKey     string
	TokenExpiry   time.Duration
	RefreshExpiry time.Duration
}

// Identity names the subject: user id, role, and the linked profile ids
// (nil for admins without a profile).
type Identity struct {
	UserID        string
	Role          string
	OrganizerID   *uuid.UUID
	ParticipantID *uuid.UUID
}

// GenerateToken generates a new access token
func GenerateToken(cfg Config, id Identity) (string, error) {
	return generate(cfg, id, TokenTypeAccess, cfg.TokenExpiry)
}

// GenerateRefreshToken generates a refresh token that lives RefreshExpiry
// (falls back to TokenExpiry when RefreshExpiry is not set).
func GenerateRefreshToken(cfg Config, id Identity) (string, error) {
	expiry := cfg.RefreshExpiry
	if expiry == 0 {
		expiry = cfg.TokenExpiry
	}
	return generate(cfg, id, TokenTypeRefresh, expiry)
}

func generate(cfg Config, id Identity, tokenType string, expiry time.Duration) (string, error) {
	claims := Claims{
		UserID:        id.UserID,
		Role:          id.Role,
		TokenType:     tokenType,
		OrganizerID:   id.OrganizerID,
		ParticipantID: id.ParticipantID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "kinetix-api",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.SecretKey))
}

// ValidateToken validates a JWT token
func ValidateToken(cfg Config, tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(cfg.SecretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// ValidateAccessToken validates a token and rejects refresh tokens. Tokens
// issued before the "typ" claim existed carry no type and are treated as
// access tokens.
func ValidateAccessToken(cfg Config, tokenString string) (*Claims, error) {
	claims, err := ValidateToken(cfg, tokenString)
	if err != nil {
		return nil, err
	}
	if claims.TokenType == TokenTypeRefresh {
		return nil, errors.New("refresh token cannot be used as access token")
	}
	return claims, nil
}

// ValidateRefreshToken validates a token and accepts only refresh tokens.
func ValidateRefreshToken(cfg Config, tokenString string) (*Claims, error) {
	claims, err := ValidateToken(cfg, tokenString)
	if err != nil {
		return nil, err
	}
	if claims.TokenType != TokenTypeRefresh {
		return nil, errors.New("not a refresh token")
	}
	return claims, nil
}
