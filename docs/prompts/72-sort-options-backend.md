# Prompt: Sort Options — Backend Sort Parameter Support

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. This is **Phase 3, prompt 2 of 4** for search highlighting & sort options.

**Prerequisite**: Prompt 71 is complete — `internal/store/store.go` now has:
- `SearchResult` with a `Highlights map[string]map[string]string` field
- `Search` requests Solr highlighting (`hl=true`, `hl.fl=...`, `hl.method=unified`)
- `parseHighlights` helper extracts and joins Solr highlight snippets

## Problem

Search always sorts by `score desc` and List always sorts by `created_at desc`. Users cannot choose to sort by popularity (stars), recent traffic (views), or newest first when searching. The sort order is hardcoded in the store methods.

## Your task

1. Add a `Sort` field to `SearchOptions`.
2. Apply the sort field in both `Search` and `List`.
3. Update the handler to parse a `sort` query parameter and pass it through.
4. Return highlights in the handler response.

## Requirements

### 1. Add `Sort` to `SearchOptions` in `internal/store/store.go`

Update the existing struct:

```go
type SearchOptions struct {
    Tags     []string // Filter by exact tag values (AND logic: all must match)
    Channel  string   // Filter by target_channel exact value
    MinStars int      // Filter to docs with stars >= this value (0 = no filter)
    Sort     string   // Sort order: "relevance", "newest", "stars", "views" (empty = default)
}
```

### 2. Add `solrSort` helper in `internal/store/store.go`

Create a helper that maps user-facing sort names to Solr sort expressions:

```go
// solrSort returns the Solr sort clause for a SearchOptions.Sort value.
// For Search queries (hasScore=true), default is "score desc".
// For List queries (hasScore=false), default is "created_at desc".
func solrSort(sort string, hasScore bool) string {
    switch sort {
    case "newest":
        return "created_at desc"
    case "stars":
        return "stars desc"
    case "views":
        return "views_14d_total desc"
    default:
        if hasScore {
            return "score desc"
        }
        return "created_at desc"
    }
}
```

**Design decisions:**
- `"relevance"` (or empty/unknown) → `score desc` for Search, `created_at desc` for List (List has no scores).
- `"newest"` → `created_at desc` — most recently generated content first.
- `"stars"` → `stars desc` — most popular repos first.
- `"views"` → `views_14d_total desc` — most viewed repos first (recent traction signal).

### 3. Use `solrSort` in `Search`

Replace the hardcoded `"sort": {"score desc"}` with:

```go
"sort": {solrSort(opts.Sort, true)},
```

### 4. Use `solrSort` in `List`

Replace the hardcoded `"sort": {"created_at desc"}` with:

```go
"sort": {solrSort(opts.Sort, false)},
```

### 5. Update handler in `internal/handler/search.go`

#### 5a. Parse `sort` query parameter

Add this after the `minStars` parsing block:

```go
sortBy := r.URL.Query().Get("sort")
```

#### 5b. Include in `SearchOptions`

```go
opts := store.SearchOptions{
    Tags:     r.URL.Query()["tag"],
    Channel:  r.URL.Query().Get("channel"),
    MinStars: minStars,
    Sort:     sortBy,
}
```

#### 5c. Update `searchResponse` to include highlights

Add a `Highlights` field to the response struct:

```go
type searchResponse struct {
    Results    []store.Promotion                  `json:"results"`
    Count      int                                `json:"count"`
    Facets     map[string][]store.Facet            `json:"facets,omitempty"`
    Highlights map[string]map[string]string        `json:"highlights,omitempty"`
}
```

#### 5d. Return highlights in the response

```go
json.NewEncoder(w).Encode(searchResponse{
    Results:    sr.Results,
    Count:      len(sr.Results),
    Facets:     sr.Facets,
    Highlights: sr.Highlights,
})
```

## Verification

After applying the changes:

1. Run `go build ./...` — must compile without errors.
2. Run `go test ./internal/store/... -tags integration` — all existing tests must pass.
3. Quick manual check with Solr and the app running:
   ```
   curl "http://localhost:8080/api/search?q=Go&sort=stars"
   ```
   Should return results sorted by `stars` descending.
   ```
   curl "http://localhost:8080/api/search?sort=newest"
   ```
   Should return all results sorted by `created_at` descending (same as browse, but explicit).

## Files to modify

- `internal/store/store.go` — add `Sort` to `SearchOptions`, add `solrSort` helper, use it in `Search` and `List`
- `internal/handler/search.go` — parse `sort` param, pass in `SearchOptions`, add `Highlights` to response

## Files NOT to modify

- `static/index.html` — will be updated in prompt 73
- `internal/store/store_test.go` — no changes needed
