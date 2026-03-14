# Prompt: Relevance Tuning — Field Boosting and Phrase Fields

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. The app stores AI-generated promotional content for GitHub repos in **Apache Solr 10** and exposes full-text search via `GET /api/search?q=...`.

The Solr store is working (prompts 54–62 are complete). The `Store.Search()` method in `internal/store/store.go` uses the **edismax** query parser, but **all query fields are equally weighted** — a match in `headline` ranks the same as a match deep inside `twitter_posts`. This makes search results feel random and unfocused.

This is **Phase 1, prompt 1 of 3** for relevance tuning.

## Current Search implementation (`internal/store/store.go`)

The `Search` method builds Solr query parameters like this:

```go
params := url.Values{
    "q":       {q},
    "defType": {"edismax"},
    "qf":      {"repo_name headline summary key_benefits tags twitter_posts linkedin_post call_to_action target_audience readme"},
    "rows":    {fmt.Sprintf("%d", limit)},
    "sort":    {"score desc"},
    "wt":      {"json"},
    "fl":      {"*"},
}
```

**Problems:**
1. All 10 fields in `qf` have equal boost weight (implicit `^1.0`).
2. No phrase-field boosting (`pf`) — documents where the query appears as an exact phrase get no extra score.
3. No tie-breaker (`tie`) — only the single best-matching field contributes to score; other matching fields are ignored.

## Your task

Update the `Search` method in `internal/store/store.go` to add **field boost weights**, **phrase-field boosting**, and a **tie-breaker parameter** to the edismax query.

## Requirements

### 1. Add boost weights to `qf`

Replace the flat `qf` value with field-specific boost weights. Use this weighting logic:

| Field | Boost | Rationale |
|---|---|---|
| `headline` | `^4` | Most important — the catchy one-liner; exact match here is strongest signal |
| `repo_name` | `^3` | Users often search by project name |
| `tags` | `^3` | Tags are curated keywords — a match is highly intentional |
| `summary` | `^2` | Core description, but longer so matches are more diluted |
| `key_benefits` | `^1.5` | Useful content but secondary |
| `target_audience` | `^1.5` | Audience-specific searches should match well |
| `call_to_action` | `^1` | Rarely searched but may contain repo-specific terms |
| `twitter_posts` | `^1` | Channel-specific content, lower priority |
| `linkedin_post` | `^1` | Channel-specific content, lower priority |
| `readme` | `^0.5` | Longest field, most noise — match here is weakest signal |

The resulting `qf` string should look like:

```
repo_name^3 headline^4 summary^2 key_benefits^1.5 tags^3 twitter_posts^1 linkedin_post^1 call_to_action^1 target_audience^1.5 readme^0.5
```

### 2. Add phrase-field boosting (`pf`)

Add a `pf` parameter to boost documents where the user's **full query appears as an exact phrase** in high-value fields. This makes multi-word queries far more precise.

```
pf = headline^6 summary^3 repo_name^4
```

When a user searches for `"CLI tool for testing"`, documents with that exact phrase in the headline get a large score boost.

### 3. Add tie-breaker (`tie`)

Add `tie` parameter with value `0.1`. This ensures that when a query matches multiple fields, the non-top fields still contribute 10% of their score. Without this, only the best-matching field matters and multi-field matches are ignored.

```
tie = 0.1
```

### 4. Implementation

Apply the changes **only** to the `params` construction in the `Search` method. No other functions should change.

The updated params should be:

```go
params := url.Values{
    "q":       {q},
    "defType": {"edismax"},
    "qf":      {"repo_name^3 headline^4 summary^2 key_benefits^1.5 tags^3 twitter_posts^1 linkedin_post^1 call_to_action^1 target_audience^1.5 readme^0.5"},
    "pf":      {"headline^6 summary^3 repo_name^4"},
    "tie":     {"0.1"},
    "rows":    {fmt.Sprintf("%d", limit)},
    "sort":    {"score desc"},
    "wt":      {"json"},
    "fl":      {"*"},
}
```

### 5. Add a code comment

Add a brief comment above the `params` block explaining the boost strategy:

```go
// edismax with field boosting: headline/tags/name weighted highest,
// summary mid-tier, social posts baseline, readme lowest.
// pf boosts exact phrase matches in key fields.
// tie=0.1 lets non-top matching fields contribute 10% of their score.
```

## Verification

After applying the change:

1. Run `go build ./...` — must compile without errors.
2. Run `go test ./internal/store/...` — all existing tests must pass.

## Files to modify

- `internal/store/store.go` — `Search` method only (the `params` variable)

## Files NOT to modify

- `internal/handler/search.go` — no changes needed
- `static/index.html` — no changes needed
- Any test files — no changes needed (existing tests don't assert on Solr query params)
