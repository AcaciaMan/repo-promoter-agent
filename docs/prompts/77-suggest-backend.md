# Prompt: Solr Suggester Configuration & Suggest API Endpoint

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. The app stores AI-generated promotional content in **Apache Solr 10** and exposes full-text search via `GET /api/search?q=...`.

This is **Phase 4, prompt 3 of 5**. Prompts 75–76 added indexed analysis fields and updated `Save`/`Search` to use them. This prompt configures the Solr Suggester component for autocomplete and adds the backend endpoint.

## Current state

### Solr core: `promotions` at `http://localhost:8983`

The core has text fields including `repo_name`, `headline`, `tags`, and the new analysis fields. No suggester component is configured yet.

### `cmd/server/main.go` — route registration

```go
mux.Handle("/api/generate", limiter.Middleware("generate")(handler.NewGenerateHandler(agentClient, githubClient, st, analysisClient)))
mux.Handle("/api/search", limiter.Middleware("search")(handler.NewSearchHandler(st)))
mux.Handle("/", noCacheHandler(http.FileServerFS(static.Files)))
```

### `internal/store/store.go` — Store struct

```go
type Store struct {
    baseURL string
    core    string
    client  *http.Client
}
```

### `internal/handler/search.go` — only handler file in the handler package

Contains `SearchHandler` with `ServeHTTP` and helper `writeError`.

## Your task

1. Configure the Solr Suggester search component via the Config API
2. Add a `Store.Suggest()` method in `internal/store/store.go`
3. Create a new `SuggestHandler` in `internal/handler/suggest.go`
4. Wire the new endpoint in `cmd/server/main.go`

## Requirements

### 1. Configure Solr Suggester via Config API

The Solr Suggester needs to be configured as a search component and a request handler. Run these PowerShell commands:

```powershell
$solrBase = "http://localhost:8983/solr/promotions/config"

# Step 1: Add the suggester search component
$body = @'
{
  "add-searchcomponent": {
    "name": "suggest",
    "class": "solr.SuggestComponent",
    "suggester": {
      "name": "default",
      "lookupImpl": "AnalyzingInfixLookupFactory",
      "dictionaryImpl": "DocumentDictionaryFactory",
      "field": "headline",
      "weightField": "stars",
      "suggestAnalyzerFieldType": "text_general",
      "buildOnStartup": "true",
      "buildOnCommit": "true"
    }
  }
}
'@
Invoke-RestMethod -Uri $solrBase -Method POST -ContentType "application/json" -Body $body

# Step 2: Add the /suggest request handler
$body = @'
{
  "add-requesthandler": {
    "name": "/suggest",
    "class": "solr.SearchHandler",
    "defaults": {
      "suggest": "true",
      "suggest.count": "10",
      "suggest.dictionary": "default"
    },
    "components": ["suggest"]
  }
}
'@
Invoke-RestMethod -Uri $solrBase -Method POST -ContentType "application/json" -Body $body
```

**Design choices**:
- Uses `AnalyzingInfixLookupFactory` — matches anywhere within the field, not just prefix. This means typing "cli" will match "Lightning Fast CLI Framework" which is much more useful than prefix-only matching.
- Uses `headline` as the suggestion source — it's the most descriptive, human-readable field for suggestions.
- Uses `stars` as the weight field — popular repos rank higher in suggestions.
- `buildOnCommit=true` — the suggestion index rebuilds automatically when new docs are committed.

### 2. Verify the suggester is configured

```powershell
# Build the suggester index
Invoke-RestMethod -Uri "http://localhost:8983/solr/promotions/suggest?suggest.build=true" -Method GET

# Test with a prefix query (will return empty if no docs exist yet, but should not error)
$r = Invoke-RestMethod -Uri "http://localhost:8983/solr/promotions/suggest?suggest.q=test" -Method GET
$r | ConvertTo-Json -Depth 5
```

### 3. Add `Suggest()` method to `internal/store/store.go`

Add the following method and types after the `List` method:

