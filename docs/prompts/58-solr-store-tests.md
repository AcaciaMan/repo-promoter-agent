# Prompt: Solr Store — Integration Tests

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. I'm replacing SQLite with **Apache Solr 10** as the sole data store. This is **Phase 2, prompt 2 of 4**.

**Prerequisite**: Prompt 57 is complete — `internal/store/store.go` has been replaced with a Solr-backed implementation. The `Store` struct now uses `net/http` to talk to Solr at `http://localhost:8983` with core `promotions`.

The previous `store_test.go` used SQLite `:memory:` databases. Since Solr requires a running instance, the new tests must use a build tag to avoid breaking CI or normal `go test` runs.

## Current public API (from prompt 57)

```go
func New(solrURL, core string) (*Store, error)       // Pings Solr, returns Store
func (s *Store) Close() error                         // No-op
func (s *Store) Save(ctx, *Promotion) error           // Upsert doc by repo_url as id
func (s *Store) Search(ctx, query, limit) ([]Promotion, error) // edismax full-text search
func (s *Store) List(ctx, limit) ([]Promotion, error) // All docs sorted by created_at desc
```

## Your task

**Replace** `internal/store/store_test.go` with integration tests for the Solr-backed store. Tests require a running Solr instance.

## Requirements

### 1. Build tag

The file must start with:

```go
//go:build integration

package store
```

This ensures `go test ./...` skips these tests unless explicitly requested:

```powershell
go test -tags integration ./internal/store/...
```

### 2. Test helper — `newTestStore`

```go
func newTestStore(t *testing.T) *Store {
    t.Helper()
    solrURL := os.Getenv("SOLR_URL")
    if solrURL == "" {
        solrURL = "http://localhost:8983"
    }
    solrCore := os.Getenv("SOLR_CORE")
    if solrCore == "" {
        solrCore = "promotions"
    }
    st, err := New(solrURL, solrCore)
    if err != nil {
        t.Fatalf("failed to create Solr store: %v", err)
    }
    return st
}
```

### 3. Test helper — `cleanupSolr`

Delete all documents from the core to ensure test isolation. Call this at the beginning of each test:

```go
func cleanupSolr(t *testing.T, st *Store) {
    t.Helper()
    // POST {"delete":{"query":"*:*"},"commit":{}} to /solr/{core}/update
    // Fail the test if the cleanup fails.
}
```

### 4. Keep `samplePromotion` and `sampleAnalysisJSON` helpers

Port them from the old test file — they are still valid:

```go
func samplePromotion() *Promotion {
    return &Promotion{
        RepoURL:        "https://github.com/testowner/testrepo",
        RepoName:       "testrepo",
        Headline:       "Test Headline",
        Summary:        "Test Summary",
        KeyBenefits:    []string{"benefit1", "benefit2"},
        Tags:           []string{"go", "testing"},
        TwitterPosts:   []string{"tweet1", "tweet2", "tweet3"},
        LinkedInPost:   "LinkedIn post content",
        CallToAction:   "Star the repo!",
        TargetChannel:  "general",
        TargetAudience: "Go developers",
    }
}

func sampleAnalysisJSON() json.RawMessage {
    return json.RawMessage(`{
        "primary_value_proposition": "Helps developers test efficiently.",
        "ideal_audience": ["Go developers", "TDD practitioners"],
        "key_features": ["Fast execution", "Simple API"]
    }`)
}
```

### 5. Test cases

Implement these test functions. Each test must call `cleanupSolr` at the start to ensure isolation:

#### `TestSave_Basic`
- Save a `samplePromotion()`
- Verify no error
- Verify `p.CreatedAt` is non-zero after save

#### `TestSave_WithAnalysis`
- Save a promotion with `AnalysisJSON = sampleAnalysisJSON()`
- List it back
- Verify `AnalysisJSON` is non-nil and valid JSON
- Verify it contains `"primary_value_proposition"`

#### `TestSave_WithoutAnalysis`
- Save a promotion without `AnalysisJSON` (nil)
- List it back
- Verify `AnalysisJSON` is nil

#### `TestSave_Upsert`
- Save a promotion with headline "Original"
- Save again with same `RepoURL` but headline "Updated"
- List all — should return exactly 1 result
- Verify headline is "Updated"

#### `TestSearch_FullText`
- Save a promotion with repo_name "testrepo" and summary containing "kubernetes deployment"
- Search for "kubernetes"
- Verify 1 result returned with correct repo_name

#### `TestSearch_EmptyQuery`
- Search with query ""
- Verify empty result set (not an error)

#### `TestSearch_NoMatch`
- Clean core (no documents)
- Search for "nonexistent"
- Verify empty result set

#### `TestList_OrderByDate`
- Save promotion A with `CreatedAt` at `2026-03-14T10:00:00Z`
- Save promotion B with `CreatedAt` at `2026-03-14T12:00:00Z` (different `RepoURL`)
- List with limit 10
- Verify 2 results, B comes first (more recent)

#### `TestList_RespectsLimit`
- Save 3 promotions (different `RepoURL` values)
- List with limit 2
- Verify exactly 2 results returned

#### `TestList_Empty`
- Clean core
- List with limit 10
- Verify empty result set (not nil — `[]Promotion{}`)

#### `TestSearch_SpecialCharacters`
- Save a promotion
- Search for `C++ (advanced)` — contains Solr special chars
- Verify no error (query is sanitized, may return 0 or 1 results — just don't crash)

#### `TestSave_WithTrafficMetrics`
- Save a promotion with non-zero `Stars`, `Forks`, `Watchers`, `Views14dTotal`, `Views14dUnique`, `Clones14dTotal`, `Clones14dUnique`
- List it back
- Verify all integer fields round-trip correctly

### 6. No SQLite references

The file must not import `database/sql` or `modernc.org/sqlite`. Only `os`, `context`, `encoding/json`, `testing`, `time`.

## Running the tests

```powershell
# Start Solr first (if not already running)
cd C:\Tools\solr-10.0.0
bin\solr.cmd start

# Run integration tests
cd C:\work\GitHub\repo-promoter-agent
go test -tags integration -v ./internal/store/...
```

## Verification

- [ ] `go test -tags integration -v ./internal/store/...` — all tests pass
- [ ] `go test ./internal/store/...` — **no tests run** (build tag prevents it)
- [ ] Each test is independent — can run in any order due to `cleanupSolr`

## Notes

- Solr commits are synchronous (we use `?commit=true`), so documents are immediately searchable after `Save`.
- The `cleanupSolr` helper uses `delete-by-query` with `*:*` — this clears everything in the core.
- `Promotion.ID` will be `0` in all tests — this is expected since Solr uses string IDs (`repo_url`).
- Multi-valued fields (`KeyBenefits`, `Tags`, `TwitterPosts`) should round-trip as proper `[]string` slices, not JSON strings.
