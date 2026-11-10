package admin

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"

	"github.com/google/uuid"
)

// StateV1 es el estado OAuth firmado (PLAN-16 T13). El UUID crudo del
// organizador nunca viaja como estado: el callback exige la MAC.
const stateMagic = "mp-state-v1:"

var ErrInvalidState = errors.New("invalid oauth state")

// SignOAuthState returns the signed, URL-safe state token for organizer.
func SignOAuthState(secret string, organizerID uuid.UUID) string {
	payload := stateMagic + organizerID.String()
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	sig := mac.Sum(nil)
	return base64.RawURLEncoding.EncodeToString(sig) + "." + base64.RawURLEncoding.EncodeToString([]byte(payload))
}

// VerifyOAuthState accepts a state produced by SignOAuthState and returns
// the organizer it binds to. It is constant-time on the MAC comparison.
func VerifyOAuthState(secret, state string) (uuid.UUID, error) {
	parts := strings.SplitN(state, ".", 2)
	if len(parts) != 2 {
		return uuid.Nil, ErrInvalidState
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return uuid.Nil, ErrInvalidState
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return uuid.Nil, ErrInvalidState
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return uuid.Nil, ErrInvalidState
	}
	if !strings.HasPrefix(string(payload), stateMagic) {
		return uuid.Nil, ErrInvalidState
	}
	return uuid.Parse(strings.TrimPrefix(string(payload), stateMagic))
}
