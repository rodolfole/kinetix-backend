package mp

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"kinetix-api/internal/config"
)

func testClient(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	c := NewClient(config.MercadoPagoConfig{ClientID: "cid", ClientSecret: "csecret", RedirectURI: "http://localhost/cb"})
	c.apiBase = srv.URL
	return c
}

func TestNewClient_DefaultsToProductionAPI(t *testing.T) {
	if got := NewClient(config.MercadoPagoConfig{}).apiBase; got != "https://api.mercadopago.com" {
		t.Fatalf("apiBase = %q", got)
	}
}

func TestOAuthAuthorizeURL(t *testing.T) {
	c := NewClient(config.MercadoPagoConfig{ClientID: "cid", RedirectURI: "http://localhost/cb"})
	u, err := url.Parse(c.OAuthAuthorizeURL("state-1"))
	if err != nil {
		t.Fatal(err)
	}
	if u.Host != "auth.mercadopago.com.mx" || u.Path != "/authorization" {
		t.Errorf("url = %s", u)
	}
	q := u.Query()
	for k, want := range map[string]string{"client_id": "cid", "response_type": "code", "platform_id": "mp", "redirect_uri": "http://localhost/cb", "state": "state-1"} {
		if q.Get(k) != want {
			t.Errorf("%s = %q, want %q", k, q.Get(k), want)
		}
	}
}

func TestExchangeCodeForToken_RequiresCredentials(t *testing.T) {
	called := false
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) { called = true })
	c.cfg.ClientSecret = ""
	if _, err := c.ExchangeCodeForToken("code"); err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("err = %v, want not configured", err)
	}
	if called {
		t.Error("no request must be sent without credentials")
	}
}

func TestExchangeCodeForToken_PostsFormAndDecodes(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/oauth/token" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
			t.Errorf("Content-Type = %q", ct)
		}
		_ = r.ParseForm()
		for k, want := range map[string]string{"grant_type": "authorization_code", "client_id": "cid", "client_secret": "csecret", "code": "abc", "redirect_uri": "http://localhost/cb"} {
			if r.PostForm.Get(k) != want {
				t.Errorf("form %s = %q, want %q", k, r.PostForm.Get(k), want)
			}
		}
		_, _ = io.WriteString(w, `{"access_token":"at","refresh_token":"rt","user_id":42,"expires_in":3600}`)
	})
	tok, err := c.ExchangeCodeForToken("abc")
	if err != nil {
		t.Fatal(err)
	}
	if tok.AccessToken != "at" || tok.RefreshToken != "rt" || tok.UserID != 42 || tok.ExpiresIn != 3600 {
		t.Errorf("token = %+v", tok)
	}
}

func TestExchangeCodeForToken_HTTPErrorIncludesStatus(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, "invalid_grant")
	})
	if _, err := c.ExchangeCodeForToken("abc"); err == nil || !strings.Contains(err.Error(), "HTTP 400") || !strings.Contains(err.Error(), "invalid_grant") {
		t.Fatalf("err = %v", err)
	}
}

func TestCreatePayout_RequiresOrganizerToken(t *testing.T) {
	called := false
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) { called = true })
	if _, err := c.CreatePayout("", PayoutRequest{Amount: 10}); err == nil {
		t.Fatal("expected error without organizer token")
	}
	if called {
		t.Error("no request must be sent without a token")
	}
}

func TestCreatePayout_SendsBearerAndJSON(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/mp/payouts" || r.Header.Get("Authorization") != "Bearer org-token" {
			t.Errorf("path=%s auth=%q", r.URL.Path, r.Header.Get("Authorization"))
		}
		var got PayoutRequest
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil || got.Amount != 450.5 || got.ExternalReference != "payout-1" {
			t.Errorf("body = %+v err=%v", got, err)
		}
		_, _ = io.WriteString(w, `{"id":7,"status":"approved","external_reference":"payout-1"}`)
	})
	out, err := c.CreatePayout("org-token", PayoutRequest{Amount: 450.5, ExternalReference: "payout-1"})
	if err != nil {
		t.Fatal(err)
	}
	if out.ID != 7 || out.Status != "approved" {
		t.Errorf("response = %+v", out)
	}
}

func TestCreatePayout_HTTPErrorIncludesStatus(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	})
	if _, err := c.CreatePayout("org-token", PayoutRequest{Amount: 1}); err == nil || !strings.Contains(err.Error(), "HTTP 502") {
		t.Fatalf("err = %v", err)
	}
}
