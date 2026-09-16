package tags

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	kinetixjson "kinetix-api/internal/json"
)

// Handler handles HTTP requests for event tags
type Handler struct {
	repo *Repository
}

// NewHandler creates a new tags handler
func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// ListByEvent handles GET /events/{eventId}/tags
func (h *Handler) ListByEvent(w http.ResponseWriter, r *http.Request) {
	eventID, err := uuid.Parse(chi.URLParam(r, "eventId"))
	if err != nil {
		kinetixjson.Write(w, http.StatusBadRequest, map[string]string{"error": "invalid event ID"})
		return
	}

	tags, err := h.repo.ListByEvent(r.Context(), eventID)
	if err != nil {
		kinetixjson.Write(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	kinetixjson.Write(w, http.StatusOK, map[string]interface{}{
		"message": "Tags retrieved successfully",
		"data":    tags,
	})
}

// createTagRequest represents the request body for creating a tag
type createTagRequest struct {
	EventID uuid.UUID `json:"event_id"`
	Name    string    `json:"name"`
}

// Create handles POST /events/tags
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		kinetixjson.Write(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Name == "" {
		kinetixjson.Write(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}

	if req.EventID == uuid.Nil {
		kinetixjson.Write(w, http.StatusBadRequest, map[string]string{"error": "event_id is required"})
		return
	}

	tag, err := h.repo.Create(r.Context(), req.EventID, req.Name)
	if err != nil {
		kinetixjson.Write(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	kinetixjson.Write(w, http.StatusCreated, map[string]interface{}{
		"message": "Tag created successfully",
		"data":    tag,
	})
}

// Delete handles DELETE /events/tags/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		kinetixjson.Write(w, http.StatusBadRequest, map[string]string{"error": "invalid tag ID"})
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		kinetixjson.Write(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	kinetixjson.Write(w, http.StatusOK, map[string]string{"message": "Tag deleted successfully"})
}
