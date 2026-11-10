// Package mp is a thin MercadoPago REST client used by the OAuth flow and
// the payouts cron job. All endpoints are placeholders that read MP_CLIENT_ID
// and MP_CLIENT_SECRET from config — the user must register a Marketplace
// app at https://www.mercadopago.com/developers/panel before this works.
package mp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"kinetix-api/internal/config"
)

// Client wraps the MP REST API.
type Client struct {
	cfg     config.MercadoPagoConfig
	http    *http.Client
	apiBase string // MP REST base URL; overridden in tests
}

const defaultAPIBase = "https://api.mercadopago.com"

// NewClient creates a new MP client from the app config.
func NewClient(cfg config.MercadoPagoConfig) *Client {
	return &Client{
		cfg:     cfg,
		http:    &http.Client{Timeout: 15 * time.Second},
		apiBase: defaultAPIBase,
	}
}

// OAuthAuthorizeURL returns the URL the organizer should be redirected to
// in order to grant the platform access to their MP account.
func (c *Client) OAuthAuthorizeURL(state string) string {
	q := url.Values{}
	q.Set("client_id", c.cfg.ClientID)
	q.Set("response_type", "code")
	q.Set("platform_id", "mp")
	q.Set("redirect_uri", c.cfg.RedirectURI)
	q.Set("state", state)
	return "https://auth.mercadopago.com.mx/authorization?" + q.Encode()
}

// OAuthTokenResponse is the body MP returns after the organizer authorizes
// the app on the redirect.
type OAuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	UserID       int64  `json:"user_id"`
	Scope        string `json:"scope"`
	PublicKey    string `json:"public_key"`
	LiveMode     bool   `json:"live_mode"`
}

// ExchangeCodeForToken trades the authorization code for an access_token + refresh_token.
func (c *Client) ExchangeCodeForToken(code string) (*OAuthTokenResponse, error) {
	if c.cfg.ClientID == "" || c.cfg.ClientSecret == "" {
		return nil, fmt.Errorf("MP_CLIENT_ID / MP_CLIENT_SECRET not configured")
	}
	body := url.Values{}
	body.Set("grant_type", "authorization_code")
	body.Set("client_id", c.cfg.ClientID)
	body.Set("client_secret", c.cfg.ClientSecret)
	body.Set("code", code)
	body.Set("redirect_uri", c.cfg.RedirectURI)

	req, _ := http.NewRequest(http.MethodPost, c.apiBase+"/oauth/token", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(resp.Body)
		return nil, fmt.Errorf("token exchange HTTP %d: %s", resp.StatusCode, buf.String())
	}
	var out OAuthTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("token decode failed: %w", err)
	}
	return &out, nil
}

// PayoutRequest asks MP to disburse funds to an organizer's linked account.
type PayoutRequest struct {
	Amount            float64 `json:"amount"` // in major units (e.g. MXN)
	ExternalReference string  `json:"external_reference"`
	// In a real impl you'd also pass the collector_id (from mp_user_id) and
	// payment_instruction_id; here we keep the surface minimal.
}

// CreatePayoutResponse is what MP returns from /v1/mp/payouts.
type CreatePayoutResponse struct {
	ID                int64  `json:"id"`
	Status            string `json:"status"`
	ExternalReference string `json:"external_reference"`
}

// CreatePayout requests a disbursement to the organizer. Requires the
// organizer's stored access_token (from oauth flow).
func (c *Client) CreatePayout(organizerAccessToken string, req PayoutRequest) (*CreatePayoutResponse, error) {
	if organizerAccessToken == "" {
		return nil, fmt.Errorf("organizer access_token required")
	}
	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequest(http.MethodPost, c.apiBase+"/v1/mp/payouts", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+organizerAccessToken)
	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("payout request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(resp.Body)
		return nil, fmt.Errorf("payout HTTP %d: %s", resp.StatusCode, buf.String())
	}
	var out CreatePayoutResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("payout decode failed: %w", err)
	}
	return &out, nil
}
