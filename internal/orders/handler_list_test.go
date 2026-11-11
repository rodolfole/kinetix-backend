package orders

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type listRecorderRepo struct {
	OrderRepository
	calls []string
}

func (f *listRecorderRepo) ListByEvent(ctx context.Context, id uuid.UUID) ([]Order, error) {
	f.calls = append(f.calls, "event:"+id.String())
	return []Order{}, nil
}

func (f *listRecorderRepo) ListByParticipant(ctx context.Context, id uuid.UUID) ([]Order, error) {
	f.calls = append(f.calls, "participant:"+id.String())
	return []Order{}, nil
}

func TestList_QueryIDs(t *testing.T) {
	valid := uuid.New().String()
	tests := []struct {
		name       string
		query      string
		wantStatus int
		wantCalls  []string
	}{
		{"invalid event_id rejected", "event_id=not-a-uuid", http.StatusBadRequest, nil},
		{"invalid participant_id rejected", "participant_id=123", http.StatusBadRequest, nil},
		{"valid event_id listed", "event_id=" + valid, http.StatusOK, []string{"event:" + valid}},
		{"valid participant_id listed", "participant_id=" + valid, http.StatusOK, []string{"participant:" + valid}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &listRecorderRepo{}
			h := NewHandler(&Service{repo: repo}, "secret", nil)
			r := chi.NewRouter()
			h.Routes(r)

			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/?"+tt.query, nil))

			if rr.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body=%s)", rr.Code, tt.wantStatus, rr.Body.String())
			}
			if strings.Join(repo.calls, ",") != strings.Join(tt.wantCalls, ",") {
				t.Errorf("service calls = %v, want %v", repo.calls, tt.wantCalls)
			}
			if tt.wantStatus == http.StatusBadRequest {
				if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
					t.Errorf("Content-Type = %q, want application/json", ct)
				}
			}
		})
	}
}
