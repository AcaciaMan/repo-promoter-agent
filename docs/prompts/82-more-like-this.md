# Prompt: More Like This — "Find Similar" for Result Cards

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. The app stores AI-generated promotional content in **Apache Solr 10** and exposes full-text search.

This is **Phase 5, prompt 3 of 5**. Prompts 80–81 added spellcheck and result grouping. This prompt adds "More Like This" — a button on expanded result cards that finds similar promotions based on content similarity.

## Current state

### `Store` in `internal/store/store.go`

```go
type Store struct {
    baseURL string
    core    string
    client  *http.Client
}
```

Methods: `New()`, `Save()`, `Search()`, `List()`, `Suggest()`, plus helpers.

### Route registration in `cmd/server/main.go`

```go
mux.Handle("/api/generate", limiter.Middleware("generate")(handler.NewGenerateHandler(agentClient, githubClient, st, analysisClient)))
mux.Handle("/api/search", limiter.Middleware("search")(handler.NewSearchHandler(st)))
mux.Handle("/api/suggest", limiter.Middleware("search")(handler.NewSuggestHandler(st)))
mux.Handle("/", noCacheHandler(http.FileServerFS(static.Files)))
```

### Frontend `renderExpandedBody(d)` — generates expanded card content

```js
function renderExpandedBody(d){
    let h='<p>'+hlField(d.repo_url,'summary',d.summary)+'</p>';
    if(d.key_benefits&&d.key_benefits.length){...}
    if(d.twitter_posts&&d.twitter_posts.length){...}
    if(d.linkedin_post){...}
    if(d.call_to_action){...}
    h+=renderRepoStats(d);
    h+=renderTraffic(d);
    h+=renderAnalysis(d.analysis);
    return h;
}
```

## Your task

1. Add a `Store.MoreLikeThis()` method using Solr's MLT handler
2. Create `MoreLikeThisHandler` and `GET /api/mlt?id=<repo_url>` endpoint
3. Wire the route in `main.go`
4. Add a "Find Similar" button in expanded cards that displays similar results

## Requirements

### 1. Add `MoreLikeThis()` method to `internal/store/store.go`

Add after the `Suggest` method:

```go
// MoreLikeThis returns promotions similar to the document identified by docID.
// It uses Solr's MLT (More Like This) query parser on content fields.
func (s *Store) MoreLikeThis(ctx context.Context, docID string, limit int) ([]Promotion, error) {
	if limit <= 0 {
		limit = 5
	}
	if limit > 10 {
		limit = 10
	}

	docID = strings.TrimSpace(docID)
	if docID == "" {
		return nil, nil
	}

	params := url.Values{
		"q":             {fmt.Sprintf("id:%q", docID)},
		"fl":            {"*"},
		"wt":            {"json"},
		"mlt":           {"true"},
		"mlt.fl":        {"summary,tags,key_benefits,headline,analysis_value_proposition"},
		"mlt.mintf":     {"1"},
		"mlt.mindf":     {"1"},
		"mlt.count":     {fmt.Sprintf("%d", limit)},
		"mlt.interestingTerms": {"details"},
	}

	selectURL := fmt.Sprintf("%s/solr/%s/select?%s", s.baseURL, s.core, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, selectURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create mlt request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mlt from solr: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read mlt response: %w", err)
	}

	return parseMLTDocs(body)
}

// parseMLTDocs extracts the More Like This results from a Solr MLT response.
func parseMLTDocs(body []byte) ([]Promotion, error) {
	// Solr MLT component returns moreLikeThis → {docID} → {docs}
	var envelope struct {
		MoreLikeThis map[string]struct {
			Docs []map[string]interface{} `json:"docs"`
		} `json:"moreLikeThis"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("parse mlt response: %w", err)
	}

	var allDocs []map[string]interface{}
	for _, group := range envelope.MoreLikeThis {
		allDocs = append(allDocs, group.Docs...)
	}
	if len(allDocs) == 0 {
		return nil, nil
	}

	// Reuse parseSolrDocs logic by re-wrapping into the standard response format
	wrapped := struct {
		Response struct {
			Docs []map[string]interface{} `json:"docs"`
		} `json:"response"`
	}{}
	wrapped.Response.Docs = allDocs
	wrappedBytes, err := json.Marshal(wrapped)
	if err != nil {
		return nil, fmt.Errorf("rewrap mlt docs: %w", err)
	}
	return parseSolrDocs(wrappedBytes)
}
```

**Design notes**:
- Uses `mlt.fl` with the most distinctive content fields — `summary`, `tags`, `key_benefits`, `headline`, `analysis_value_proposition`
- `mlt.mintf=1` and `mlt.mindf=1` ensure matching works even with small corpora
- Limit capped at 10 to prevent abuse
- Reuses `parseSolrDocs` via re-wrapping to avoid duplicating field extraction logic

### 2. Create `internal/handler/mlt.go`

```go
package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"repo-promoter-agent/internal/store"
)

// MLTHandler handles GET /api/mlt requests for "More Like This" results.
type MLTHandler struct {
	store *store.Store
}

// NewMLTHandler creates a MLTHandler with the given store.
func NewMLTHandler(st *store.Store) *MLTHandler {
	return &MLTHandler{store: st}
}

