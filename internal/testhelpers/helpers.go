// Package testhelpers provides shared test helpers used across integration tests.
// These tests are intended to run against a live API instance running on
// http://localhost:8085 (the docker-compose default).
package testhelpers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"
)

// BaseURL is the base URL the integration tests hit.
func BaseURL() string {
	if v := os.Getenv("KINETIX_API_URL"); v != "" {
		return v
	}
	return "http://localhost:8085"
}

// HTTPClient returns a client with sensible timeouts for integration tests.
func HTTPClient() *http.Client {
	return &http.Client{Timeout: 10 * time.Second}
}

// tokenCache memoizes login responses so we don't exceed the 10/min IP-based rate limit.
var (
	tokenMu   sync.Mutex
	tokenCache = map[string]string{}
)

// Login attempts to log in with the provided credentials. Returns the access token on success.
// Returns empty string and fails the test if the login fails. Calls within the same test run
// are memoized per (email, password) pair to avoid hammering the auth endpoint.
func Login(t *testing.T, email, password string) string {
	t.Helper()
	tokenMu.Lock()
	cacheKey := email + "|" + password
	if tok, ok := tokenCache[cacheKey]; ok {
		tokenMu.Unlock()
		return tok
	}
	tokenMu.Unlock()

	body := map[string]string{"email": email, "password": password}
	var resp struct {
		AccessToken string `json:"access_token"`
	}
	status, _ := JSONRequest(t, http.MethodPost, "/api/v1/auth/login", body, nil, &resp)
	if status != http.StatusOK {
		t.Fatalf("login failed (status=%d): %s/%s", status, email, password)
	}
	if resp.AccessToken == "" {
		t.Fatalf("login response missing access_token for %s", email)
	}

	tokenMu.Lock()
	tokenCache[cacheKey] = resp.AccessToken
	tokenMu.Unlock()
	return resp.AccessToken
}

// JSONRequest issues a JSON request and decodes the response into out.
func JSONRequest(t *testing.T, method, path string, body any, headers map[string]string, out any) (int, http.Header) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, BaseURL()+path, reader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := HTTPClient().Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			t.Fatalf("decode response: %v", err)
		}
	}
	return resp.StatusCode, resp.Header
}

// AuthHeader returns an Authorization header for the given token.
func AuthHeader(token string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + token}
}

// GetJSON issues an authenticated GET request and returns the status code + decodes into out.
func GetJSON(t *testing.T, path, key string, out any) int {
	t.Helper()
	status, _ := JSONRequest(t, http.MethodGet, path, nil, AuthHeader(key), out)
	return status
}

// PostJSON issues an authenticated POST request and returns the status code.
func PostJSON(t *testing.T, path, key string, body, out any) int {
	t.Helper()
	status, _ := JSONRequest(t, http.MethodPost, path, body, AuthHeader(key), out)
	return status
}

// PutJSON issues an authenticated PUT request and returns the status code.
func PutJSON(t *testing.T, path, key string, body, out any) int {
	t.Helper()
	status, _ := JSONRequest(t, http.MethodPut, path, body, AuthHeader(key), out)
	return status
}

// Delete issues an authenticated DELETE request and returns the status code.
func Delete(t *testing.T, path, key string) int {
	t.Helper()
	status, _ := JSONRequest(t, http.MethodDelete, path, nil, AuthHeader(key), nil)
	return status
}