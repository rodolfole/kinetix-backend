package orders

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// mpSig builds the real MercadoPago signature manifest:
// HMAC-SHA256(secret, "id:<data.id>;request-id:<x-request-id>;ts:<ts>;") hex.
func mpSig(t *testing.T, secret, dataID, requestID, ts string) string {
	t.Helper()
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(fmt.Sprintf("id:%s;request-id:%s;ts:%s;", dataID, requestID, ts)))
	return hex.EncodeToString(mac.Sum(nil))
}

func mpSignedRequest(t *testing.T, secret, dataID, requestID string) *http.Request {
	t.Helper()
	ts := fmt.Sprintf("%d", time.Now().Unix())
	sig := mpSig(t, secret, dataID, requestID, ts)
	req := httptest.NewRequest(http.MethodPost, "/orders/webhook", nil)
	req.Header.Set("x-signature", fmt.Sprintf("ts=%s,v1=%s", ts, sig))
	req.Header.Set("x-request-id", requestID)
	return req
}

// mpSignedRequestHeaderMismatch signs over signedReqID but declares a
// different header value, so re-computation cannot match.
func mpSignedRequestHeaderMismatch(t *testing.T, secret, dataID, signedReqID string) *http.Request {
	t.Helper()
	req := mpSignedRequest(t, secret, dataID, signedReqID)
	req.Header.Set("x-request-id", "header-req")
	return req
}

func TestMPWebhookSignatureScheme(t *testing.T) {
	secret := "mp-secret"
	const dataID = "pay-1"
	const reqID = "req-1"

	tests := []struct {
		name    string
		req     *http.Request
		dataID  string
		wantErr error
	}{
		{"valid signature", mpSignedRequest(t, secret, dataID, reqID), dataID, nil},
		{"wrong data id", mpSignedRequest(t, secret, "other-pay", reqID), dataID, ErrInvalidSignature},
		{"wrong request id", mpSignedRequestHeaderMismatch(t, secret, dataID, "other-req"), dataID, ErrInvalidSignature},
		{"wrong secret", mpSignedRequest(t, "other-secret", dataID, reqID), dataID, ErrInvalidSignature},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWebhookRequest(secret, tt.req, []byte(`{"data":{"id":"pay-1"}}`), tt.dataID)
			if tt.wantErr != nil && err != tt.wantErr {
				t.Fatalf("want %v, got %v", tt.wantErr, err)
			}
			if tt.wantErr == nil && err != nil {
				t.Fatalf("expected nil, got %v", err)
			}
		})
	}
}

func TestMPWebhookSignature_MalformedHeader(t *testing.T) {
	secret := "mp-secret"
	req := httptest.NewRequest(http.MethodPost, "/orders/webhook", nil)
	req.Header.Set("x-signature", "garbage-without-ts")
	if err := ValidateWebhookRequest(secret, req, []byte("{}"), "pay-1"); err != ErrMissingSignature && err != ErrInvalidSignature {
		t.Fatalf("garbage x-signature must be rejected, got %v", err)
	}
}
