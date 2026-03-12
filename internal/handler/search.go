package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"repo-promoter-agent/internal/store"
)

// SearchHandler handles GET /api/search requests.
type SearchHandler struct {
	store *store.Store
}

// NewSearchHandler creates a SearchHandler with the given store.
func NewSearchHandler(st *store.Store) *SearchHandler {
	return &SearchHandler{store: st}
}

type searchResponse struct {
	Results []store.Promotion `json:"results"`
	Count   int               `json:"count"`
}

func (h *SearchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	q := r.URL.Query().Get("q")
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if limit > 100 {
		limit = 100
	}

	var (
		results []store.Promotion
		err     error
	)
	if q == "" {
		results, err = h.store.List(r.Context(), limit)
	} else {
		results, err = h.store.Search(r.Context(), q, limit)
	}
	if err != nil {
		log.Printf("Search/list failed: %v", err)
		writeError(w, http.StatusInternalServerError, "search failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(searchResponse{
		Results: results,
		Count:   len(results),
	})
}
