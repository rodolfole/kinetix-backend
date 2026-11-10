package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestGenerateToken(t *testing.T) {
	cfg := Config{
		SecretKey:   "test-secret-key",
		TokenExpiry: 1 * time.Hour,
	}

	token, err := GenerateToken(cfg, Identity{UserID: "user-123", Role: "organizer"})
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
}

func TestValidateToken_Roundtrip(t *testing.T) {
	cfg := Config{
		SecretKey:   "test-secret-key",
		TokenExpiry: 1 * time.Hour,
	}

	token, err := GenerateToken(cfg, Identity{UserID: "user-123", Role: "participant"})
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}

	claims, err := ValidateToken(cfg, token)
	if err != nil {
		t.Fatalf("ValidateToken returned error: %v", err)
	}

	if claims.UserID != "user-123" {
		t.Errorf("expected UserID=user-123, got %q", claims.UserID)
	}
	if claims.Role != "participant" {
		t.Errorf("expected Role=participant, got %q", claims.Role)
	}
}

func TestValidateToken_InvalidSecret(t *testing.T) {
	cfg := Config{SecretKey: "test-secret-key", TokenExpiry: 1 * time.Hour}
	cfg2 := Config{SecretKey: "different-secret", TokenExpiry: 1 * time.Hour}

	token, err := GenerateToken(cfg, Identity{UserID: "user-123", Role: "organizer"})
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}

	if _, err := ValidateToken(cfg2, token); err == nil {
		t.Fatal("expected validation with different secret to fail")
	}
}

func TestValidateToken_Expired(t *testing.T) {
	cfg := Config{
		SecretKey:   "test-secret-key",
		TokenExpiry: -1 * time.Second, // already expired
	}

	token, err := GenerateToken(cfg, Identity{UserID: "user-123", Role: "organizer"})
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}

	if _, err := ValidateToken(cfg, token); err == nil {
		t.Fatal("expected expired token validation to fail")
	}
}
func TestGenerateRefreshToken_UsesRefreshTypeAndExpiry(t *testing.T) {
	cfg := Config{SecretKey: "test-secret-key", TokenExpiry: time.Hour, RefreshExpiry: 7 * 24 * time.Hour}

	token, err := GenerateRefreshToken(cfg, Identity{UserID: "user-123", Role: "participant"})
	if err != nil {
		t.Fatalf("GenerateRefreshToken returned error: %v", err)
	}
	claims, err := ValidateRefreshToken(cfg, token)
	if err != nil {
		t.Fatalf("ValidateRefreshToken returned error: %v", err)
	}
	if claims.TokenType != TokenTypeRefresh {
		t.Errorf("expected typ=%q, got %q", TokenTypeRefresh, claims.TokenType)
	}
	if ttl := time.Until(claims.ExpiresAt.Time); ttl < 6*24*time.Hour {
		t.Errorf("expected refresh expiry ~7d, got %v", ttl)
	}
}

func TestValidateAccessToken_RejectsRefreshToken(t *testing.T) {
	cfg := Config{SecretKey: "test-secret-key", TokenExpiry: time.Hour, RefreshExpiry: 24 * time.Hour}

	refresh, err := GenerateRefreshToken(cfg, Identity{UserID: "user-123", Role: "organizer"})
	if err != nil {
		t.Fatalf("GenerateRefreshToken returned error: %v", err)
	}
	if _, err := ValidateAccessToken(cfg, refresh); err == nil {
		t.Fatal("expected refresh token to be rejected as access token")
	}
}

func TestValidateAccessToken_AcceptsAccessToken(t *testing.T) {
	cfg := Config{SecretKey: "test-secret-key", TokenExpiry: time.Hour}

	access, err := GenerateToken(cfg, Identity{UserID: "user-123", Role: "organizer"})
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}
	claims, err := ValidateAccessToken(cfg, access)
	if err != nil {
		t.Fatalf("expected access token to be accepted: %v", err)
	}
	if claims.TokenType != TokenTypeAccess || claims.Role != "organizer" {
		t.Errorf("unexpected claims: typ=%q role=%q", claims.TokenType, claims.Role)
	}
}

func TestValidateRefreshToken_RejectsAccessToken(t *testing.T) {
	cfg := Config{SecretKey: "test-secret-key", TokenExpiry: time.Hour}

	access, err := GenerateToken(cfg, Identity{UserID: "user-123", Role: "organizer"})
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}
	if _, err := ValidateRefreshToken(cfg, access); err == nil {
		t.Fatal("expected access token to be rejected as refresh token")
	}
}

func TestServiceRefreshToken_RejectsAccessToken(t *testing.T) {
	cfg := Config{SecretKey: "test-secret-key", TokenExpiry: time.Hour}
	svc := NewService(nil, cfg) // the store must not be reached

	access, err := GenerateToken(cfg, Identity{UserID: "user-123", Role: "organizer"})
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}
	if _, _, err := svc.RefreshToken(context.Background(), access); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestGenerateToken_CarriesProfileClaims(t *testing.T) {
	cfg := Config{SecretKey: "test-secret", TokenExpiry: time.Hour}
	orgID := uuid.New()
	partID := uuid.New()

	token, err := GenerateToken(cfg, Identity{UserID: "u1", Role: "organizer", OrganizerID: &orgID, ParticipantID: &partID})
	if err != nil {
		t.Fatal(err)
	}
	claims, err := ValidateToken(cfg, token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.OrganizerID == nil || *claims.OrganizerID != orgID {
		t.Errorf("organizer claim = %v, want %s", claims.OrganizerID, orgID)
	}
	if claims.ParticipantID == nil || *claims.ParticipantID != partID {
		t.Errorf("participant claim = %v, want %s", claims.ParticipantID, partID)
	}

	empty, err := GenerateToken(cfg, Identity{UserID: "u2", Role: "admin"})
	if err != nil {
		t.Fatal(err)
	}
	claims2, err := ValidateToken(cfg, empty)
	if err != nil {
		t.Fatal(err)
	}
	if claims2.OrganizerID != nil || claims2.ParticipantID != nil {
		t.Errorf("admin identity must carry nil profile claims: %+v", claims2)
	}
}
