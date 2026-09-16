package auth

import (
	"testing"
	"time"
)

func TestGenerateToken(t *testing.T) {
	cfg := Config{
		SecretKey:   "test-secret-key",
		TokenExpiry: 1 * time.Hour,
	}

	token, err := GenerateToken(cfg, "user-123", "organizer")
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

	token, err := GenerateToken(cfg, "user-123", "participant")
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

	token, err := GenerateToken(cfg, "user-123", "organizer")
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

	token, err := GenerateToken(cfg, "user-123", "organizer")
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}

	if _, err := ValidateToken(cfg, token); err == nil {
		t.Fatal("expected expired token validation to fail")
	}
}

func TestExtractTokenFromHeader(t *testing.T) {
	tests := []struct {
		name       string
		header     string
		wantToken  string
		wantErrMsg string
	}{
		{
			name:      "valid bearer token",
			header:    "Bearer abc.def.ghi",
			wantToken: "abc.def.ghi",
		},
		{
			name:       "missing header",
			header:     "",
			wantErrMsg: "authorization header required",
		},
		{
			name:       "wrong prefix",
			header:     "Basic abc.def.ghi",
			wantErrMsg: "invalid authorization header format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newAuthRequest(tt.header)
			tok, err := ExtractTokenFromHeader(r)
			if tt.wantErrMsg != "" {
				if err == nil || err.Error() != tt.wantErrMsg {
					t.Fatalf("want err %q, got %v", tt.wantErrMsg, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tok != tt.wantToken {
				t.Errorf("want token %q, got %q", tt.wantToken, tok)
			}
		})
	}
}