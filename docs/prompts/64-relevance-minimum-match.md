# Prompt: Relevance Tuning — Minimum Match and Phrase Slop

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. This is **Phase 1, prompt 2 of 3** for relevance tuning.

**Prerequisite**: Prompt 63 is complete — the `Search` method in `internal/store/store.go` now uses edismax with field boost weights (`qf` with `^` boosts), phrase-field boosting (`pf`), and a tie-breaker (`tie=0.1`).

### Current problem

The `Search` method has no `mm` (minimum match) or `ps` (phrase slop) parameters:

- **No `mm`**: With edismax's default, a search like `"Go CLI testing framework"` may return documents matching only *one* of those four words. Short queries return too many irrelevant results.
- **No `ps`**: The `pf` phrase-field boosting added in prompt 63 requires an **exact contiguous phrase** match. A user searching `"CLI testing tool"` won't get a phrase boost for a headline containing `"CLI unit testing tool"` (one word apart) because the default phrase slop is 0.

## Current Search params (after prompt 63)

```go
// edismax with field boosting: headline/tags/name weighted highest,
// summary mid-tier, social posts baseline, readme lowest.
// pf boosts exact phrase matches in key fields.
// tie=0.1 lets non-top matching fields contribute 10% of their score.
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

## Your task

Add **minimum match** (`mm`) and **phrase slop** (`ps`) parameters to the `Search` method's Solr query params in `internal/store/store.go`.

## Requirements

### 1. Add minimum match (`mm`)

Add the `mm` parameter with value `"2<-1 5<80%"`. This means:

| Terms in query | Required matches | Example |
|---|---|---|
| 1 term | 1 (all) | `"Go"` → must match `Go` |
| 2 terms | 2 (all) | `"Go CLI"` → both must match |
| 3 terms | 2 (3 minus 1) | `"Go CLI testing"` → at least 2 must match |
| 4 terms | 3 (4 minus 1) | `"Go CLI testing framework"` → at least 3 |
| 5+ terms | 80% | `"lightweight Go CLI testing framework"` → 4 of 5 must match |

This ensures short queries are strict (no single-word garbage results) while longer queries allow some flexibility.

### 2. Add phrase slop (`ps`)

Add the `ps` parameter with value `"2"`. This allows `pf` phrase matching to tolerate up to **2 extra/moved words** between query terms.

Example: A search for `"CLI testing tool"` will still get a phrase boost if the headline says `"CLI unit testing tool"` (1 word between "CLI" and "testing").

Without `ps`, `pf` only boosts **exact** contiguous phrases, which is too strict for natural language queries.

### 3. Implementation

Add `mm` and `ps` to the existing `params` block. Place them after `pf` and before `tie` for logical grouping. The final params block should be:

```go
// edismax with field boosting: headline/tags/name weighted highest,
// summary mid-tier, social posts baseline, readme lowest.
// pf boosts exact phrase matches in key fields; ps allows 2-word slop.
// mm requires most query terms to match; tie lets other fields contribute.
params := url.Values{
    "q":       {q},
    "defType": {"edismax"},
    "qf":      {"repo_name^3 headline^4 summary^2 key_benefits^1.5 tags^3 twitter_posts^1 linkedin_post^1 call_to_action^1 target_audience^1.5 readme^0.5"},
    "pf":      {"headline^6 summary^3 repo_name^4"},
    "ps":      {"2"},
    "mm":      {"2<-1 5<80%"},
    "tie":     {"0.1"},
    "rows":    {fmt.Sprintf("%d", limit)},
    "sort":    {"score desc"},
    "wt":      {"json"},
    "fl":      {"*"},
}
```

### 4. Update the comment

Update the existing comment above `params` to mention `mm` and `ps` (see above).

## Verification

After applying the change:

1. Run `go build ./...` — must compile without errors.
2. Run `go test ./internal/store/...` — all existing tests must pass.

## Files to modify

- `internal/store/store.go` — `Search` method only (the `params` block and its comment)

## Files NOT to modify

- `internal/handler/search.go` — no changes needed
- `static/index.html` — no changes needed
- Any test files — existing tests don't assert on Solr query params
