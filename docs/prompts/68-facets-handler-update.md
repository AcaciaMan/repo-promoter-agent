# Prompt: Faceted Search — Update Search Handler for Facets and Filters

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. This is **Phase 2, prompt 3 of 5** for faceted search and filtering.

**Prerequisites**: Prompts 66–67 are complete. `internal/store/store.go` now has:
- `SearchResult` struct: `{ Results []Promotion, Facets map[string][]Facet }`
- `Facet` struct: `{ Value string, Count int }`
- `SearchOptions` struct: `{ Tags []string, Channel string, MinStars int }`
- `Search(ctx, query, limit, opts SearchOptions) (SearchResult, error)`
- `List(ctx, limit, opts SearchOptions) (SearchResult, error)`
- `applyFilters()` helper adds `fq` params for tags, channel, and min stars

The project **does not compile** because `internal/handler/search.go` still uses the old `Search`/`List` signatures that return `([]Promotion, error)`.

## Current `internal/handler/search.go`

```go
package handler

import (
    "encoding/json"
    "log"
    "net/http"
    "strconv"

    "repo-promoter-agent/internal/store"
)

type SearchHandler struct {
    store *store.Store
}

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
```

## Your task

Update `internal/handler/search.go` to:
1. Parse new filter query parameters from the request.
2. Build a `SearchOptions` and pass it to `Search`/`List`.
3. Return facet data alongside results in the API response.

## Requirements

### 1. Parse filter query parameters

Extract these from the request URL query string:

| Param | Type | Example | Notes |
|---|---|---|---|
| `q` | string | `?q=CLI+tool` | Existing — full-text query |
| `limit` | int | `?limit=50` | Existing — max results |
| `tag` | string (repeatable) | `?tag=go&tag=cli` | NEW — filter by tag(s), AND logic |
| `channel` | string | `?channel=twitter` | NEW — filter by target_channel |
| `min_stars` | int | `?min_stars=100` | NEW — minimum stars filter |

For `tag`, use `r.URL.Query()["tag"]` to get all values as a `[]string`.

For `channel`, use `r.URL.Query().Get("channel")`.

For `min_stars`, parse with `strconv.Atoi` and ignore if invalid or <= 0.

### 2. Build `SearchOptions`

```go
opts := store.SearchOptions{
    Tags:     r.URL.Query()["tag"],
    Channel:  r.URL.Query().Get("channel"),
    MinStars: minStars, // parsed from min_stars param
}
```

### 3. Update `Search`/`List` calls

Pass `opts` as the last argument:

```go
if q == "" {
    sr, err = h.store.List(r.Context(), limit, opts)
} else {
    sr, err = h.store.Search(r.Context(), q, limit, opts)
}
```

Where `sr` is of type `store.SearchResult`.

### 4. Update response struct

Change `searchResponse` to include facets:

```go
type searchResponse struct {
    Results []store.Promotion        `json:"results"`
    Count   int                      `json:"count"`
    Facets  map[string][]store.Facet `json:"facets,omitempty"`
}
```

### 5. Update response encoding

```go
w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(searchResponse{
    Results: sr.Results,
    Count:   len(sr.Results),
    Facets:  sr.Facets,
})
```

### 6. Full updated `ServeHTTP` method

The complete method should look like this after the changes:

```go
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

    var minStars int
    if ms := r.URL.Query().Get("min_stars"); ms != "" {
        if parsed, err := strconv.Atoi(ms); err == nil && parsed > 0 {
            minStars = parsed
        }
    }

    opts := store.SearchOptions{
        Tags:     r.URL.Query()["tag"],
        Channel:  r.URL.Query().Get("channel"),
        MinStars: minStars,
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
        Results: sr.Results,
        Count:   len(sr.Results),
        Facets:  sr.Facets,
    })
}
```

## Verification

After applying the changes:

1. Run `go build ./...` — must compile without errors.
2. Run `go test ./internal/store/... -tags integration` — existing store tests must pass (update any test calls to `Search`/`List` that now require `SearchOptions{}` as the last argument).
3. Quick manual check: if Solr and the app are running,
   ```
   curl "http://localhost:8080/api/search"
   ```
   should return JSON with `"facets"` alongside `"results"`.

## Files to modify

- `internal/handler/search.go` — update `searchResponse`, parse new query params, pass `SearchOptions`
- `internal/store/store_test.go` — update calls to `Search` and `List` to pass `store.SearchOptions{}` as last argument (add the zero-value struct to fix compile errors)

## Files NOT to modify

- `internal/store/store.go` — already done in prompts 66–67
- `static/index.html` — will be updated in prompt 69
