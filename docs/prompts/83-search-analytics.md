# Prompt: Search Analytics — Query Logging & Popular Searches

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. The app stores AI-generated promotional content in **Apache Solr 10** and exposes full-text search.

This is **Phase 5, prompt 4 of 5**. Prompts 80–82 added spellcheck, result grouping, and More Like This. This prompt adds lightweight search analytics — logging queries and showing "Popular searches" on the search page.

## Current state

### Route registration in `cmd/server/main.go`

```go
mux.Handle("/api/generate", limiter.Middleware("generate")(handler.NewGenerateHandler(agentClient, githubClient, st, analysisClient)))
mux.Handle("/api/search", limiter.Middleware("search")(handler.NewSearchHandler(st)))
mux.Handle("/api/suggest", limiter.Middleware("search")(handler.NewSuggestHandler(st)))
mux.Handle("/api/mlt", limiter.Middleware("search")(handler.NewMLTHandler(st)))
mux.Handle("/", noCacheHandler(http.FileServerFS(static.Files)))
```

### `SearchHandler` in `internal/handler/search.go`

```go
type SearchHandler struct {
    store *store.Store
}

func NewSearchHandler(st *store.Store) *SearchHandler {
    return &SearchHandler{store: st}
}
```

### Frontend — search panel HTML

```html
<div class="tab-content" id="panel-search">
<section class="search-section">
    <h2>Search &amp; Browse</h2>
    <div style="display:flex;gap:.5rem">
        <div class="autocomplete-wrap" style="flex:1">
            <input type="text" id="search-q" ... autocomplete="off">
            <div id="suggest-list" class="autocomplete-list" role="listbox"></div>
        </div>
        <button id="search-btn" ...>Search</button>
    </div>
    <div id="search-status" class="status"></div>
    <div id="search-error" class="error"></div>
    <div id="search-collation"></div>
    <div id="active-filters" ...></div>
    <div id="search-facets" ...></div>
    <div id="search-sort" ...>...</div>
    <div id="search-results"></div>
</section>
</div>
```

## Your task

1. Create a simple in-memory search analytics tracker in a new package
2. Integrate it into the search handler to record queries
3. Add a `GET /api/analytics/popular` endpoint
4. Show "Popular searches" as clickable chips on the search page

## Requirements

### 1. Create `internal/analytics/tracker.go`

This is a simple in-memory query counter. No persistence needed — it resets on server restart. Good enough for a hackathon demo.

```go
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
```

### 2. Update `SearchHandler` to accept and use the tracker

In `internal/handler/search.go`:

Add the import for analytics:

```go
import (
    "encoding/json"
    "log"
    "net/http"
    "strconv"

    "repo-promoter-agent/internal/analytics"
    "repo-promoter-agent/internal/store"
)
```

Update the struct and constructor:

Replace:

```go
type SearchHandler struct {
    store *store.Store
}

func NewSearchHandler(st *store.Store) *SearchHandler {
    return &SearchHandler{store: st}
}
```

With:

```go
type SearchHandler struct {
    store   *store.Store
    tracker *analytics.Tracker
}

func NewSearchHandler(st *store.Store, tracker *analytics.Tracker) *SearchHandler {
    return &SearchHandler{store: st, tracker: tracker}
}
```

In `ServeHTTP`, after getting the query `q` and before building opts, record the query:

Add after the line `q := r.URL.Query().Get("q")`:

```go
if q != "" && h.tracker != nil {
    h.tracker.Record(q)
}
```

### 3. Create `internal/handler/analytics.go`

```go
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
```

### 4. Wire in `cmd/server/main.go`

Add the analytics import and create the tracker. Then update `NewSearchHandler` call and add the popular endpoint.

Add import:

```go
"repo-promoter-agent/internal/analytics"
```

Create the tracker before the routes section (after the rate limiter setup):

```go
// Create search analytics tracker.
tracker := analytics.NewTracker()
```

