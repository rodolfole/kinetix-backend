package json

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWrite_SetsContentTypeAndStatus(t *testing.T) {
	rr := httptest.NewRecorder()
	Write(rr, http.StatusCreated, map[string]string{"hello": "world"})

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}
	if !strings.Contains(rr.Body.String(), `"hello":"world"`) {
		t.Errorf("expected body to contain hello:world, got %s", rr.Body.String())
	}
}

func TestWriteError_EncodesErrorMessage(t *testing.T) {
	rr := httptest.NewRecorder()
	WriteError(rr, http.StatusBadRequest, "missing email")

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}
	if !strings.Contains(rr.Body.String(), `"error":"missing email"`) {
		t.Errorf("expected body to contain error message, got %s", rr.Body.String())
	}
}

type testPayload struct {
	Name string `json:"name"`
}

func TestRead_DecodesValidPayload(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"kinetix"}`))
	var p testPayload
	if err := Read(req, &p); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if p.Name != "kinetix" {
		t.Errorf("expected Name kinetix, got %q", p.Name)
	}
}

func TestRead_RejectsUnknownFields(t *testing.T) {
	// Read always rejects unknown fields — this is the default behavior.
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"x","extra":"y"}`))
	var p testPayload
	if err := Read(req, &p); err == nil {
		t.Error("expected error for unknown field, got nil")
	}
}

func TestRead_RejectsExtraJSONAfterPayload(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"x"}{}`))
	var p testPayload
	err := Read(req, &p)
	if err == nil {
		t.Error("expected error for extra JSON after payload, got nil")
	}
	if _, ok := err.(*json.SyntaxError); !ok {
		t.Errorf("expected *json.SyntaxError, got %T: %v", err, err)
	}
}

func TestReadAllowUnknown_AcceptsUnknownFields(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"x","extra":"y"}`))
	var p testPayload
	if err := ReadAllowUnknown(req, &p); err != nil {
		t.Errorf("expected no error with unknown fields allowed, got %v", err)
	}
	if p.Name != "x" {
		t.Errorf("expected Name x, got %q", p.Name)
	}
}
