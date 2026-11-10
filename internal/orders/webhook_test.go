package orders

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func computeHMAC(secret, body string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(body))
	return hex.EncodeToString(mac.Sum(nil))
}

func TestValidateMercadoPagoWebhook_EmptySecretFailsClosed(t *testing.T) {
	// A missing secret must never disable verification.
	if err := ValidateMercadoPagoWebhook("", "body", computeHMAC("", "body")); !errors.Is(err, ErrWebhookNotConfigured) {
		t.Fatalf("expected ErrWebhookNotConfigured when secret is empty, got %v", err)
	}
}

func TestValidateMercadoPagoWebhook_MissingSignature(t *testing.T) {
	err := ValidateMercadoPagoWebhook("secret", "body", "")
	if err != ErrMissingSignature {
		t.Fatalf("expected ErrMissingSignature, got %v", err)
	}
}

func TestValidateMercadoPagoWebhook_WrongSignature(t *testing.T) {
	err := ValidateMercadoPagoWebhook("secret", "body", "deadbeef")
	if err != ErrInvalidSignature {
		t.Fatalf("expected ErrInvalidSignature, got %v", err)
	}
}

func TestValidateMercadoPagoWebhook_ValidSignature(t *testing.T) {
	secret := "mp-access-token"
	body := `{"type":"payment","data":{"id":"123"}}`
	sig := computeHMAC(secret, body)
	if err := ValidateMercadoPagoWebhook(secret, body, sig); err != nil {
		t.Fatalf("expected nil for valid signature, got %v", err)
	}
}

func TestValidateMercadoPagoWebhook_BodyMismatch(t *testing.T) {
	secret := "mp-access-token"
	sig := computeHMAC(secret, "body-A")
	// Verify against a different body
	err := ValidateMercadoPagoWebhook(secret, "body-B", sig)
	if err != ErrInvalidSignature {
		t.Fatalf("expected ErrInvalidSignature for tampered body, got %v", err)
	}
}

func TestExtractSignatureFromRequest(t *testing.T) {
	tests := []struct {
		name    string
		headers map[string]string
		want    string
	}{
		{
			name:    "X-Merchant-Token",
			headers: map[string]string{"X-Merchant-Token": "abc"},
			want:    "abc",
		},
		{
			name:    "x-merchant-token fallback",
			headers: map[string]string{"x-merchant-token": "low"},
			want:    "low",
		},
		{
			name:    "X-Signature fallback",
			headers: map[string]string{"X-Signature": "sig"},
			want:    "sig",
		},
		{
			name:    "x-signature fallback",
			headers: map[string]string{"x-signature": "lowsig"},
			want:    "lowsig",
		},
		{
			name:    "X-Merchant-Token wins over X-Signature",
			headers: map[string]string{"X-Merchant-Token": "primary", "X-Signature": "secondary"},
			want:    "primary",
		},
		{
			name:    "none returns empty",
			headers: map[string]string{},
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/", nil)
			for k, v := range tt.headers {
				r.Header.Set(k, v)
			}
			got := ExtractSignatureFromRequest(r)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateWebhookRequest_Delegates(t *testing.T) {
	secret := "mp-access-token"
	body := `{"hello":"world"}`
	sig := computeHMAC(secret, body)

	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.Header.Set("X-Merchant-Token", sig)

	if err := ValidateWebhookRequest(secret, r, []byte(body)); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestValidateWebhookRequest_MissingSig(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	err := ValidateWebhookRequest("secret", r, []byte("body"))
	if err != ErrMissingSignature {
		t.Fatalf("expected ErrMissingSignature, got %v", err)
	}
}

// Valid MP IPN-shaped payload
func mpPayload(paymentID string) []byte {
	type data struct {
		ID string `json:"id"`
	}
	type payload struct {
		Action        string `json:"action"`
		ApplicationID string `json:"application_id"`
		Data          data   `json:"data"`
		Type          string `json:"type"`
	}
	b, _ := json.Marshal(payload{
		Action:        "payment.created",
		ApplicationID: "12345",
		Data:          data{ID: paymentID},
		Type:          "payment",
	})
	return b
}

func TestPaymentWebhook_MissingSignatureReturns401(t *testing.T) {
	svc := NewService(nil, nil, nil)
	h := NewHandler(svc, "mp-secret")

	body := mpPayload("pay-123")
	req := httptest.NewRequest(http.MethodPost, "/orders/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.PaymentWebhook(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for missing signature, got %d", rr.Code)
	}
}

func TestPaymentWebhook_WrongSignatureReturns403(t *testing.T) {
	svc := NewService(nil, nil, nil)
	h := NewHandler(svc, "mp-secret")

	body := mpPayload("pay-123")
	req := httptest.NewRequest(http.MethodPost, "/orders/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Merchant-Token", "wrong-signature")
	rr := httptest.NewRecorder()

	h.PaymentWebhook(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 for wrong signature, got %d", rr.Code)
	}
}

func TestPaymentWebhook_ValidSignatureEmptyPaymentIDReturns400(t *testing.T) {
	secret := "mp-access-token"
	svc := NewService(nil, nil, nil)
	h := NewHandler(svc, secret)

	// Send payload with empty data.id
	body := []byte(`{"action":"payment.created","data":{"id":""},"type":"payment"}`)
	req := httptest.NewRequest(http.MethodPost, "/orders/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Merchant-Token", computeHMAC(secret, string(body)))
	rr := httptest.NewRecorder()

	h.PaymentWebhook(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty payment id, got %d (body=%s)", rr.Code, rr.Body.String())
	}
}

func TestPaymentWebhook_ValidSignatureInvalidJSONReturns400(t *testing.T) {
	secret := "mp-access-token"
	svc := NewService(nil, nil, nil)
	h := NewHandler(svc, secret)

	body := []byte(`not json`)
	req := httptest.NewRequest(http.MethodPost, "/orders/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Merchant-Token", computeHMAC(secret, string(body)))
	rr := httptest.NewRecorder()

	h.PaymentWebhook(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", rr.Code)
	}
}
func TestPaymentWebhook_EmptySecretReturns503(t *testing.T) {
	// nil service: reaching ProcessWebhook would panic, proving the request is rejected first.
	h := NewHandler(nil, "")

	body := mpPayload("pay-123")
	req := httptest.NewRequest(http.MethodPost, "/orders/webhook", bytes.NewReader(body))
	req.Header.Set("X-Signature", computeHMAC("", string(body)))
	rr := httptest.NewRecorder()

	h.PaymentWebhook(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d (body=%s)", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	if got := strings.TrimSpace(rr.Body.String()); got != `{"error":"webhook verification not configured"}` {
		t.Errorf("body = %s", got)
	}
}