Update the SearchHandler creation and add the popular route:

Replace:

```go
mux.Handle("/api/search", limiter.Middleware("search")(handler.NewSearchHandler(st)))
```

With:

```go
mux.Handle("/api/search", limiter.Middleware("search")(handler.NewSearchHandler(st, tracker)))
mux.Handle("/api/analytics/popular", limiter.Middleware("search")(handler.NewPopularHandler(tracker)))
```

### 5. Add "Popular Searches" UI in `static/index.html`

#### 5a. Add CSS (in `<style>`, after the `.mlt-card .mlt-meta` rule):

```css
.popular-section{margin:.75rem 0}
.popular-section h4{font-size:.85rem;color:#555;margin:0 0 .4rem;font-weight:600}
.popular-chip{display:inline-block;background:#f3e8ff;color:#7c3aed;border:1px solid #ddd6fe;border-radius:12px;padding:.2rem .6rem;font-size:.8rem;cursor:pointer;margin:.15rem .2rem;transition:background .15s}
.popular-chip:hover{background:#ddd6fe}
.popular-chip .pop-count{font-size:.7rem;color:#a78bfa;margin-left:.25rem}
```

#### 5b. Add HTML container

Add `<div id="popular-searches"></div>` right after `<div id="search-sort" ...>...</div>` and before `<div id="search-results"></div>`:

```html
<div id="popular-searches"></div>
<div id="search-results"></div>
```

#### 5c. Add JavaScript to load and render popular searches

Add in the `<script>` block, before the final `doSearch();` at the end:

```js
async function loadPopular(){
    try{
        const res=await fetch('/api/analytics/popular?limit=8');
        if(!res.ok)return;
        const data=await res.json();
        if(!data||!data.length){document.getElementById('popular-searches').innerHTML='';return}
        let h='<div class="popular-section"><h4>🔥 Popular Searches</h4>';
        data.forEach(p=>{
            h+='<span class="popular-chip" onclick="searchQ.value=\''+esc(p.query).replace(/'/g,"\\'")+'\';doSearch()">'+esc(p.query)+'<span class="pop-count">'+p.count+'</span></span>';
        });
        h+='</div>';
        document.getElementById('popular-searches').innerHTML=h;
    }catch(e){}
}
```

#### 5d. Call `loadPopular()` after each search and on page load

In `doSearch()`, at the very end of the `finally` block, add:

```js
loadPopular();
```

Also add `loadPopular();` right before or after the final `doSearch();` call at the bottom:

```js
doSearch();
loadPopular();
```

This ensures popular searches update after every search and are visible on initial page load.

## What NOT to do

- Do **NOT** persist analytics to disk or Solr — in-memory is sufficient for a hackathon
- Do **NOT** record empty/browse queries — only actual text searches
- Do **NOT** log user IPs or any PII — just the query text and count
- Do **NOT** modify `store.go` — analytics is a separate concern

## Verification

```powershell
go build ./...
```

After starting the server:

```powershell
# Do a few searches
Invoke-RestMethod -Uri "http://localhost:8080/api/search?q=container" -Method GET | Out-Null
Invoke-RestMethod -Uri "http://localhost:8080/api/search?q=container" -Method GET | Out-Null
Invoke-RestMethod -Uri "http://localhost:8080/api/search?q=python" -Method GET | Out-Null
Invoke-RestMethod -Uri "http://localhost:8080/api/search?q=container" -Method GET | Out-Null

# Check popular
$r = Invoke-RestMethod -Uri "http://localhost:8080/api/analytics/popular" -Method GET
$r | ForEach-Object { "$($_.query): $($_.count)" }
# Expected: container: 3, python: 1
```

Frontend:
1. Browse/search a few times with different queries
2. "🔥 Popular Searches" section should appear with clickable chips
3. Click a popular search chip → should fill the search box and execute the search
4. Counts should update after each search
