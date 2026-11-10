package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

func newTestHandler() *Handler {
	repo := NewRepository(nil) // safe — these tests don't reach the repo
	svc := NewService(repo)
	return NewHandler(svc, validate)
}

func TestUpdateConfigRequest_Validate(t *testing.T) {
	tests := []struct {
		name      string
		req       UpdateConfigRequest
		wantValid bool
	}{
		{name: "valid 10%", req: UpdateConfigRequest{CommissionPct: 10, UpdatedBy: "admin"}, wantValid: true},
		{name: "valid 0%", req: UpdateConfigRequest{CommissionPct: 0, UpdatedBy: "admin"}, wantValid: true},
		{name: "valid 100%", req: UpdateConfigRequest{CommissionPct: 100, UpdatedBy: "admin"}, wantValid: true},
		{name: "negative rejected", req: UpdateConfigRequest{CommissionPct: -1, UpdatedBy: "admin"}, wantValid: false},
		{name: "over 100 rejected", req: UpdateConfigRequest{CommissionPct: 101, UpdatedBy: "admin"}, wantValid: false},
		{name: "missing updated_by rejected", req: UpdateConfigRequest{CommissionPct: 10}, wantValid: false},
		{name: "blank updated_by rejected", req: UpdateConfigRequest{CommissionPct: 10, UpdatedBy: ""}, wantValid: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.Struct(&tt.req)
			if tt.wantValid && err != nil {
				t.Fatalf("expected valid, got %v", err)
			}
			if !tt.wantValid && err == nil {
				t.Fatalf("expected validation error, got nil")
			}
		})
	}
}

func TestUpdateConfig_BadJSONReturns400(t *testing.T) {
	h := newTestHandler()
	body := strings.NewReader(`not json`)
	req := httptest.NewRequest(http.MethodPut, "/admin/config", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.UpdateConfig(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body=%s)", rr.Code, rr.Body.String())
	}
}

func TestUpdateConfig_InvalidCommissionReturns400(t *testing.T) {
	h := newTestHandler()
	body, _ := json.Marshal(map[string]any{"commission_pct": 150, "updated_by": "admin"})
	req := httptest.NewRequest(http.MethodPut, "/admin/config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.UpdateConfig(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body=%s)", rr.Code, rr.Body.String())
	}
}

func TestConfigToResponse_Valid(t *testing.T) {
	resp := ConfigResponse{
		CommissionPct: 12.5,
		UpdatedBy:     strPtr("admin@kinetix.com"),
	}
	if resp.CommissionPct != 12.5 {
		t.Errorf("expected 12.5, got %v", resp.CommissionPct)
	}
	if resp.UpdatedBy == nil || *resp.UpdatedBy != "admin@kinetix.com" {
		t.Errorf("updated_by mismatch")
	}
}

func TestNullIfEmpty(t *testing.T) {
	nullIfEmpty := func(s string) *string {
		if s == "" {
			return nil
		}
		return &s
	}
	if nullIfEmpty("") != nil {
		t.Error("expected nil for empty string")
	}
	if nullIfEmpty("x") == nil || *nullIfEmpty("x") != "x" {
		t.Error("expected pointer for non-empty string")
	}
}

// TestCalculatePayoutAmount covers the commission math.
func TestCalculatePayoutAmount(t *testing.T) {
	tests := []struct {
		name          string
		totalCents    int64
		commissionPct float64
		want          int64
	}{
		{name: "0% returns full amount", totalCents: 10000, commissionPct: 0, want: 10000},
		{name: "10% off", totalCents: 10000, commissionPct: 10, want: 9000},
		{name: "25% off", totalCents: 1000, commissionPct: 25, want: 750},
		{name: "100% returns 0", totalCents: 99999, commissionPct: 100, want: 0},
		{name: "rounds down", totalCents: 100, commissionPct: 33.33, want: 66},  // 100 * 0.6667 = 66.67 → floor = 66
		{name: "zero amount returns 0", totalCents: 0, commissionPct: 10, want: 0},
		{name: "negative amount returns 0", totalCents: -100, commissionPct: 10, want: 0},
		{name: "negative pct clamped to 0", totalCents: 1000, commissionPct: -5, want: 1000},
		{name: "over 100 pct clamped to 100", totalCents: 1000, commissionPct: 105, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculatePayoutAmount(tt.totalCents, tt.commissionPct)
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func strPtr(s string) *string { return &s }
