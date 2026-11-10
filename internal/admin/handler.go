package admin

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	db "kinetix-api/internal/adapters/postgresql/sqlc"
	kinetixjson "kinetix-api/internal/json"
	"kinetix-api/internal/mp"
)

var _ = time.Time{}
var _ = pgtype.Numeric{}

var ErrInvalidCommission = errors.New("commission_pct must be between 0 and 100")

// Handler exposes /admin/* HTTP routes.
type Handler struct {
	service     *Service
	validate    *validator.Validate
	mpClient    *mp.Client
	oauthSecret string
}

func NewHandler(s *Service, v *validator.Validate) *Handler {
	return &Handler{service: s, validate: v}
}

// NewHandlerWithMP builds a handler with MP OAuth + payout support wired in.
// oauthSecret signs the OAuth `state` token (JWT_SECRET in production).
func NewHandlerWithMP(s *Service, v *validator.Validate, mpClient *mp.Client, oauthSecret string) *Handler {
	return &Handler{service: s, validate: v, mpClient: mpClient, oauthSecret: oauthSecret}
}

func (h *Handler) validateStruct(s any) error {
	return h.validate.Struct(s)
}

// Routes returns an http.Handler for /admin/*. Mount via chi.Mux.Handle("/admin/*", h.Routes()).
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /config", h.GetConfig)
	mux.HandleFunc("PUT /config", h.UpdateConfig)
	mux.HandleFunc("GET /payouts", h.ListPayouts)
	return mux
}

// GetConfig handles GET /admin/config (exported for chi router registration).
func (h *Handler) GetConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.service.GetConfig(r.Context())
	if err != nil {
		kinetixjson.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	kinetixjson.Write(w, http.StatusOK, map[string]any{
		"data": configToResponse(cfg),
	})
}

