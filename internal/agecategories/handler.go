package agecategories

import (
	"net/http"

	"kinetix-api/internal/json"
)

// Handler handles HTTP requests for age categories
type Handler struct {
	repo *Repository
}

// NewHandler creates a new age categories handler
func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// List handles GET /age-categories
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	categories, err := h.repo.List(r.Context())
	if err != nil {
		json.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	data := make([]AgeCategoryResponse, len(categories))
	for i, cat := range categories {
		data[i] = toResponse(cat)
	}

	json.Write(w, http.StatusOK, GenericResponse{
		Message: "Age categories retrieved successfully",
		Data:    data,
	})
}
