package mp

import (
	"io"
	"net/http"
	"testing"
)

func TestGetPayment_SendsBearerAndDecodes(t *testing.T) {
	var gotAuth string
	var gotPath string
	var gotMethod string
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		gotMethod = r.Method
		_, _ = io.WriteString(w, `{"id":123,"status":"approved","status_detail":"accredited","external_reference":"reg-1"}`)
	})
	c.cfg.AccessToken = "platform-token"

	p, err := c.GetPayment("123")
	if err != nil {
		t.Fatalf("GetPayment: %v", err)
	}
	if gotAuth != "Bearer platform-token" || gotMethod != http.MethodGet || gotPath != "/v1/payments/123" {
		t.Errorf("request = %s %s auth=%q", gotMethod, gotPath, gotAuth)
	}
	if p.Status != "approved" || p.ID != 123 || p.ExternalReference == "" {
		t.Errorf("payment = %+v", p)
	}
}

func TestGetPayment_HTTPErrorIncludesStatus(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	c.cfg.AccessToken = "platform-token"
	if _, err := c.GetPayment("1"); err == nil {
		t.Fatal("expected error for HTTP 404")
	}
}

func TestGetPayment_RequiresPlatformToken(t *testing.T) {
	called := false
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) { called = true })
	if _, err := c.GetPayment("1"); err == nil {
		t.Fatal("expected error without platform token")
	}
	if called {
		t.Error("no request must be sent without token")
	}
}