// UpdateConfig handles PUT /admin/config (exported for chi router registration).
func (h *Handler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	var req UpdateConfigRequest
	if err := kinetixjson.Read(r, &req); err != nil {
		kinetixjson.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.validateStruct(&req); err != nil {
		kinetixjson.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	cfg, err := h.service.UpdateConfig(r.Context(), req.CommissionPct, req.UpdatedBy)
	if err != nil {
		if errors.Is(err, ErrInvalidCommission) {
			kinetixjson.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		kinetixjson.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	kinetixjson.Write(w, http.StatusOK, map[string]any{
		"data": configToResponse(cfg),
	})
}

// ListPayouts handles GET /admin/payouts (exported for chi router registration).
func (h *Handler) ListPayouts(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	payouts, err := h.service.ListPayouts(r.Context(), status, 100)
	if err != nil {
		kinetixjson.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	resp := PayoutsListResponse{
		Data:  make([]PayoutResponse, 0, len(payouts)),
		Total: len(payouts),
	}
	for _, p := range payouts {
		resp.Data = append(resp.Data, payoutToResponse(p))
	}
	kinetixjson.Write(w, http.StatusOK, resp)
}

// ConnectMP returns the URL the admin should redirect the organizer to in
// order to authorize the platform's MP Marketplace app on their account.
func (h *Handler) ConnectMP(w http.ResponseWriter, r *http.Request) {
	if h.mpClient == nil {
		kinetixjson.WriteError(w, http.StatusServiceUnavailable, "MP client not configured")
		return
	}
	// /admin/organizers/:id/connect-mp — extract :id from URL path.
	// chi stores URL params in r.PathValue; for plain net/http we parse.
	parts := splitPath(r.URL.Path, "/admin/organizers/", "/connect-mp")
	if len(parts) == 0 {
		kinetixjson.WriteError(w, http.StatusBadRequest, "missing organizer id")
		return
	}
	orgID, err := uuid.Parse(parts[0])
	if err != nil {
		kinetixjson.WriteError(w, http.StatusBadRequest, "invalid organizer id")
		return
	}
	state := SignOAuthState(h.oauthSecret, orgID) // PLAN-16 T13: signed state
	url := h.mpClient.OAuthAuthorizeURL(state)
	kinetixjson.Write(w, http.StatusOK, ConnectMPResponse{
		AuthorizeURL: url,
		State:        state,
	})
}

// OAuthCallback handles MP's redirect back to us after organizer authorizes.
func (h *Handler) OAuthCallback(w http.ResponseWriter, r *http.Request) {
	if h.mpClient == nil {
		kinetixjson.WriteError(w, http.StatusServiceUnavailable, "MP client not configured")
		return
	}
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" {
		kinetixjson.WriteError(w, http.StatusBadRequest, "missing code")
		return
	}
	orgID, err := VerifyOAuthState(h.oauthSecret, state)
	if err != nil {
		kinetixjson.WriteError(w, http.StatusBadRequest, "invalid state")
		return
	}
	tok, err := h.mpClient.ExchangeCodeForToken(code)
	if err != nil {
		kinetixjson.WriteError(w, http.StatusBadGateway, err.Error())
		return
	}
	_, err = h.service.UpsertMPCredentials(r.Context(), orgID, tok)
	if err != nil {
		kinetixjson.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	kinetixjson.Write(w, http.StatusOK, map[string]any{
		"organizer_id": orgID,
		"mp_user_id":   tok.UserID,
		"expires_in":   tok.ExpiresIn,
	})
}

// SendPayout triggers MP to disburse a single pending payout. Marks the row as
// sent (or failed) based on the API response.
func (h *Handler) SendPayout(w http.ResponseWriter, r *http.Request) {
	if h.mpClient == nil {
		kinetixjson.WriteError(w, http.StatusServiceUnavailable, "MP client not configured")
		return
	}
	parts := splitPath(r.URL.Path, "/admin/payouts/", "/send")
	if len(parts) == 0 {
		kinetixjson.WriteError(w, http.StatusBadRequest, "missing payout id")
		return
	}
	payoutID, err := uuid.Parse(parts[0])
	if err != nil {
		kinetixjson.WriteError(w, http.StatusBadRequest, "invalid payout id")
		return
	}
	if err := h.service.SendPayout(r.Context(), h.mpClient, payoutID); err != nil {
		kinetixjson.WriteError(w, http.StatusBadGateway, err.Error())
		return
	}
	kinetixjson.Write(w, http.StatusOK, map[string]any{"payout_id": payoutID, "status": "sent"})
}

// splitPath extracts the segment between prefix and suffix from a path.
// Returns the captured segment(s).
func splitPath(path, prefix, suffix string) []string {
	if !hasPrefix(path, prefix) {
		return nil
	}
	rest := path[len(prefix):]
	if suffix != "" {
		if !hasSuffix(rest, suffix) {
			return nil
		}
		rest = rest[:len(rest)-len(suffix)]
	}
	return []string{rest}
}

func hasPrefix(s, p string) bool   { return len(s) >= len(p) && s[:len(p)] == p }
func hasSuffix(s, suf string) bool { return len(s) >= len(suf) && s[len(s)-len(suf):] == suf }

func configToResponse(cfg db.PlatformConfig) ConfigResponse {
	resp := ConfigResponse{
		CommissionPct: numericToFloat(cfg.CommissionPct),
		UpdatedAt:     cfg.UpdatedAt,
	}
	if cfg.UpdatedBy.Valid {
		s := cfg.UpdatedBy.String
		resp.UpdatedBy = &s
	}
	return resp
}

func payoutToResponse(p db.OrganizerPayout) PayoutResponse {
	resp := PayoutResponse{
		ID:            p.ID,
		OrganizerID:   p.OrganizerID,
		OrderID:       p.OrderID,
		AmountCents:   p.AmountCents,
		Currency:      p.Currency,
		CommissionPct: numericToFloat(p.CommissionPct),
		Status:        p.Status,
		CreatedAt:     p.CreatedAt,
	}
	if p.MpPayoutID.Valid {
		s := p.MpPayoutID.String
		resp.MPPayoutID = &s
	}
	if p.FailureReason.Valid {
		s := p.FailureReason.String
		resp.FailureReason = &s
	}
	if p.SentAt.Valid {
		t := p.SentAt.Time
		resp.SentAt = &t
	}
	return resp
}

func numericToFloat(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	f, _ := n.Float64Value()
	return f.Float64
}
