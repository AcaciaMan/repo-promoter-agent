package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"repo-promoter-agent/internal/store"
)

// SuggestHandler handles GET /api/suggest requests.
type SuggestHandler struct {
	store *store.Store
}

// NewSuggestHandler creates a SuggestHandler with the given store.
func NewSuggestHandler(st *store.Store) *SuggestHandler {
	return &SuggestHandler{store: st}
}

func (h *SuggestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	q := r.URL.Query().Get("q")
	if q == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]store.Suggestion{})
		return
	}

	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	suggestions, err := h.store.Suggest(r.Context(), q, limit)
	if err != nil {
		log.Printf("Suggest failed: %v", err)
		writeError(w, http.StatusInternalServerError, "suggest failed")
		return
	}
	if suggestions == nil {
		suggestions = []store.Suggestion{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(suggestions)
}
