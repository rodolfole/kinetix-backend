package orders

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func computeHMAC(secret, body string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(body))
	return hex.EncodeToString(mac.Sum(nil))
}

func TestValidateMercadoPagoWebhook_EmptySecretFailsClosed(t *testing.T) {
	// A missing secret must never disable verification.
	if err := ValidateMercadoPagoWebhook("", "pay-1", "req-1", "1718625911", []string{"deadbeef"}); !errors.Is(err, ErrWebhookNotConfigured) {
		t.Fatalf("expected ErrWebhookNotConfigured when secret is empty, got %v", err)
	}
}

func TestValidateMercadoPagoWebhook_MissingSignature(t *testing.T) {
	err := ValidateMercadoPagoWebhook("secret", "pay-1", "req-1", "", []string{"deadbeef"})
	if err != ErrMissingSignature {
		t.Fatalf("expected ErrMissingSignature, got %v", err)
	}
}

func TestValidateMercadoPagoWebhook_WrongSignature(t *testing.T) {
	err := ValidateMercadoPagoWebhook("secret", "pay-1", "req-1", "1718625911", []string{"deadbeef"})
	if err != ErrInvalidSignature {
		t.Fatalf("expected ErrInvalidSignature, got %v", err)
	}
}

func TestValidateMercadoPagoWebhook_ValidSignature(t *testing.T) {
	secret := "mp-access-token"
	ts := fmt.Sprintf("%d", time.Now().Unix())
	sig := webhookSignature(secret, "123", "req-1", ts)
	if err := ValidateMercadoPagoWebhook(secret, "123", "req-1", ts, []string{sig}); err != nil {
		t.Fatalf("expected nil for valid signature, got %v", err)
	}
}

func TestValidateMercadoPagoWebhook_BodyMismatch(t *testing.T) {
	secret := "mp-access-token"
	ts := fmt.Sprintf("%d", time.Now().Unix())
	sig := webhookSignature(secret, "pay-A", "req-1", ts)
	// Verify against a different notification id
	err := ValidateMercadoPagoWebhook(secret, "pay-B", "req-1", ts, []string{sig})
	if err != ErrInvalidSignature {
		t.Fatalf("expected ErrInvalidSignature for tampered body, got %v", err)
	}
}

func TestParseXSignature(t *testing.T) {
	tests := []struct {
		name      string
		header    string
		wantTS    string
		wantV1    []string
		wantFound bool
	}{
		{"single v1", "ts=1718625911,v1=abc123", "1718625911", []string{"abc123"}, true},
		{"multiple v1", "ts=1718625911,v1=abc123,v1=def456", "1718625911", []string{"abc123", "def456"}, true},
		{"spaced", "ts= 1718625911 , v1=abc123 ", "1718625911", []string{"abc123"}, true},
		{"no v1", "ts=1718625911", "1718625911", nil, false},
		{"garbage", "garbage", "", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, v1, ok := parseXSignature(tt.header)
			if ok != tt.wantFound || ts != tt.wantTS {
				t.Fatalf("ts=%q ok=%v, want %q/%v", ts, ok, tt.wantTS, tt.wantFound)
			}
			if len(v1) != len(tt.wantV1) {
				t.Fatalf("v1 = %v, want %v", v1, tt.wantV1)
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

	if err := ValidateWebhookRequest(secret, r, []byte(body), "pay-1"); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestValidateWebhookRequest_MissingSig(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	err := ValidateWebhookRequest("secret", r, []byte("body"), "pay-1")
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
