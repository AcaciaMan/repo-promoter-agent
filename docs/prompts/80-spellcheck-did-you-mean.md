# Prompt: Spell Correction — "Did You Mean?" with Solr SpellCheck

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. The app stores AI-generated promotional content in **Apache Solr 10** and exposes full-text search via `GET /api/search?q=...`.

Phase 4 (analysis indexing & autocomplete, prompts 75–79) is complete. This is **Phase 5, prompt 1 of 5** for advanced features.

**Goal**: When a user misspells a query (e.g. "containr" instead of "container"), show a "Did you mean: container?" link above the results. This uses Solr's built-in SpellCheckComponent.

## Current state

### `SearchResult` in `internal/store/store.go`

```go
type SearchResult struct {
    Results    []Promotion                  `json:"results"`
    Facets     map[string][]Facet           `json:"facets,omitempty"`
    Highlights map[string]map[string]string `json:"highlights,omitempty"`
}
```

### `searchResponse` in `internal/handler/search.go`

```go
type searchResponse struct {
    Results    []store.Promotion            `json:"results"`
    Count      int                          `json:"count"`
    Facets     map[string][]store.Facet     `json:"facets,omitempty"`
    Highlights map[string]map[string]string `json:"highlights,omitempty"`
}
```

### `Store.Search()` — params (key section)

```go
params := url.Values{
    "q":              {q},
    "defType":        {"edismax"},
    "qf":             {"repo_name^3 headline^4 ..."},
    // ... pf, ps, mm, tie, rows, sort, wt, fl, facet, hl params ...
}
```

### Frontend `doSearch()` — results handling

```js
const d=await res.json();
currentHighlights=d.highlights||{};
renderActiveFilters();
renderFacets(d.facets);
if(!d.results||d.results.length===0){searchResults.innerHTML='<p style="color:#888">No results found.</p>';return}
searchResults.innerHTML=d.results.map(renderCompact).join('');
```

## Your task

1. Configure Solr SpellCheckComponent via Config API
2. Add spellcheck params to `Store.Search()` and parse the collation
3. Add the collation to `SearchResult` and the handler response
4. Show the "Did you mean?" link in the frontend

## Requirements

### 1. Configure Solr SpellCheckComponent

Run in PowerShell:

```powershell
$solrBase = "http://localhost:8983/solr/promotions/config"

# Add the spellcheck search component using DirectSolrSpellChecker on _text_
$body = @'
{
  "add-searchcomponent": {
    "name": "spellcheck",
    "class": "solr.SpellCheckComponent",
    "spellchecker": {
      "name": "default",
      "classname": "solr.DirectSolrSpellChecker",
      "field": "_text_",
      "distanceMeasure": "internal",
      "accuracy": "0.5",
      "maxEdits": "2",
      "minPrefix": "1",
      "maxInspections": "5",
      "minQueryLength": "3",
      "maxQueryFrequency": "0.01"
    }
  }
}
'@
Invoke-RestMethod -Uri $solrBase -Method POST -ContentType "application/json" -Body $body

# Update the /select request handler to include spellcheck as a last-component
# This adds spellcheck processing to the standard search handler
$body = @'
{
  "update-requesthandler": {
    "name": "/select",
    "class": "solr.SearchHandler",
    "defaults": {
      "wt": "json"
    },
    "last-components": ["spellcheck"]
  }
}
'@
Invoke-RestMethod -Uri $solrBase -Method POST -ContentType "application/json" -Body $body
```

Verify:
```powershell
# Test spellcheck directly — should not error
$r = Invoke-RestMethod -Uri "http://localhost:8983/solr/promotions/select?q=containr&spellcheck=true&spellcheck.collate=true&wt=json" -Method GET
$r | ConvertTo-Json -Depth 5
```

### 2. Update `SearchResult` in `internal/store/store.go`

Add a `Collation` field:

```go
type SearchResult struct {
    Results    []Promotion                  `json:"results"`
    Facets     map[string][]Facet           `json:"facets,omitempty"`
    Highlights map[string]map[string]string `json:"highlights,omitempty"`
    Collation  string                       `json:"collation,omitempty"`
}
```

### 3. Add spellcheck params to `Store.Search()`

Add these params to the existing `params` url.Values in the `Search` method (add after the `"hl.method"` line):

```go
"spellcheck":          {"true"},
"spellcheck.collate":  {"true"},
"spellcheck.count":    {"5"},
"spellcheck.maxCollations": {"1"},
```

### 4. Add `parseCollation` helper to `internal/store/store.go`

Add after the `parseHighlights` function:

