# Prompt: Extend SQLite Schema for Traffic Metrics

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. I'm adding GitHub traffic metrics (views & clones) for AcaciaMan repos.

This is **Phase 1, Step 3**. The previous prompts added:

- Prompt 16: GitHub token support, `RepoOwner()` helper.
- Prompt 17: `FetchTrafficMetrics()` method, `TrafficMetrics` type, `HasToken()`.

The full intent document is at `docs/intent-for-views-clones.md`.

## Current store (`internal/store/store.go`)

The `promotions` table schema:

```sql
CREATE TABLE IF NOT EXISTS promotions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    repo_url TEXT NOT NULL,
    repo_name TEXT NOT NULL,
    headline TEXT NOT NULL,
    summary TEXT NOT NULL,
    key_benefits TEXT NOT NULL,
    tags TEXT NOT NULL,
    twitter_posts TEXT NOT NULL,
    linkedin_post TEXT NOT NULL,
    call_to_action TEXT NOT NULL,
    target_channel TEXT NOT NULL DEFAULT '',
    target_audience TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

The `Promotion` struct:

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

Key store methods:

- `New(dbPath string)` — opens DB, runs `CREATE TABLE IF NOT EXISTS` and `CREATE VIRTUAL TABLE IF NOT EXISTS` for FTS.
- `Save(ctx, *Promotion)` — deletes old record by `repo_url`, inserts new one with `RETURNING id, created_at`.
- `Search(ctx, query, limit)` — FTS search, returns `[]Promotion`.
- `List(ctx, limit)` — recent promotions, returns `[]Promotion`.
- `scanPromotions(rows)` — internal helper that scans all columns in a fixed order.

There is **no migration framework** — schema is applied via `db.Exec(schema)` in `New()`.

## TrafficMetrics type (from `internal/github/client.go`, added in prompt 17)

```go
type TrafficMetrics struct {
    Views14dTotal   int `json:"views_14d_total"`
    Views14dUnique  int `json:"views_14d_unique"`
    Clones14dTotal  int `json:"clones_14d_total"`
    Clones14dUnique int `json:"clones_14d_unique"`
}
```

## Your task

Extend the store to persist and retrieve traffic metrics alongside promotions.

### 1. Schema migration for new columns

Since there's no migration framework and the app uses `CREATE TABLE IF NOT EXISTS`, add a **separate migration step** in `New()` that runs **after** the existing schema:

```go
const migration001 = `
ALTER TABLE promotions ADD COLUMN views_14d_total INTEGER NOT NULL DEFAULT 0;
ALTER TABLE promotions ADD COLUMN views_14d_unique INTEGER NOT NULL DEFAULT 0;
ALTER TABLE promotions ADD COLUMN clones_14d_total INTEGER NOT NULL DEFAULT 0;
ALTER TABLE promotions ADD COLUMN clones_14d_unique INTEGER NOT NULL DEFAULT 0;
`
```

**Important:** `ALTER TABLE ADD COLUMN` will fail if the column already exists. Wrap each `ALTER TABLE` in an error check that **ignores** "duplicate column" errors but propagates other errors. A simple approach:

```go
func (s *Store) applyMigrations() error {
    columns := []string{
        "views_14d_total INTEGER NOT NULL DEFAULT 0",
        "views_14d_unique INTEGER NOT NULL DEFAULT 0",
        "clones_14d_total INTEGER NOT NULL DEFAULT 0",
        "clones_14d_unique INTEGER NOT NULL DEFAULT 0",
    }
    for _, col := range columns {
        _, err := s.db.Exec("ALTER TABLE promotions ADD COLUMN " + col)
        if err != nil && !strings.Contains(err.Error(), "duplicate column") {
            return fmt.Errorf("migration failed: %w", err)
        }
    }
    return nil
}
```

Call `applyMigrations()` at the end of `New()`, after the existing `db.Exec(schema)` call.

### 2. Extend the `Promotion` struct

Add four new fields:

```go
Views14dTotal   int `json:"views_14d_total"`
Views14dUnique  int `json:"views_14d_unique"`
Clones14dTotal  int `json:"clones_14d_total"`
Clones14dUnique int `json:"clones_14d_unique"`
```

### 3. Update `Save`

Add the four new columns to the `INSERT` statement and pass the new field values. The columns have `DEFAULT 0`, so old code that doesn't set these fields will get zeros — backward compatible.

### 4. Update `scanPromotions`

Add the four new columns to all `SELECT` queries (in `Search`, `List`, and the FTS join query) and scan them in `scanPromotions`.

**Important:** All three SELECT queries must list the same columns in the same order, because they all use `scanPromotions`. Add the new columns at the end of each SELECT list.

### 5. Do NOT add metrics to FTS

Traffic numbers are integers, not text — they don't belong in the FTS5 virtual table. Leave the FTS table and its triggers unchanged.

## What NOT to do

- Do NOT modify the handler, agent, GitHub client, or frontend.
- Do NOT create a separate metrics table — keep it simple with additional columns on `promotions`.
- Do NOT change the FTS virtual table or its triggers.
- Do NOT remove or rename any existing columns or fields.

## Verification

After implementation:

1. `go build ./...` compiles without errors.
2. **Fresh database:** Starting with no `.db` file creates the table with the new columns.
3. **Existing database:** Starting with an existing `.db` file from before this change successfully adds the new columns via migration. Running a second time doesn't fail (duplicate column errors are ignored).
4. `Save` works with `Views14dTotal` etc. set to 0 (default behavior, backward compatible).
5. `Search` and `List` return promotions with the new fields (all zeros for old records).
6. The JSON output from the API now includes `views_14d_total`, `views_14d_unique`, `clones_14d_total`, `clones_14d_unique` fields (all 0 until the handler wires them in).
