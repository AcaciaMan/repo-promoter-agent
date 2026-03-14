# Prompt: Solr Store — Replace SQLite Store with Solr Implementation

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. Phase 1 (prompts 54–56) is complete — Solr 10 is running locally at `http://localhost:8983` with a `promotions` core, all 20 schema fields defined, and copy-field rules verified.

This prompt **replaces the entire SQLite store** (`internal/store/store.go`) with a Solr-backed implementation. There is no rollback path — SQLite is gone after this.

## Current `internal/store/store.go`

The file currently contains:

- `Promotion` struct (18+ fields with JSON tags) — **keep this unchanged**
- `Store` struct wrapping `*sql.DB` — **replace with Solr HTTP client**
- `New(dbPath string) (*Store, error)` — opens SQLite, runs schema/migrations
- `Close() error` — closes DB connection
- `Save(ctx, *Promotion) error` — delete-then-insert by `repo_url`, sets `p.ID` and `p.CreatedAt`
- `Search(ctx, query, limit) ([]Promotion, error)` — FTS5 full-text search
- `List(ctx, limit) ([]Promotion, error)` — returns recent promotions sorted by `created_at DESC`
- Helper functions: `scanPromotions`, `parseTime`, `marshalJSON`, `unmarshalJSONOrEmpty`, `sanitizeFTSQuery`

## Solr schema (already created in Phase 1)

The `promotions` core has these fields:

| Field | Solr Type | Multi-Valued | Notes |
|---|---|---|---|
| `id` | string | no | Unique key — stores `repo_url` value |
| `repo_url` | string | no | Indexed for exact match |
| `repo_name` | text_general | no | Full-text searchable |
| `headline` | text_general | no | Full-text searchable |
| `summary` | text_general | no | Full-text searchable |
| `key_benefits` | text_general | yes | Array of benefits |
| `tags` | string | yes | Exact-match for faceting |
| `twitter_posts` | text_general | yes | Array of tweets |
| `linkedin_post` | text_general | no | Full-text searchable |
| `call_to_action` | text_general | no | Full-text searchable |
| `target_channel` | string | no | Exact filter |
| `target_audience` | text_general | no | Full-text searchable |
| `created_at` | pdate | no | Sortable date |
| `stars`, `forks`, `watchers` | pint | no | Metrics |
| `views_14d_total`, `views_14d_unique`, `clones_14d_total`, `clones_14d_unique` | pint | no | Traffic |
| `analysis_json` | string | no | Stored, not indexed — raw JSON blob |

Copy fields: 9 text fields → `_text_` catch-all for default search.

## Your task

**Completely replace** the contents of `internal/store/store.go`. Keep the `Promotion` struct exactly as-is; replace everything else with a Solr-backed implementation.

## Requirements

### 1. Keep the `Promotion` struct unchanged

The `Promotion` struct with all its fields and JSON tags must remain exactly as it is today. It is used by handlers and the rest of the codebase.

### 2. Replace `Store` struct

```go
// Store is a Solr-backed store for promotional content.
type Store struct {
    baseURL string     // e.g. "http://localhost:8983"
    core    string     // e.g. "promotions"
    client  *http.Client
}
```

### 3. Replace `New` constructor

```go
func New(solrURL, core string) (*Store, error)
```

- Accepts `solrURL` (e.g. `"http://localhost:8983"`) and `core` (e.g. `"promotions"`)
- Creates an `http.Client` with a sensible timeout (e.g. 30 seconds)
- **Pings Solr** to verify connectivity: `GET {solrURL}/solr/{core}/admin/ping`
- Returns error if Solr is not reachable or the core doesn't exist
- Note: signature changes from `New(dbPath)` to `New(solrURL, core)` — callers will be updated in a later prompt

### 4. Replace `Close`

```go
func (s *Store) Close() error
```

- No-op — the HTTP client is stateless, no connection to close
- Return `nil`

### 5. Replace `Save`

```go
func (s *Store) Save(ctx context.Context, p *Promotion) error
```

Behavior:
- POST the Promotion as a JSON document to `{baseURL}/solr/{core}/update/json/docs?commit=true`
- The document must include `"id": p.RepoURL` — Solr uses this as the unique key, so re-posting the same `repo_url` replaces the previous document (upsert)
- Map each Promotion field to its Solr field name:
  - String fields map directly
  - `[]string` fields (`KeyBenefits`, `Tags`, `TwitterPosts`) are sent as JSON arrays — Solr handles multiValued fields natively
  - `AnalysisJSON` (`json.RawMessage`): if non-nil, send as a string value; if nil, omit or send empty string
  - `CreatedAt`: if zero, set to `time.Now()` before sending; format as RFC3339 for Solr `pdate`
  - Integer fields (`Stars`, `Forks`, etc.) map directly
