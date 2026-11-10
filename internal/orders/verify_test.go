package orders

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"kinetix-api/internal/mp"
)

// fakeVerifier implements the payment confirmation the webhook needs.
type fakeVerifier struct {
	result *mp.PaymentConfirmation
	err    error
	calls  int
}

func (f *fakeVerifier) GetPayment(id string) (*mp.PaymentConfirmation, error) {
	f.calls++
	return f.result, f.err
}

func TestProcessWebhook_RejectsUnapprovedPayment(t *testing.T) {
	order := pendingOrder()
	repo := &fakeOrderRepo{order: order, organizerID: uuid.New()}
	payouts := &fakePayoutRepo{commissionPct: 10}
	svc := &Service{repo: repo, adminRepo: payouts, logPayoutError: func(o Order, err error) {}}

	got, err := svc.ProcessWebhook(context.Background(), "mp-1", &mp.PaymentConfirmation{ID: 12345, Status: "rejected"})
	if err != nil {
		t.Fatalf("unapproved payment must not fail the webhook: %v", err)
	}
	if got.Status != "pending" {
		t.Errorf("status = %q, want still pending", got.Status)
	}
	if len(repo.statusUpdates) != 0 {
		t.Errorf("status updates = %v, want none", repo.statusUpdates)
	}
	if len(payouts.created) != 0 {
		t.Errorf("payouts created = %d, want 0", len(payouts.created))
	}
}

func TestProcessWebhook_RequiresConfirmation(t *testing.T) {
	order := pendingOrder()
	repo := &fakeOrderRepo{order: order, organizerID: uuid.New()}
	svc := &Service{repo: repo}

	if _, err := svc.ProcessWebhook(context.Background(), "mp-1", nil); err == nil {
		t.Fatal("nil confirmation must be rejected before trusting the notification")
	}
	if len(repo.statusUpdates) != 0 {
		t.Errorf("no writes without a confirmation, got %v", repo.statusUpdates)
	}
}

// Handler-level: the verifier runs before any write; its errors map to 502
// and unapproved payments answer 200 without touching the order (the nil
// repo service would panic otherwise, proving nothing ran).
func TestPaymentWebhook_UsesVerifier(t *testing.T) {
	secret := "mp-secret"

	tests := []struct {
		name        string
		verifier    *fakeVerifier
		wantStatus  int
		wantContain string
	}{
		{
			"verification failure -> 502",
			&fakeVerifier{err: fmt.Errorf("payment HTTP 500")},
			http.StatusServiceUnavailable,
			"payment verification failed",
		},
		{
			"unapproved -> 200 untouched",
			&fakeVerifier{result: &mp.PaymentConfirmation{ID: 7, Status: "rejected"}},
			http.StatusOK,
			"not approved",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(nil, nil, nil, nil)
			h := NewHandler(svc, secret, tt.verifier)

			body := mpPayload("pay-1")
			req := httptest.NewRequest(http.MethodPost, "/orders/webhook", strings.NewReader(string(body)))
			req.Header.Set("Content-Type", "application/json")
			sig := webhookSignature(secret, "pay-1", "req-1", "1718625911")
			req.Header.Set("x-signature", "ts=1718625911,v1="+sig)
			req.Header.Set("x-request-id", "req-1")

			rr := httptest.NewRecorder()
			h.PaymentWebhook(rr, req)

			if rr.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body=%s)", rr.Code, tt.wantStatus, rr.Body.String())
			}
			if !strings.Contains(rr.Body.String(), tt.wantContain) {
				t.Errorf("body = %s, want contains %q", rr.Body.String(), tt.wantContain)
			}
			if tt.wantStatus == http.StatusOK && strings.Contains(rr.Body.String(), `"paid"`) {
				t.Errorf("unapproved payment must not be marked paid: %s", rr.Body.String())
			}
		})
	}
}
