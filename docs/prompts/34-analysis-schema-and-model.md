# Prompt: Add analysis_json Column and Update Store Functions

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. In Phase 1 (prompts 31–33) I created the Analysis Agent client with types, prompt template, and tests. Now I'm starting **Phase 2** — extending the data model and SQLite persistence to store analysis JSON alongside promotions.

The full intent document is at `docs/intent-for-analysis-agent.md` — read it for high-level context.

## Current state

### `internal/store/store.go`

**Promotion struct** (current):
```go
type Promotion struct {
    ID              int64     `json:"id"`
    RepoURL         string    `json:"repo_url"`
    RepoName        string    `json:"repo_name"`
    Headline        string    `json:"headline"`
    Summary         string    `json:"summary"`
    KeyBenefits     []string  `json:"key_benefits"`
    Tags            []string  `json:"tags"`
    TwitterPosts    []string  `json:"twitter_posts"`
    LinkedInPost    string    `json:"linkedin_post"`
    CallToAction    string    `json:"call_to_action"`
    TargetChannel   string    `json:"target_channel"`
    TargetAudience  string    `json:"target_audience"`
    CreatedAt       time.Time `json:"created_at"`
    Views14dTotal   int       `json:"views_14d_total"`
    Views14dUnique  int       `json:"views_14d_unique"`
    Clones14dTotal  int       `json:"clones_14d_total"`
    Clones14dUnique int       `json:"clones_14d_unique"`
}
```

**Schema** — `promotions` table with 15 columns + FTS5 virtual table + sync triggers. Traffic metric columns were added via `applyMigrations()` using `ALTER TABLE ADD COLUMN` with `strings.Contains(err.Error(), "duplicate column")` guard.

**Key functions:**
- `Save(ctx, *Promotion)` — deletes old row for same `repo_url`, inserts new row with `RETURNING id, created_at`.
- `Search(ctx, query, limit)` — FTS5 search joining `promotions_fts` with `promotions`.
- `List(ctx, limit)` — recent promotions ordered by `created_at DESC`.
- `scanPromotions(rows)` — shared helper that scans all 17 columns from query results.

### What needs to change

The `Promotion` struct and all store functions need an `analysis_json` field. The FTS5 index does **not** need to include analysis.

## Your task

### 1. Add `AnalysisJSON` field to the `Promotion` struct

Add a new field to the `Promotion` struct:

```go
AnalysisJSON json.RawMessage `json:"analysis"`
```

**Why `json.RawMessage`:**
- It serializes to the frontend as a nested JSON object (not a string), so the UI can access `response.analysis.primary_value_proposition` directly.
- It deserializes from the DB column (TEXT) naturally.
- When the column is `NULL` (legacy rows or no analysis), it marshals to `null` in JSON.
- No need to import `agent.AnalysisOutput` in the store package — keeps packages decoupled.

Place it after `Clones14dUnique` (last current field).

### 2. Add migration for `analysis_json` column

In `applyMigrations()`, add one more column to the migration list:

```go
columns := []string{
    "views_14d_total INTEGER NOT NULL DEFAULT 0",
    "views_14d_unique INTEGER NOT NULL DEFAULT 0",
    "clones_14d_total INTEGER NOT NULL DEFAULT 0",
    "clones_14d_unique INTEGER NOT NULL DEFAULT 0",
    "analysis_json TEXT DEFAULT NULL",
}
```

Note: unlike the traffic metrics columns, `analysis_json` is nullable (no `NOT NULL`) and defaults to `NULL`. This is intentional — legacy rows won't have analysis data.

### 3. Update `Save()` to persist `analysis_json`

Update the INSERT statement to include the new column:

```go
const q = `INSERT INTO promotions
    (repo_url, repo_name, headline, summary, key_benefits, tags, twitter_posts,
     linkedin_post, call_to_action, target_channel, target_audience,
     views_14d_total, views_14d_unique, clones_14d_total, clones_14d_unique,
     analysis_json)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    RETURNING id, created_at`
```

Add `p.AnalysisJSON` (which may be `nil`) as the last bind parameter in the `QueryRowContext` call. When `json.RawMessage` is `nil`, SQLite receives `NULL`.

**Important:** Handle the case where `AnalysisJSON` is nil. Pass it directly — `database/sql` will convert a nil `[]byte` to SQL `NULL`. If you get issues, wrap it:

```go
var analysisVal interface{}
if p.AnalysisJSON != nil {
    analysisVal = string(p.AnalysisJSON)
} else {
    analysisVal = nil
}
```

### 4. Update `scanPromotions()` to load `analysis_json`

In the `scanPromotions` helper, add a `sql.NullString` variable to scan the nullable column:

```go
var analysisJSON sql.NullString
```

Add `&analysisJSON` to the `rows.Scan()` call after `&p.Clones14dUnique`.

After scanning, convert back to `json.RawMessage`:

```go
if analysisJSON.Valid {
    p.AnalysisJSON = json.RawMessage(analysisJSON.String)
}
```

If `analysisJSON.Valid` is false (NULL in DB), `p.AnalysisJSON` stays nil, which marshals to `null` in JSON — exactly what we want.

### 5. Update `Search()` and `List()` SQL queries to include `analysis_json`

Add `p.analysis_json` as the **last** selected column in both queries.

**Search query:**
```sql
SELECT p.id, p.repo_url, p.repo_name, p.headline, p.summary,
    p.key_benefits, p.tags, p.twitter_posts, p.linkedin_post,
    p.call_to_action, p.target_channel, p.target_audience, p.created_at,
    p.views_14d_total, p.views_14d_unique, p.clones_14d_total, p.clones_14d_unique,
    p.analysis_json
    FROM promotions_fts fts
    JOIN promotions p ON p.id = fts.rowid
    WHERE promotions_fts MATCH ?
    ORDER BY rank
    LIMIT ?
```

**List query:**
```sql
SELECT id, repo_url, repo_name, headline, summary,
    key_benefits, tags, twitter_posts, linkedin_post,
    call_to_action, target_channel, target_audience, created_at,
    views_14d_total, views_14d_unique, clones_14d_total, clones_14d_unique,
    analysis_json
    FROM promotions
    ORDER BY created_at DESC
    LIMIT ?
```

### 6. Do NOT change FTS5

The `analysis_json` column is **not** added to the FTS5 virtual table, triggers, or schema. Analysis data is not searchable — this is intentional per the intent doc.

## What NOT to do

- Do NOT modify the agent client, handler, or frontend files.
- Do NOT add `analysis_json` to the FTS5 virtual table or triggers.
- Do NOT change the existing `schema` const — the migration handles the new column.
- Do NOT write tests yet (that's the next prompt).
- Do NOT add `encoding/json` to imports if it's already imported (it is — used by `marshalJSON`).
- Do NOT add `database/sql` to imports if it's already imported (check — you may need to add it for `sql.NullString`).

## Verification

1. `go build ./...` compiles without errors.
2. The `Promotion` struct has an `AnalysisJSON json.RawMessage` field with JSON tag `"analysis"`.
3. `applyMigrations()` adds `analysis_json TEXT DEFAULT NULL` column.
4. `Save()` persists `analysis_json` (nil → NULL, non-nil → TEXT).
5. `Search()` and `List()` queries select `analysis_json`.
6. `scanPromotions()` scans `analysis_json` using `sql.NullString` and converts to `json.RawMessage`.
7. FTS5 index, virtual table, and triggers are unchanged.
8. The app starts successfully with the existing database (migration adds the column if missing).
