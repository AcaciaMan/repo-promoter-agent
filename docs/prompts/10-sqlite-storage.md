# Prompt: Implement SQLite Storage with Full-Text Search

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. Phase 1 is complete (hardcoded data → agent → promotional JSON). I'm now building Phase 2.

The previous prompt (09) added a GitHub API client. This prompt adds **SQLite persistence with Full-Text Search** so generated promotional content is stored and searchable.

## Current project state

```
cmd/server/main.go
internal/agent/client.go        # Gradient agent client
internal/github/client.go       # GitHub API client (prompt 09)
internal/handler/generate.go    # POST /api/generate handler
static/index.html
```

### Output JSON structure (what the agent returns)

```json
{
  "repo_url": "string",
  "repo_name": "string",
  "headline": "string",
  "summary": "string",
  "key_benefits": ["string"],
  "tags": ["string"],
  "twitter_posts": ["string"],
  "linkedin_post": "string",
  "call_to_action": "string"
}
```

## Your task

Create `internal/store/store.go` — a SQLite-backed store that persists promotional content and supports full-text search.

## Requirements

### 1. Database schema

One main table and one FTS5 virtual table:

```sql
CREATE TABLE IF NOT EXISTS promotions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    repo_url TEXT NOT NULL,
    repo_name TEXT NOT NULL,
    headline TEXT NOT NULL,
    summary TEXT NOT NULL,
    key_benefits TEXT NOT NULL,   -- JSON array stored as text
    tags TEXT NOT NULL,           -- JSON array stored as text
    twitter_posts TEXT NOT NULL,  -- JSON array stored as text
    linkedin_post TEXT NOT NULL,
    call_to_action TEXT NOT NULL,
    target_channel TEXT NOT NULL DEFAULT '',
    target_audience TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE VIRTUAL TABLE IF NOT EXISTS promotions_fts USING fts5(
    repo_name,
    headline,
    summary,
    tags,
    linkedin_post,
    call_to_action,
    content=promotions,
    content_rowid=id
);
```

Plus triggers to keep FTS in sync with inserts/updates/deletes.

### 2. Store struct

```go
type Store struct {
    db *sql.DB
}

func New(dbPath string) (*Store, error)    // opens DB, runs schema migration
func (s *Store) Close() error
```

- Use `modernc.org/sqlite` (pure Go, no CGO) OR `github.com/mattn/go-sqlite3` (CGO). Recommend one and explain the trade-off.
- Run the schema creation in `New()` so the database is ready immediately.

### 3. Promotion type

Define a Go struct that represents a stored promotion:

```go
type Promotion struct {
    ID             int64     `json:"id"`
    RepoURL        string    `json:"repo_url"`
    RepoName       string    `json:"repo_name"`
    Headline       string    `json:"headline"`
    Summary        string    `json:"summary"`
    KeyBenefits    []string  `json:"key_benefits"`
    Tags           []string  `json:"tags"`
    TwitterPosts   []string  `json:"twitter_posts"`
    LinkedInPost   string    `json:"linkedin_post"`
    CallToAction   string    `json:"call_to_action"`
    TargetChannel  string    `json:"target_channel"`
    TargetAudience string    `json:"target_audience"`
    CreatedAt      time.Time `json:"created_at"`
}
```

### 4. Methods

**Save** — insert a new promotion:

```go
func (s *Store) Save(ctx context.Context, p *Promotion) error
```

- Inserts into `promotions` table.
- The FTS trigger handles the virtual table.
- Sets `p.ID` and `p.CreatedAt` from the insert result.

**Search** — full-text search:

```go
func (s *Store) Search(ctx context.Context, query string, limit int) ([]Promotion, error)
```

- Uses `promotions_fts` to search across repo_name, headline, summary, tags, linkedin_post, call_to_action.
- Returns results ordered by relevance (FTS rank).
- Applies `limit` (default to 20 if 0).
- Returns empty slice (not nil) if no matches.

**List** — get recent promotions:

```go
func (s *Store) List(ctx context.Context, limit int) ([]Promotion, error)
```

- Returns most recent promotions ordered by `created_at DESC`.
- For the browse/history view.

### 5. JSON array handling

`key_benefits`, `tags`, and `twitter_posts` are `[]string` in Go but stored as JSON text in SQLite. Create helpers to marshal/unmarshal these cleanly during insert and scan.

### 6. FTS query sanitization

The `query` string comes from user input. Sanitize it to prevent FTS5 syntax errors:
- Escape or strip special FTS5 characters (`"`, `*`, `(`, `)`, etc.).
- Or wrap each token in double quotes.
- Test that empty queries and single-character queries don't crash.

## Deliverables

1. **`internal/store/store.go`** — full, working Go code with all types and methods.
2. **`go get` command** — for whichever SQLite driver you recommend.
3. **Update to `main.go`** — show how to add store initialization (open DB, defer close). The handler wiring happens in prompt 11.

## Constraints

- The database file should be configurable via env var (e.g., `DB_PATH`, default `promotions.db`).
- Keep it simple — no migrations framework, just `CREATE TABLE IF NOT EXISTS`.
- The `Save` method should accept the agent's output (which is `json.RawMessage` in the handler) easily. Show how the handler will convert between `json.RawMessage` and `Promotion`.
- No tests in this prompt.
