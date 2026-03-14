package analytics

import (
	"sort"
	"strings"
	"sync"
)

// PopularQuery represents a search query and how many times it was used.
type PopularQuery struct {
	Query string `json:"query"`
	Count int    `json:"count"`
}

// Tracker records search queries and provides popularity stats.
type Tracker struct {
	mu     sync.Mutex
	counts map[string]int
}

// NewTracker creates a new analytics Tracker.
func NewTracker() *Tracker {
	return &Tracker{counts: make(map[string]int)}
}

// Record increments the count for a search query.
// Empty or blank queries are ignored.
func (t *Tracker) Record(query string) {
	q := strings.TrimSpace(strings.ToLower(query))
	if q == "" {
		return
	}
	t.mu.Lock()
	t.counts[q]++
	t.mu.Unlock()
}

// Popular returns the top N most-searched queries, sorted by count descending.
func (t *Tracker) Popular(limit int) []PopularQuery {
	if limit <= 0 {
		limit = 10
	}
	t.mu.Lock()
	items := make([]PopularQuery, 0, len(t.counts))
	for q, c := range t.counts {
		items = append(items, PopularQuery{Query: q, Count: c})
	}
	t.mu.Unlock()

	sort.Slice(items, func(i, j int) bool {
		if items[i].Count != items[j].Count {
			return items[i].Count > items[j].Count
		}
		return items[i].Query < items[j].Query
	})

	if len(items) > limit {
		items = items[:limit]
	}
	return items
}
