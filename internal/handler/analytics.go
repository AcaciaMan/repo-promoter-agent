package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"repo-promoter-agent/internal/analytics"
)

// PopularHandler handles GET /api/analytics/popular requests.
type PopularHandler struct {
	tracker *analytics.Tracker
}

// NewPopularHandler creates a PopularHandler with the given tracker.
func NewPopularHandler(tracker *analytics.Tracker) *PopularHandler {
	return &PopularHandler{tracker: tracker}
}

func (h *PopularHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if limit > 50 {
		limit = 50
	}

	popular := h.tracker.Popular(limit)
	if popular == nil {
		popular = []analytics.PopularQuery{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(popular)
}
