package orders

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
)

var (
	ErrInvalidSignature = errors.New("invalid webhook signature")
	ErrMissingSignature = errors.New("missing webhook signature header")
)

// ValidateMercadoPagoWebhook validates the MercadoPago webhook signature
// MercadoPago sends the signature in the X-Merchant-Token header as HMAC-SHA256
func ValidateMercadoPagoWebhook(webhookSecret, body, signatureHeader string) error {
	if webhookSecret == "" {
		// If no secret configured, skip validation (not recommended for production)
		return nil
	}

	if signatureHeader == "" {
		return ErrMissingSignature
	}

	// Compute expected signature
	mac := hmac.New(sha256.New, []byte(webhookSecret))
	mac.Write([]byte(body))
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	// Compare signatures (constant-time to prevent timing attacks)
	if !hmac.Equal([]byte(signatureHeader), []byte(expectedSignature)) {
		return ErrInvalidSignature
	}

	return nil
}

// ExtractSignatureFromRequest extracts the signature from request headers
func ExtractSignatureFromRequest(r *http.Request) string {
	// MercadoPago typically sends the signature in this header
	sig := r.Header.Get("X-Merchant-Token")
	if sig != "" {
		return sig
	}

	// Fallback: try lowercase
	sig = r.Header.Get("x-merchant-token")
	if sig != "" {
		return sig
	}

	// Some MercadoPago integrations use this header
	sig = r.Header.Get("X-Signature")
	if sig != "" {
		return sig
	}

	return r.Header.Get("x-signature")
}

// ValidateWebhookRequest validates a webhook request with its signature
func ValidateWebhookRequest(webhookSecret string, r *http.Request, body []byte) error {
	signature := ExtractSignatureFromRequest(r)
	return ValidateMercadoPagoWebhook(webhookSecret, string(body), signature)
}