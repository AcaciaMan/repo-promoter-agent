# Prompt: Add Search Endpoint and Update Frontend

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. Phase 2 implementation is nearly complete:

- `internal/github/client.go` — fetches real repo data (prompt 09)
- `internal/store/store.go` — SQLite + FTS persistence (prompt 10)
- `internal/handler/generate.go` — evolved handler with GitHub + storage (prompt 11)

This prompt adds the **search endpoint** and updates the **frontend** to support both generation with a real URL input and search/browsing of stored promotions.

## Current project state

```
cmd/server/main.go
internal/agent/client.go
internal/github/client.go
internal/handler/generate.go    # POST /api/generate (done — prompt 11)
internal/store/store.go         # has Search(ctx, query, limit) and List(ctx, limit) methods
static/index.html               # Phase 1 test page — needs update
```

### Store methods available

```go
func (s *Store) Search(ctx context.Context, query string, limit int) ([]Promotion, error)
func (s *Store) List(ctx context.Context, limit int) ([]Promotion, error)
```

## Your task — two parts

### Part 1: `GET /api/search` handler

Create `internal/handler/search.go`:

```go
type SearchHandler struct {
    store *store.Store
}
func NewSearchHandler(store *store.Store) *SearchHandler
```

**Behavior**:

1. Accept `GET /api/search?q=...&limit=...`.
2. If `q` is empty or missing, call `store.List(ctx, limit)` to return recent promotions.
3. If `q` is provided, call `store.Search(ctx, q, limit)`.
4. Default `limit` to 20, cap at 100.
5. Return `200 OK` with:
   ```json
   {
     "results": [ /* array of Promotion objects */ ],
     "count": 5
   }
   ```
6. If `GET` method is wrong, return `405`.

**Wire into `main.go`**:
```go
mux.Handle("/api/search", handler.NewSearchHandler(store))
```

### Part 2: Updated `static/index.html`

Replace the Phase 1 test page with a **two-section page**:

#### Section 1: Generate

- **Text input** for GitHub repo URL (placeholder: "https://github.com/owner/repo").
- **Optional dropdowns/fields** for `target_channel` (select: twitter, linkedin, general) and `target_audience` (text input).
- **"Generate" button**.
- Calls `POST /api/generate` with `{"repo_url": "...", "target_channel": "...", "target_audience": "..."}`.
- Displays the result as **structured cards**, not raw JSON:
  - Headline and summary at top.
  - Key benefits as a bullet list.
  - Twitter posts as individual copyable items.
  - LinkedIn post as a copyable text block.
  - Call to action highlighted.
  - Tags as pill/badge elements.
  - Each content section gets a small "Copy" button.
- Still keep a "Show raw JSON" toggle for debugging.

#### Section 2: Search

- **Search input** with a "Search" button.
- Calls `GET /api/search?q=...`.
- Displays results as **compact cards** showing:
  - Repo name (linked to GitHub).
  - Headline.
  - First tweet preview.
  - Tags as pills.
  - Created date.
  - "Expand" button to show full content.

#### Design

- Single HTML file, inline CSS and JS. No CDN, no build tools.
- Clean, readable layout. Two clear sections separated visually.
- Responsive enough to be usable (simple max-width container is fine).
- Visually distinguish "Generate" section from "Search" section.

## Deliverables

1. **`internal/handler/search.go`** — full Go file.
2. **Updated `cmd/server/main.go`** — add search handler route. Show the full file.
3. **Updated `static/index.html`** — full replacement of the Phase 1 page.
4. **Verification** — `go build ./...` must succeed.

## Constraints

- Standard library only for the Go handler (no routing library).
- No external frontend dependencies (no CDN, no npm).
- The page should work as a single HTML file served by Go's `http.FileServer`.
- Copy buttons should use `navigator.clipboard.writeText()`.
- Keep the HTML under ~300 lines. Prioritize functionality over beauty.
- Search should work with empty query (shows recent promotions as a browse view).
