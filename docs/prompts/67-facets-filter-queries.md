# Prompt: Faceted Search — Filter Queries in Store

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. This is **Phase 2, prompt 2 of 5** for faceted search and filtering.

**Prerequisite**: Prompt 66 is complete — `internal/store/store.go` now has:
- `SearchResult` struct with `Results []Promotion` and `Facets map[string][]Facet`
- `Facet` struct with `Value string` and `Count int`
- `Search()` returns `(SearchResult, error)` and requests `facet=true` with `facet.field=tags` and `facet.field=target_channel`
- `List()` returns `(SearchResult, error)` with the same facet params
- `parseFacets()` helper that parses Solr's alternating name/count facet arrays

## Current problem

Users can see facet counts (e.g. "go: 5", "twitter: 3") but cannot **click on a facet to filter results**. The `Search` and `List` methods have no way to accept filter criteria.

## Your task

Add a `SearchOptions` struct and update `Search` and `List` to accept filter parameters, translated into Solr `fq` (filter query) params.

## Requirements

### 1. Add `SearchOptions` struct

Add this struct after the existing `Facet` struct in `internal/store/store.go`:

```go
// SearchOptions holds optional filter parameters for search and list queries.
type SearchOptions struct {
    Tags     []string // Filter by exact tag values (AND logic: all must match)
    Channel  string   // Filter by target_channel exact value
    MinStars int      // Filter to docs with stars >= this value (0 = no filter)
}
```

### 2. Update `Search` signature

Change from:
```go
func (s *Store) Search(ctx context.Context, query string, limit int) (SearchResult, error)
```
to:
```go
func (s *Store) Search(ctx context.Context, query string, limit int, opts SearchOptions) (SearchResult, error)
```

### 3. Update `List` signature

Change from:
```go
func (s *Store) List(ctx context.Context, limit int) (SearchResult, error)
```
to:
```go
func (s *Store) List(ctx context.Context, limit int, opts SearchOptions) (SearchResult, error)
```

### 4. Add `applyFilters` helper

Create a helper that takes `url.Values` and `SearchOptions` and appends `fq` params:

```go
// applyFilters adds Solr filter query (fq) parameters based on SearchOptions.
func applyFilters(params url.Values, opts SearchOptions) {
    for _, tag := range opts.Tags {
        params.Add("fq", fmt.Sprintf("tags:%q", tag))
    }
    if opts.Channel != "" {
        params.Add("fq", fmt.Sprintf("target_channel:%q", opts.Channel))
    }
    if opts.MinStars > 0 {
        params.Add("fq", fmt.Sprintf("stars:[%d TO *]", opts.MinStars))
    }
}
```

Key design decisions:
- **`fq` (filter query)** is used instead of adding filters to `q`. Solr caches `fq` results separately, making filtered searches very fast.
- **Tags use AND logic**: each tag becomes a separate `fq`, so if the user selects tags "go" AND "cli", both must be present.
- **`%q` quoting** ensures tag/channel values with spaces are safely quoted.
- **Stars filter** uses Solr range syntax `[N TO *]` for "greater than or equal to N".

### 5. Call `applyFilters` in both methods

In both `Search` and `List`, call `applyFilters(params, opts)` after constructing the base params and before building the select URL:

```go
params := url.Values{
    // ... existing params ...
}
applyFilters(params, opts)
selectURL := fmt.Sprintf(...)
```

### 6. Do NOT update callers yet

The handler in `internal/handler/search.go` still calls `Search` and `List` with the old signatures. It will be updated in prompt 68. This prompt may leave the project in a non-compiling state — that is expected.

## Verification

After applying the changes:

1. Run `go vet ./internal/store/...` — should pass for the store package itself.
2. `go build ./...` will fail due to handler calling old signatures — this is expected and will be fixed in prompt 68.

## Files to modify

- `internal/store/store.go` — add `SearchOptions`, update `Search`/`List` signatures, add `applyFilters`

## Files NOT to modify

- `internal/handler/search.go` — will be updated in prompt 68
- `static/index.html` — will be updated in prompt 69
- `internal/store/store_test.go` — tests will be updated after handler is done