```go
// Suggestion represents a single autocomplete suggestion.
type Suggestion struct {
	Term   string `json:"term"`
	Weight int    `json:"weight"`
}

// Suggest returns autocomplete suggestions for the given prefix.
func (s *Store) Suggest(ctx context.Context, prefix string, limit int) ([]Suggestion, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 20 {
		limit = 20
	}

	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return nil, nil
	}

	params := url.Values{
		"suggest.q":     {prefix},
		"suggest.count": {fmt.Sprintf("%d", limit)},
	}
	suggestURL := fmt.Sprintf("%s/solr/%s/suggest?%s", s.baseURL, s.core, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, suggestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create suggest request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("suggest from solr: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read suggest response: %w", err)
	}

	return parseSuggestions(body)
}

// parseSuggestions extracts suggestions from a Solr suggest response.
func parseSuggestions(body []byte) ([]Suggestion, error) {
	var envelope struct {
		Suggest map[string]map[string]struct {
			Suggestions []struct {
				Term   string `json:"term"`
				Weight int    `json:"weight"`
			} `json:"suggestions"`
		} `json:"suggest"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("parse suggest response: %w", err)
	}

	var result []Suggestion
	for _, dict := range envelope.Suggest {
		for _, entry := range dict {
			for _, s := range entry.Suggestions {
				result = append(result, Suggestion{
					Term:   s.Term,
					Weight: s.Weight,
				})
			}
		}
	}
	return result, nil
}
```

**Key points**:
- `Suggestion` is a new exported type — lightweight, just term + weight
- `parseSuggestions` handles the nested Solr response structure: `suggest → {dictionary} → {query} → suggestions[]`
- Limit is capped at 20 to prevent abuse
- Empty prefix returns nil immediately (no Solr call)
- No new imports needed — `strings`, `url`, `fmt`, `json`, `io`, `http`, `context` are all already imported

### 4. Create `internal/handler/suggest.go`

Create a new file `internal/handler/suggest.go`:

```go
package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"repo-promoter-agent/internal/store"
)

// SuggestHandler handles GET /api/suggest requests.
type SuggestHandler struct {
	store *store.Store
}

// NewSuggestHandler creates a SuggestHandler with the given store.
func NewSuggestHandler(st *store.Store) *SuggestHandler {
	return &SuggestHandler{store: st}
}

func (h *SuggestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	q := r.URL.Query().Get("q")
	if q == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]store.Suggestion{})
		return
	}

	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	suggestions, err := h.store.Suggest(r.Context(), q, limit)
	if err != nil {
		log.Printf("Suggest failed: %v", err)
		writeError(w, http.StatusInternalServerError, "suggest failed")
		return
	}
	if suggestions == nil {
		suggestions = []store.Suggestion{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(suggestions)
}
```

**Note**: `writeError` is already defined in `search.go` in the same package — no duplication needed.

### 5. Wire the endpoint in `cmd/server/main.go`

Add the suggest route alongside the existing search route. Update the route registration section:

Replace:

```go
mux.Handle("/api/generate", limiter.Middleware("generate")(handler.NewGenerateHandler(agentClient, githubClient, st, analysisClient)))
mux.Handle("/api/search", limiter.Middleware("search")(handler.NewSearchHandler(st)))
mux.Handle("/", noCacheHandler(http.FileServerFS(static.Files)))
```

With:

```go
mux.Handle("/api/generate", limiter.Middleware("generate")(handler.NewGenerateHandler(agentClient, githubClient, st, analysisClient)))
mux.Handle("/api/search", limiter.Middleware("search")(handler.NewSearchHandler(st)))
mux.Handle("/api/suggest", limiter.Middleware("search")(handler.NewSuggestHandler(st)))
mux.Handle("/", noCacheHandler(http.FileServerFS(static.Files)))
```

The suggest endpoint uses the same `"search"` rate limit bucket — it's lightweight and shares the same usage pattern.

## What NOT to do

- Do **NOT** modify `static/index.html` — frontend autocomplete is in prompt 78
- Do **NOT** add a separate rate limit bucket for suggest — reuse `"search"`
- Do **NOT** use `sanitizeSolrQuery` on suggestion prefixes — the Solr suggest handler doesn't use query syntax

## Verification

```powershell
# Build successfully
go build ./...

# Verify the new suggest.go file exists
Test-Path internal/handler/suggest.go  # Should print True
```

After starting the server (if Solr is running with some indexed data):

```powershell
# Test empty query
Invoke-RestMethod -Uri "http://localhost:8080/api/suggest?q=" -Method GET
# Should return []

# Test with a prefix (results depend on indexed data)
Invoke-RestMethod -Uri "http://localhost:8080/api/suggest?q=cli" -Method GET
# Should return suggestions array (possibly empty if no matching docs)
```
