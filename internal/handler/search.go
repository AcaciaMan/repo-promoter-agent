package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"repo-promoter-agent/internal/analytics"
	"repo-promoter-agent/internal/store"
)

// SearchHandler handles GET /api/search requests.
type SearchHandler struct {
	store   *store.Store
	tracker *analytics.Tracker
}

// NewSearchHandler creates a SearchHandler with the given store.
func NewSearchHandler(st *store.Store, tracker *analytics.Tracker) *SearchHandler {
	return &SearchHandler{store: st, tracker: tracker}
}

type searchResponse struct {
	Results    []store.Promotion            `json:"results"`
	Count      int                          `json:"count"`
	Facets     map[string][]store.Facet     `json:"facets,omitempty"`
	Highlights map[string]map[string]string `json:"highlights,omitempty"`
	Collation  string                       `json:"collation,omitempty"`
}

func (h *SearchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	q := r.URL.Query().Get("q")
	if q != "" && h.tracker != nil {
		h.tracker.Record(q)
	}
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if limit > 100 {
		limit = 100
	}

	var minStars int
	if ms := r.URL.Query().Get("min_stars"); ms != "" {
		if parsed, err := strconv.Atoi(ms); err == nil && parsed > 0 {
			minStars = parsed
		}
	}

	sortBy := r.URL.Query().Get("sort")

	opts := store.SearchOptions{
		Tags:     r.URL.Query()["tag"],
		Channel:  r.URL.Query().Get("channel"),
		MinStars: minStars,
		Sort:     sortBy,
	}

	var (
		sr  store.SearchResult
		err error
	)
	if q == "" {
		sr, err = h.store.List(r.Context(), limit, opts)
	} else {
		sr, err = h.store.Search(r.Context(), q, limit, opts)
	}
	if err != nil {
		log.Printf("Search/list failed: %v", err)
		writeError(w, http.StatusInternalServerError, "search failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(searchResponse{
		Results:    sr.Results,
		Count:      len(sr.Results),
		Facets:     sr.Facets,
		Highlights: sr.Highlights,
		Collation:  sr.Collation,
	})
}
