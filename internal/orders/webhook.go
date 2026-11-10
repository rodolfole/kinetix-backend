package orders

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
)

var (
	ErrInvalidSignature = errors.New("invalid webhook signature")
	ErrMissingSignature = errors.New("missing webhook signature header")
	// ErrWebhookNotConfigured is returned when MP_WEBHOOK_SECRET is empty.
	// Verification fails closed: an unconfigured secret never skips it.
	ErrWebhookNotConfigured = errors.New("webhook verification not configured")
)

// signatureManifest builds MercadoPago's canonical HMAC input:
//
//	id:<data.id>;request-id:<x-request-id>;ts:<ts>;
func signatureManifest(dataID, requestID, ts string) string {
	return "id:" + dataID + ";request-id:" + requestID + ";ts:" + ts + ";"
}

// webhookSignature computes the hex HMAC-SHA256 of the manifest.
func webhookSignature(secret, dataID, requestID, ts string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signatureManifest(dataID, requestID, ts)))
	return hex.EncodeToString(mac.Sum(nil))
}

// parseXSignature splits MercadoPago's header into ts and the v1 candidates:
// "ts=1718625911,v1=<hex>[,v1=<hex>...]". ok is false for malformed headers.
func parseXSignature(header string) (ts string, v1 []string, ok bool) {
	found := false
	for _, part := range strings.Split(header, ",") {
		parts := strings.SplitN(part, "=", 2)
		if len(parts) != 2 {
			continue
		}
		switch strings.TrimSpace(parts[0]) {
		case "ts":
			ts = strings.TrimSpace(parts[1])
		case "v1":
			v1 = append(v1, strings.TrimSpace(parts[1]))
			found = true
		}
	}
	return ts, v1, found
}

// ValidateMercadoPagoWebhook verifies the x-signature scheme where the
// header carries `ts` and at least one `v1` HMAC over the manifest.
func ValidateMercadoPagoWebhook(webhookSecret, dataID, requestID, ts string, v1 []string) error {
	if webhookSecret == "" {
		return ErrWebhookNotConfigured
	}
	if requestID == "" || ts == "" || len(v1) == 0 {
		return ErrMissingSignature
	}
	expected := webhookSignature(webhookSecret, dataID, requestID, ts)
	for _, candidate := range v1 {
		if hmac.Equal([]byte(candidate), []byte(expected)) {
			return nil
		}
	}
	return ErrInvalidSignature
}

// extractLegacySignature returns the raw-body HMAC header used by local
// tooling before the MercadoPago x-signature scheme landed.
func extractLegacySignature(r *http.Request) (string, bool) {
	for _, h := range []string{"X-Merchant-Token", "x-merchant-token"} {
		if sig := r.Header.Get(h); sig != "" {
			return sig, true
		}
	}
	return "", false
}

// legacyBodySignature computes the hex HMAC-SHA256 of the raw body.
func legacyBodySignature(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// ValidateWebhookRequest validates the notification against the configured
// secret. MercadoPago's x-signature scheme takes precedence; the legacy
// raw-body HMAC stays accepted for local tooling. An empty secret fails
// closed (ErrWebhookNotConfigured); it never disables verification.
func ValidateWebhookRequest(webhookSecret string, r *http.Request, body []byte, dataID string) error {
	if webhookSecret == "" {
		return ErrWebhookNotConfigured
	}

	hasXSignature := r.Header.Get("x-signature") != "" || r.Header.Get("X-Signature") != ""
	if sig, ok := extractLegacySignature(r); ok && !hasXSignature {
		if hmac.Equal([]byte(sig), []byte(legacyBodySignature(webhookSecret, body))) {
			return nil
		}
		return ErrInvalidSignature
	}

	if !hasXSignature {
		return ErrMissingSignature
	}

	ts, v1, ok := parseXSignature(r.Header.Get("x-signature"))
	if !ok {
		return ErrInvalidSignature
	}
	if v1 == nil {
		return ErrInvalidSignature
	}
	requestID := r.Header.Get("x-request-id")

	sigErr := ValidateMercadoPagoWebhook(webhookSecret, dataID, requestID, ts, v1)
	return sigErr
}