```go
// parseCollation extracts the best spell-check collation from a Solr response.
func parseCollation(body []byte) string {
	var envelope struct {
		Spellcheck struct {
			Collations []interface{} `json:"collations"`
		} `json:"spellcheck"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return ""
	}
	// Solr returns collations as alternating ["collation", "suggested query", ...]
	for i := 0; i+1 < len(envelope.Spellcheck.Collations); i += 2 {
		key, _ := envelope.Spellcheck.Collations[i].(string)
		if key == "collation" {
			if val, ok := envelope.Spellcheck.Collations[i+1].(string); ok {
				return val
			}
			// Could also be an object with "collationQuery" field
			if obj, ok := envelope.Spellcheck.Collations[i+1].(map[string]interface{}); ok {
				if cq, ok := obj["collationQuery"].(string); ok {
					return cq
				}
			}
		}
	}
	return ""
}
```

### 5. Use `parseCollation` in `Search()` return

In the `Search` method, after parsing highlights, also parse the collation. Update the return block:

Replace:

```go
docs, err := parseSolrDocs(body)
if err != nil {
    return SearchResult{}, err
}
facets := parseFacets(body)
highlights := parseHighlights(body)
return SearchResult{Results: docs, Facets: facets, Highlights: highlights}, nil
```

With:

```go
docs, err := parseSolrDocs(body)
if err != nil {
    return SearchResult{}, err
}
facets := parseFacets(body)
highlights := parseHighlights(body)
collation := parseCollation(body)
return SearchResult{Results: docs, Facets: facets, Highlights: highlights, Collation: collation}, nil
```

### 6. Update `searchResponse` in `internal/handler/search.go`

Add Collation to the response struct:

```go
type searchResponse struct {
    Results    []store.Promotion            `json:"results"`
    Count      int                          `json:"count"`
    Facets     map[string][]store.Facet     `json:"facets,omitempty"`
    Highlights map[string]map[string]string `json:"highlights,omitempty"`
    Collation  string                       `json:"collation,omitempty"`
}
```

And update the `json.NewEncoder` call at the end of `ServeHTTP`:

Replace:

```go
json.NewEncoder(w).Encode(searchResponse{
    Results:    sr.Results,
    Count:      len(sr.Results),
    Facets:     sr.Facets,
    Highlights: sr.Highlights,
})
```

With:

```go
json.NewEncoder(w).Encode(searchResponse{
    Results:    sr.Results,
    Count:      len(sr.Results),
    Facets:     sr.Facets,
    Highlights: sr.Highlights,
    Collation:  sr.Collation,
})
```

### 7. Add "Did you mean?" UI in `static/index.html`

#### 7a. Add CSS (in the `<style>` block, after the `.autocomplete-item .ac-weight` rule):

```css
.did-you-mean{background:#fff8e1;border:1px solid #ffe082;border-radius:6px;padding:.5rem .75rem;margin:.5rem 0;font-size:.9rem;color:#6d4c00}
.did-you-mean a{color:#0969da;font-weight:600;cursor:pointer;text-decoration:underline}
.did-you-mean a:hover{color:#0550ae}
```

#### 7b. Add HTML container

Add a `<div id="search-collation"></div>` right after the `<div id="search-error" class="error"></div>` line:

```html
<div id="search-error" class="error"></div>
<div id="search-collation"></div>
```

#### 7c. Update `doSearch()` in the `<script>` block

In the `doSearch` function, after `currentHighlights=d.highlights||{};`, add collation rendering. Also clear the collation div at the top of the function.

Add at the start of try block (after the first line that clears status/error/results):

```js
document.getElementById('search-collation').innerHTML='';
```

After `currentHighlights=d.highlights||{};`, add:

```js
if(d.collation){
    document.getElementById('search-collation').innerHTML='<div class="did-you-mean">Did you mean: <a onclick="searchQ.value=\''+esc(d.collation).replace(/'/g,"\\'")+'\';doSearch()">'+esc(d.collation)+'</a>?</div>';
} else {
    document.getElementById('search-collation').innerHTML='';
}
```

## What NOT to do

- Do **NOT** enable spellcheck for the `List` method — it's only useful for full-text queries
- Do **NOT** add spellcheck to the suggest endpoint — it has its own logic
- Do **NOT** modify any files beyond `store.go`, `search.go`, and `index.html`

## Verification

```powershell
go build ./...
```

After starting the server:
1. Search for "containr" (misspelled) → should show "Did you mean: container?" if container-related docs exist
2. Search for "container" (correct) → should NOT show "Did you mean?" 
3. Click the "Did you mean?" link → should re-search with the corrected term