- After successful POST, set `p.CreatedAt` to the value that was sent (so the handler can return it)
- The `p.ID` field: Solr doesn't use integer IDs. Set `p.ID = 0` or leave it — it's not critical for the Solr implementation but keep the field in the struct for JSON serialization
- Check the Solr response for errors (status != 0 in the Solr JSON response body)

Document structure to POST:
```json
{
  "id": "https://github.com/owner/repo",
  "repo_url": "https://github.com/owner/repo",
  "repo_name": "repo",
  "headline": "...",
  "summary": "...",
  "key_benefits": ["benefit1", "benefit2"],
  "tags": ["tag1", "tag2"],
  "twitter_posts": ["tweet1", "tweet2"],
  "linkedin_post": "...",
  "call_to_action": "...",
  "target_channel": "general",
  "target_audience": "...",
  "created_at": "2026-03-14T12:00:00Z",
  "stars": 10,
  "forks": 2,
  "watchers": 5,
  "views_14d_total": 100,
  "views_14d_unique": 50,
  "clones_14d_total": 20,
  "clones_14d_unique": 10,
  "analysis_json": "{\"key\":\"value\"}"
}
```

### 6. Replace `Search`

```go
func (s *Store) Search(ctx context.Context, query string, limit int) ([]Promotion, error)
```

Behavior:
- Default limit to 20 if <= 0
- If `query` is empty after trimming, return empty `[]Promotion{}`
- Sanitize the query for Solr special characters (escape: `+ - & | ! ( ) { } [ ] ^ " ~ * ? : \ /`)
- GET `{baseURL}/solr/{core}/select` with query parameters:
  - `q` = sanitized query
  - `defType` = `edismax` (user-friendly query parser)
  - `qf` = `repo_name headline summary key_benefits tags twitter_posts linkedin_post call_to_action target_audience` (query fields — the text fields to search)
  - `rows` = limit
  - `sort` = `score desc`
  - `wt` = `json`
  - `fl` = `*` (return all stored fields)
- Parse the Solr JSON response and map each document back to a `Promotion` struct
- Return `[]Promotion{}` (not nil) if no results

### 7. Replace `List`

```go
func (s *Store) List(ctx context.Context, limit int) ([]Promotion, error)
```

Behavior:
- Default limit to 20 if <= 0
- GET `{baseURL}/solr/{core}/select` with:
  - `q` = `*:*`
  - `rows` = limit
  - `sort` = `created_at desc`
  - `wt` = `json`
  - `fl` = `*`
- Parse and return `[]Promotion{}`

### 8. Solr response parsing helpers

Create internal helper functions:

- `parseSolrDocs(body []byte) ([]Promotion, error)` — parses the Solr JSON response envelope (`response.docs`) and maps each document to a `Promotion`
- When mapping Solr docs to `Promotion`:
  - Solr returns multiValued fields as JSON arrays — map directly to `[]string`
  - `analysis_json` comes back as a string — convert to `json.RawMessage`; if empty or missing, set to `nil`
  - `created_at` comes back as a date string — parse with `time.Parse(time.RFC3339, ...)`
  - `id` from Solr is the `repo_url`; set `Promotion.ID = 0` (no integer ID in Solr)
  - Integer fields may come as `float64` from JSON unmarshaling — handle the conversion

### 9. Solr query sanitizer

Replace `sanitizeFTSQuery` with a Solr-appropriate version. Solr special characters to escape with backslash: `+ - && || ! ( ) { } [ ] ^ " ~ * ? : \ /`

```go
func sanitizeSolrQuery(query string) string
```

### 10. Remove all SQLite-specific code

Remove from the file:
- The `import _ "modernc.org/sqlite"` line
- The `import "database/sql"` line
- The `const schema` SQL string
- The `applyMigrations` function
- The `scanPromotions` function
- The `marshalJSON` and `unmarshalJSONOrEmpty` helpers
- The old `sanitizeFTSQuery` function

### 11. Add required imports

The new implementation should only need:
- `"bytes"`, `"context"`, `"encoding/json"`, `"fmt"`, `"io"`, `"net/http"`, `"net/url"`, `"strings"`, `"time"`

No new external dependencies — use stdlib `net/http` only.

## Verification

After replacing the file:

```powershell
go build ./internal/store/...
```

This will compile the store package in isolation. It is expected that `cmd/server/main.go` and handler files **will not compile** yet — they still reference `store.New(dbPath)` with the old single-argument signature. That will be fixed in prompt 59.

## Notes

- The `Promotion.ID` field (`int64`) is kept for JSON serialization compatibility but will always be `0` in the Solr implementation. This is fine — the frontend doesn't rely on it for anything critical.
- Solr `pint` fields may deserialize as `float64` through Go's `json.Unmarshal` into `interface{}`. The doc parser must handle this.
- `edismax` query parser gives good out-of-the-box behavior for user queries: supports phrase matching, auto-relaxation, and doesn't require special syntax.
