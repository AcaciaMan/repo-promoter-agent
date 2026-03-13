# Prompt: Unit Tests for Analysis JSON Persistence in Store

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. In prompt 34, I extended the `Promotion` struct and store functions to persist `analysis_json` as a nullable TEXT column in SQLite.

Now I need tests to verify that analysis data is correctly saved, retrieved, and handled when NULL.

The full intent document is at `docs/intent-for-analysis-agent.md` — read it for high-level context.

## Current state

### `internal/store/store.go` (after prompt 34)

- `Promotion` struct has `AnalysisJSON json.RawMessage` with JSON tag `"analysis"`.
- `applyMigrations()` adds `analysis_json TEXT DEFAULT NULL` column.
- `Save()` persists `analysis_json` (nil → NULL, non-nil → stored as TEXT).
- `Search()` and `List()` queries include `analysis_json`.
- `scanPromotions()` uses `sql.NullString` to handle nullable column, converts to `json.RawMessage`.

### Existing tests

After prompt 33, there is `internal/agent/analysis_test.go` for the analysis client. There are no store tests yet.

## Your task

Create `internal/store/store_test.go` with tests using an **in-memory SQLite database** (`:memory:`).

### Test helper

Create a helper that opens a fresh store for each test:

```go
func newTestStore(t *testing.T) *Store {
    t.Helper()
    st, err := New(":memory:")
    if err != nil {
        t.Fatalf("failed to create test store: %v", err)
    }
    t.Cleanup(func() { st.Close() })
    return st
}
```

Also create a helper to build a standard promotion:

```go
func samplePromotion() *Promotion {
    return &Promotion{
        RepoURL:      "https://github.com/testowner/testrepo",
        RepoName:     "testrepo",
        Headline:     "Test Headline",
        Summary:      "Test Summary",
        KeyBenefits:  []string{"benefit1", "benefit2"},
        Tags:         []string{"go", "testing"},
        TwitterPosts: []string{"tweet1", "tweet2", "tweet3"},
        LinkedInPost: "LinkedIn post content",
        CallToAction: "Star the repo!",
        TargetChannel:  "general",
        TargetAudience: "Go developers",
    }
}
```

And a sample analysis JSON:

```go
func sampleAnalysisJSON() json.RawMessage {
    return json.RawMessage(`{
        "repo_url": "https://github.com/testowner/testrepo",
        "repo_name": "testrepo",
        "primary_value_proposition": "Helps developers test efficiently.",
        "ideal_audience": ["Go developers", "TDD practitioners"],
        "key_features": ["Fast execution", "Simple API"],
        "differentiators": ["Minimal dependencies"],
        "risk_or_limitations": ["Early-stage project"],
        "social_proof_signals": ["Modest traction"],
        "recommended_positioning_angle": ["Lightweight testing"]
    }`)
}
```

### Required test cases

#### 1. `TestSave_WithAnalysis`

- Create a promotion with `AnalysisJSON` set to `sampleAnalysisJSON()`.
- Call `Save()`.
- Assert: no error, `p.ID > 0`, `p.CreatedAt` is non-zero.
- Retrieve the promotion via `List(ctx, 1)`.
- Assert: returned promotion has `AnalysisJSON` that is non-nil and contains `"primary_value_proposition"`.

#### 2. `TestSave_WithoutAnalysis`

- Create a promotion with `AnalysisJSON` left as nil (zero value).
- Call `Save()`.
- Assert: no error.
- Retrieve via `List(ctx, 1)`.
- Assert: returned promotion has `AnalysisJSON` that is nil.
- Marshal the promotion to JSON and verify the `"analysis"` field is `null` (not missing, not empty string).

#### 3. `TestSave_ReplacesOldPromotion_WithAnalysis`

- Save a promotion for `repo_url = "https://github.com/testowner/testrepo"` **without** analysis.
- Save another promotion for the **same** `repo_url` **with** analysis.
- Call `List(ctx, 10)`.
- Assert: only 1 promotion exists (old one deleted), and it has `AnalysisJSON` set.

#### 4. `TestSearch_ReturnsAnalysis`

- Save a promotion with analysis.
- Call `Search(ctx, "testrepo", 10)`.
- Assert: result has 1 item, and `AnalysisJSON` is non-nil and valid JSON.

#### 5. `TestSearch_ReturnsNullAnalysis`

- Save a promotion without analysis.
- Call `Search(ctx, "testrepo", 10)`.
- Assert: result has 1 item, and `AnalysisJSON` is nil.

#### 6. `TestList_MixedAnalysis`

- Save promotion A (repo "alpha") with analysis.
- Save promotion B (repo "beta") without analysis.
- Call `List(ctx, 10)`.
- Assert: 2 results returned. One has `AnalysisJSON != nil`, the other has `AnalysisJSON == nil`.

#### 7. `TestSave_AnalysisJSON_RoundTrip`

- Create a specific analysis JSON with known values.
- Save a promotion with that analysis.
- Retrieve via `List(ctx, 1)`.
- Unmarshal the returned `AnalysisJSON` into a map and verify specific field values:
  - `primary_value_proposition` matches.
  - `ideal_audience` is an array with expected length.
  - `key_features` contains expected items.
- This tests that the JSON survives the save → SQLite TEXT → load → `json.RawMessage` round trip without corruption.

#### 8. `TestSave_AnalysisJSON_MarshalToJSON`

- Save a promotion with analysis.
- Retrieve via `List(ctx, 1)`.
- Marshal the entire `Promotion` struct to JSON using `json.Marshal()`.
- Unmarshal into a generic `map[string]interface{}`.
- Assert: the `"analysis"` key exists and is a `map[string]interface{}` (nested object, not a string).
- This verifies that `json.RawMessage` produces the correct nested JSON structure for the frontend.

#### 9. `TestSave_AnalysisJSON_NullMarshalToJSON`

- Save a promotion without analysis (nil `AnalysisJSON`).
- Retrieve and marshal to JSON.
- Unmarshal into a generic `map[string]interface{}`.
- Assert: the `"analysis"` key exists and its value is `nil` (JSON `null`).
- This verifies graceful degradation for the frontend.

#### 10. `TestMigration_AnalysisColumn`

- Open a fresh `:memory:` store (which runs `New()` and `applyMigrations()`).
- Run `applyMigrations()` **again** on the same store.
- Assert: no error (the "duplicate column" guard handles it).
- This verifies that migrations are idempotent.

### Test structure

Use standard Go testing with `testing.T`. No external test libraries. Use `t.Run` sub-tests where appropriate for readability.

Each test should use `newTestStore(t)` for a fresh database — tests must not depend on each other.

## What NOT to do

- Do NOT modify `store.go` — tests only.
- Do NOT add external test dependencies.
- Do NOT test the agent client or handler (those have their own test files).
- Do NOT test FTS5 indexing of analysis (it's explicitly excluded from FTS).

## Verification

1. `go test ./internal/store/... -v` — all tests pass.
2. `go build ./...` — still compiles.
3. Test file is `internal/store/store_test.go` with package `store`.
4. All 10 test cases listed above are implemented.
5. Tests use `:memory:` SQLite databases (no file I/O, fast, isolated).
