# Prompt: Faceted Search — Return Facet Counts from Solr

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. The app stores AI-generated promotional content for GitHub repos in **Apache Solr 10** and exposes full-text search via `GET /api/search?q=...`.

Phase 1 (relevance tuning, prompts 63–65) is complete — edismax has field boosting, phrase fields, minimum match, and tie-breaker.

This is **Phase 2, prompt 1 of 5** for faceted search and filtering. This prompt modifies `internal/store/store.go` to request facet counts from Solr and return them alongside results.

## Current state

### `Store.Search()` in `internal/store/store.go`

The `Search` method returns `([]Promotion, error)`. It builds edismax params and calls Solr's `/select` endpoint. It does **not** request facets. The full method:

```go
func (s *Store) Search(ctx context.Context, query string, limit int) ([]Promotion, error) {
    // ... sanitization, params with qf/pf/ps/mm/tie ...
    selectURL := fmt.Sprintf("%s/solr/%s/select?%s", s.baseURL, s.core, params.Encode())
    // ... HTTP GET, read body ...
    return parseSolrDocs(body)
}
```

### `Store.List()` in `internal/store/store.go`

Returns recent promotions with `q=*:*` sorted by `created_at desc`. Also returns `([]Promotion, error)` with no facets.

### Solr schema (relevant fields for faceting)

| Field | Solr Type | Notes |
|---|---|---|
| `tags` | `string` (multiValued) | Exact-match, ideal for faceting |
| `target_channel` | `string` | Exact-match, ideal for faceting |
| `stars` | `pint` | Numeric, supports range faceting |

## Your task

1. Add a `SearchResult` struct that groups results and facet counts.
2. Modify `Search` to request Solr facets and return `SearchResult`.
3. Modify `List` to also request facets and return `SearchResult`.
4. Add a `parseFacets` helper to extract facet data from Solr responses.

## Requirements

### 1. Add `SearchResult` struct

Add this struct after the existing `Promotion` struct:

```go
// SearchResult holds search results together with facet counts.
type SearchResult struct {
    Results []Promotion        `json:"results"`
    Facets  map[string][]Facet `json:"facets,omitempty"`
}

// Facet represents a single facet value and its document count.
type Facet struct {
    Value string `json:"value"`
    Count int    `json:"count"`
}
```

### 2. Update `Search` to return `SearchResult`

Change the signature from:
```go
func (s *Store) Search(ctx context.Context, query string, limit int) ([]Promotion, error)
```
to:
```go
func (s *Store) Search(ctx context.Context, query string, limit int) (SearchResult, error)
```

Add facet parameters to the Solr query `params`:
```go
"facet":          {"true"},
"facet.field":    {"tags", "target_channel"},
"facet.mincount": {"1"},
```

Note: `facet.field` must appear **twice** (one for `tags`, one for `target_channel`). In Go's `url.Values`, set it as a slice:
```go
params["facet.field"] = []string{"tags", "target_channel"}
```

Parse both docs and facets from the response body. Return:
```go
return SearchResult{Results: docs, Facets: facets}, nil
```

On error or empty query, return `SearchResult{}` with an empty or nil facets map.

### 3. Update `List` to return `SearchResult`

Change the signature the same way:
```go
func (s *Store) List(ctx context.Context, limit int) (SearchResult, error)
```

Add the same facet parameters to the List query. This lets the browse view (no search query) also show available facets.

### 4. Add `parseFacets` helper

Solr returns facets in this JSON structure:
```json
{
  "facet_counts": {
    "facet_fields": {
      "tags": ["go", 5, "testing", 3, "python", 2],
      "target_channel": ["general", 8, "twitter", 3]
    }
  }
}
```

The values are **alternating** name/count pairs in a flat array.

Add a helper function:
```go
// parseFacets extracts facet counts from a Solr response body.
func parseFacets(body []byte) map[string][]Facet {
    var envelope struct {
        FacetCounts struct {
            FacetFields map[string][]interface{} `json:"facet_fields"`
        } `json:"facet_counts"`
    }
    if err := json.Unmarshal(body, &envelope); err != nil {
        return nil
    }
    if len(envelope.FacetCounts.FacetFields) == 0 {
        return nil
    }

    facets := make(map[string][]Facet)
    for field, pairs := range envelope.FacetCounts.FacetFields {
        var items []Facet
        for i := 0; i+1 < len(pairs); i += 2 {
            name, _ := pairs[i].(string)
            count, _ := pairs[i+1].(float64)
            if name != "" && int(count) > 0 {
                items = append(items, Facet{Value: name, Count: int(count)})
            }
        }
        if len(items) > 0 {
            facets[field] = items
        }
    }
    return facets
}
```

### 5. Update `parseSolrDocs` calls

In both `Search` and `List`, after reading the response body, call both `parseSolrDocs(body)` and `parseFacets(body)` on the **same** body bytes. Combine results:

```go
docs, err := parseSolrDocs(body)
if err != nil {
    return SearchResult{}, err
}
facets := parseFacets(body)
return SearchResult{Results: docs, Facets: facets}, nil
```

## Verification

After applying the changes:

1. Run `go build ./...` — **this will fail** because `internal/handler/search.go` references the old `Search`/`List` signatures. That is expected and will be fixed in prompt 68.
2. Verify that the new types and `parseFacets` function are correct by reading through the code.

## Files to modify

- `internal/store/store.go` — add `SearchResult`/`Facet` types, update `Search`/`List` signatures, add `parseFacets`

## Files NOT to modify

- `internal/handler/search.go` — will be updated in prompt 68
- `static/index.html` — will be updated in prompt 69
- `internal/store/store_test.go` — will be updated in prompt 67
