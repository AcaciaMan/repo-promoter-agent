package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"repo-promoter-agent/internal/store"
)

// MLTHandler handles GET /api/mlt requests for "More Like This" results.
type MLTHandler struct {
	store *store.Store
}

// NewMLTHandler creates a MLTHandler with the given store.
func NewMLTHandler(st *store.Store) *MLTHandler {
	return &MLTHandler{store: st}
}

func (h *MLTHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	docID := r.URL.Query().Get("id")
	if docID == "" {
		writeError(w, http.StatusBadRequest, "id parameter required")
		return
	}

	limit := 5
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	results, err := h.store.MoreLikeThis(r.Context(), docID, limit)
	if err != nil {
		log.Printf("MLT failed: %v", err)
		writeError(w, http.StatusInternalServerError, "mlt failed")
		return
	}
	if results == nil {
		results = []store.Promotion{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Results []store.Promotion `json:"results"`
		Count   int               `json:"count"`
	}{
		Results: results,
		Count:   len(results),
	})
}
