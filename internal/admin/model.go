package admin

import (
	"time"

	"github.com/google/uuid"
)

// ConfigResponse is the platform-level config exposed to the admin UI.
type ConfigResponse struct {
	CommissionPct float64   `json:"commission_pct"`
	UpdatedAt     time.Time `json:"updated_at"`
	UpdatedBy     *string   `json:"updated_by,omitempty"`
}

// UpdateConfigRequest is the PUT body for changing the platform commission.
type UpdateConfigRequest struct {
	// Use gte=0 (not "required") so 0% is accepted as a valid zero commission.
	CommissionPct float64 `json:"commission_pct" validate:"gte=0,lte=100"`
	UpdatedBy     string  `json:"updated_by" validate:"required,min=1,max=200"`
}

// PayoutResponse is a single organizer payout row.
type PayoutResponse struct {
	ID            uuid.UUID  `json:"id"`
	OrganizerID   uuid.UUID  `json:"organizer_id"`
	OrderID       uuid.UUID  `json:"order_id"`
	AmountCents   int64      `json:"amount_cents"`
	Currency      string     `json:"currency"`
	CommissionPct float64    `json:"commission_pct"`
	Status        string     `json:"status"`
	MPPayoutID    *string    `json:"mp_payout_id,omitempty"`
	FailureReason *string    `json:"failure_reason,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	SentAt        *time.Time `json:"sent_at,omitempty"`
}

// PayoutsListResponse is the API response shape for listing payouts.
type PayoutsListResponse struct {
	Message string           `json:"message"`
	Data    []PayoutResponse `json:"data"`
	Total   int              `json:"total"`
}

// ConnectMPResponse is what GET /admin/organizers/:id/connect-mp returns —
// the admin frontend redirects the browser to this URL.
type ConnectMPResponse struct {
	AuthorizeURL string `json:"authorize_url"`
	State        string `json:"state"`
}
