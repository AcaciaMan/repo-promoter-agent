# Prompt: Highlighting — Request and Parse Solr Hit Highlights

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. The app stores AI-generated promotional content for GitHub repos in **Apache Solr 10** and exposes full-text search via `GET /api/search?q=...`.

Phase 2 (faceted search & filtering, prompts 66–70) is complete — the store returns `SearchResult` with `Results` and `Facets`, supports `SearchOptions` for filtering, and the handler/frontend wire it all up.

This is **Phase 3, prompt 1 of 4** for search highlighting & sort options. This prompt adds Solr highlighting parameters and parses highlight data from responses.

## Current state

### `SearchResult` struct in `internal/store/store.go`

```go
type SearchResult struct {
    Results []Promotion        `json:"results"`
    Facets  map[string][]Facet `json:"facets,omitempty"`
}
```

### `Search` method (key parts)

```go
func (s *Store) Search(ctx context.Context, query string, limit int, opts SearchOptions) (SearchResult, error) {
    // ...
    params := url.Values{
        "q":              {q},
        "defType":        {"edismax"},
        "qf":             {"repo_name^3 headline^4 summary^2 key_benefits^1.5 tags^3 ..."},
        "pf":             {"headline^6 summary^3 repo_name^4"},
        // ... ps, mm, tie, rows, sort, wt, fl, facet params ...
    }
    applyFilters(params, opts)
    // ... HTTP request, read body ...
    docs, err := parseSolrDocs(body)
    facets := parseFacets(body)
    return SearchResult{Results: docs, Facets: facets}, nil
}
```

### `List` method

Uses `q=*:*` with `sort=created_at desc` — no edismax, no highlighting needed.

## Problem

When users search, they see result cards but don't know **why** a result matched their query. Solr can return highlighted snippets with matched terms wrapped in `<mark>` tags, but we're not requesting or parsing them.

## Your task

1. Add a `Highlights` field to `SearchResult`.
2. Add highlighting parameters to the `Search` method's Solr query.
3. Add a `parseHighlights` helper to extract highlight data from Solr responses.
4. Return highlights in `SearchResult` from `Search` (but not from `List`).

## Requirements

### 1. Update `SearchResult` struct

Add a `Highlights` field:

```go
type SearchResult struct {
    Results    []Promotion                  `json:"results"`
    Facets     map[string][]Facet           `json:"facets,omitempty"`
    Highlights map[string]map[string]string `json:"highlights,omitempty"`
}
```

The `Highlights` map is keyed by **document ID** (which is `repo_url`), then by **field name**, with the value being an HTML string containing `<mark>` tags around matched terms.

Example:
```json
{
  "highlights": {
    "https://github.com/owner/repo": {
      "headline": "Fast <mark>CLI</mark> <mark>Testing</mark> Framework",
      "summary": "A tool for running <mark>CLI</mark> <mark>tests</mark> quickly."
    }
  }
}
```

### 2. Add highlighting params to `Search`

Add these parameters to the `params` block in the `Search` method:

```go
"hl":              {"true"},
"hl.fl":           {"headline,summary,key_benefits,linkedin_post,call_to_action,target_audience"},
"hl.simple.pre":   {"<mark>"},
"hl.simple.post":  {"</mark>"},
"hl.snippets":     {"2"},
"hl.fragsize":     {"200"},
"hl.method":       {"unified"},
```

**Design decisions:**
- `hl.fl` — highlight on the most user-visible text fields. Excludes `tags` (exact string, not tokenized), `twitter_posts` (less useful), `readme` (too long/noisy), `repo_name` (short, shown as a link).
- `hl.simple.pre/post` — wraps matched terms in `<mark>` tags (HTML5 highlight element).
- `hl.snippets=2` — return up to 2 highlighted fragments per field.
- `hl.fragsize=200` — each fragment is up to 200 characters (enough context without overwhelming the UI).
- `hl.method=unified` — Solr's recommended highlighter, handles phrase queries and multi-term matches well.

### 3. Add `parseHighlights` helper

Solr returns highlights in this JSON structure:

```json
{
  "highlighting": {
    "https://github.com/owner/repo": {
      "headline": ["Fast <mark>CLI</mark> <mark>Testing</mark> Framework"],
      "summary": ["A tool for <mark>CLI</mark> tests.", "Supports <mark>testing</mark> in CI."]
    }
  }
}
```

Each field value is an **array of snippets**. We join them with `" … "` into a single string for easier frontend consumption.

```go
// parseHighlights extracts highlighted snippets from a Solr response.
// Returns a map of document ID → field name → joined highlight HTML.
func parseHighlights(body []byte) map[string]map[string]string {
    var envelope struct {
        Highlighting map[string]map[string][]string `json:"highlighting"`
    }
    if err := json.Unmarshal(body, &envelope); err != nil {
        return nil
    }
    if len(envelope.Highlighting) == 0 {
        return nil
    }

    result := make(map[string]map[string]string)
    for docID, fields := range envelope.Highlighting {
        fieldMap := make(map[string]string)
        for field, snippets := range fields {
            if len(snippets) > 0 {
                fieldMap[field] = strings.Join(snippets, " … ")
            }
        }
        if len(fieldMap) > 0 {
            result[docID] = fieldMap
        }
    }
    return result
}
```

### 4. Update `Search` to return highlights

After reading the response body, call `parseHighlights(body)` alongside `parseSolrDocs` and `parseFacets`:

```go
docs, err := parseSolrDocs(body)
if err != nil {
    return SearchResult{}, err
}
facets := parseFacets(body)
highlights := parseHighlights(body)
return SearchResult{Results: docs, Facets: facets, Highlights: highlights}, nil
```

### 5. Do NOT add highlighting to `List`

The `List` method uses `q=*:*` (match all). Highlighting on `*:*` produces no useful output. Leave `List` unchanged — it returns `SearchResult` with `Highlights: nil`.

## Verification

After applying the changes:

1. Run `go build ./...` — must compile without errors.
2. Run `go test ./internal/store/... -tags integration` — existing tests must pass.

## Files to modify

- `internal/store/store.go` — update `SearchResult`, add hl params to `Search`, add `parseHighlights`, return highlights from `Search`

## Files NOT to modify

- `internal/handler/search.go` — will be updated in prompt 72
- `static/index.html` — will be updated in prompt 73
- `internal/store/store_test.go` — no changes needed (tests don't inspect highlights)