func (h *MLTHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	docID := r.URL.Query().Get("id")
	if docID == "" {
		writeError(w, http.StatusBadRequest, "id parameter required")
		return
	}

	limit := 5
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	results, err := h.store.MoreLikeThis(r.Context(), docID, limit)
	if err != nil {
		log.Printf("MLT failed: %v", err)
		writeError(w, http.StatusInternalServerError, "mlt failed")
		return
	}
	if results == nil {
		results = []store.Promotion{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Results []store.Promotion `json:"results"`
		Count   int               `json:"count"`
	}{
		Results: results,
		Count:   len(results),
	})
}
```

### 3. Wire the route in `cmd/server/main.go`

Add the MLT route after the suggest route:

Replace:

```go
mux.Handle("/api/suggest", limiter.Middleware("search")(handler.NewSuggestHandler(st)))
mux.Handle("/", noCacheHandler(http.FileServerFS(static.Files)))
```

With:

```go
mux.Handle("/api/suggest", limiter.Middleware("search")(handler.NewSuggestHandler(st)))
mux.Handle("/api/mlt", limiter.Middleware("search")(handler.NewMLTHandler(st)))
mux.Handle("/", noCacheHandler(http.FileServerFS(static.Files)))
```

### 4. Add frontend "Find Similar" button and rendering

#### 4a. Add CSS (in `<style>`, after the `.channel-tab .ch-icon` rule):

```css
.mlt-btn{display:inline-block;margin:.75rem 0 .25rem;padding:.35rem .75rem;font-size:.8rem;background:#f0f7ff;border:1px solid #c8e1ff;color:#0969da;border-radius:4px;cursor:pointer}
.mlt-btn:hover{background:#dbeafe;border-color:#0969da}
.mlt-results{margin:.5rem 0;padding:.5rem;background:#fafbfc;border:1px solid #e1e4e8;border-radius:6px}
.mlt-results h4{margin:0 0 .5rem;font-size:.9rem;color:#555}
.mlt-card{background:#fff;border:1px solid #e1e4e8;border-radius:4px;padding:.5rem .75rem;margin:.3rem 0;font-size:.85rem}
.mlt-card a{color:#0969da;text-decoration:none;font-weight:600}
.mlt-card a:hover{text-decoration:underline}
.mlt-card .mlt-meta{color:#666;font-size:.8rem}
```

#### 4b. Update `renderExpandedBody(d)` 

At the end of the function, before `return h;`, add the "Find Similar" button:

Replace the return section of `renderExpandedBody`:

```js
h+=renderRepoStats(d);
h+=renderTraffic(d);
h+=renderAnalysis(d.analysis);
return h;
```

With:

```js
h+=renderRepoStats(d);
h+=renderTraffic(d);
h+=renderAnalysis(d.analysis);
h+='<button class="mlt-btn" onclick="event.stopPropagation();findSimilar(this,\''+esc(d.repo_url).replace(/'/g,"\\'")+'\')">🔍 Find Similar</button>';
h+='<div class="mlt-container"></div>';
return h;
```

#### 4c. Add `findSimilar` function

Add in the `<script>` block, after the `switchChannelTab` function:

```js
async function findSimilar(btn, repoUrl){
    const container=btn.nextElementSibling;
    if(container.innerHTML){container.innerHTML='';btn.textContent='🔍 Find Similar';return}
    btn.textContent='Loading…';btn.disabled=true;
    try{
        const res=await fetch('/api/mlt?id='+encodeURIComponent(repoUrl)+'&limit=5');
        if(!res.ok)throw new Error('MLT request failed');
        const d=await res.json();
        if(!d.results||d.results.length===0){
            container.innerHTML='<div class="mlt-results"><p style="color:#888;font-size:.85rem">No similar promotions found.</p></div>';
        }else{
            let h='<div class="mlt-results"><h4>Similar Promotions</h4>';
            d.results.forEach(p=>{
                const tags=(p.tags||[]).slice(0,3).map(t=>'<span class="pill">'+esc(t)+'</span>').join('');
                h+='<div class="mlt-card">'
                    +'<a href="'+safeHref(p.repo_url)+'" target="_blank" rel="noopener">'+esc(p.repo_name)+'</a>'
                    +' <span class="mlt-meta">'+esc(p.target_channel||'general')+'</span>'
                    +'<p style="margin:.2rem 0;font-size:.85rem">'+esc(p.headline)+'</p>'
                    +'<div class="pills" style="margin-top:.2rem">'+tags+'</div>'
                    +'</div>';
            });
            h+='</div>';
            container.innerHTML=h;
        }
        btn.textContent='🔍 Hide Similar';
    }catch(e){
        container.innerHTML='<p style="color:#c00;font-size:.85rem">Failed to load similar results.</p>';
        btn.textContent='🔍 Find Similar';
    }finally{btn.disabled=false}
}
```

**UX behavior**:
- Click "Find Similar" → loads and displays similar promotions below the current card
- Click again ("Hide Similar") → collapses the similar results
- Shows up to 5 similar promotions as mini-cards with repo name, channel, headline, and top 3 tags
- Error state shows a red message, reverts button text

## What NOT to do

- Do **NOT** modify `search.go` — MLT has its own handler
- Do **NOT** use Solr's MLT request handler (`/mlt`) — use the MLT search component via `/select` with `mlt=true` which is more flexible
- Do **NOT** add MLT results to the main search response — it's a separate on-demand API call

## Verification

```powershell
go build ./...
```

After starting the server with some indexed promotions:

```powershell
# Test MLT API directly
$r = Invoke-RestMethod -Uri "http://localhost:8080/api/mlt?id=https://github.com/some/repo" -Method GET
$r | ConvertTo-Json -Depth 3
# Should return {"results": [...], "count": N}

# Test empty id
$r = Invoke-RestMethod -Uri "http://localhost:8080/api/mlt?id=" -Method GET
# Should return 400 error
```

Frontend:
1. Search or browse to see result cards
2. Expand a card → scroll to bottom → see "🔍 Find Similar" button
3. Click it → should load and display similar promotions
4. Click again → should collapse
