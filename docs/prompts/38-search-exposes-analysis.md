# Prompt: Verify Analysis Exposed via /api/search

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. In Phase 2 (prompts 34–35) I extended the `Promotion` struct with `AnalysisJSON json.RawMessage` (tagged `json:"analysis"`) and updated the store's `Search()`, `List()`, and `scanPromotions()` to load `analysis_json`. In Phase 3 (prompts 36–37) I wired the Analysis Agent into `/api/generate` so analysis is generated and stored.

Now I need to verify **Phase 4** — that `/api/search` automatically exposes the `analysis` field in each returned promotion.

The full intent document is at `docs/intent-for-analysis-agent.md` — read it for high-level context.

## Current state

### `internal/handler/search.go`

The search handler is straightforward:

```go
type searchResponse struct {
    Results []store.Promotion `json:"results"`
    Count   int               `json:"count"`
}
```

It calls `h.store.List()` or `h.store.Search()`, both of which return `[]store.Promotion`. Since `Promotion` already has `AnalysisJSON json.RawMessage \`json:"analysis"\``, the JSON encoder should automatically include the `analysis` field in every result.

### `internal/store/store.go` (after prompt 34)

- `Promotion` struct has `AnalysisJSON json.RawMessage \`json:"analysis"\`` as the last field.
- `scanPromotions()` scans `analysis_json` via `sql.NullString` and converts to `json.RawMessage`.
- `Search()` and `List()` SQL queries include `analysis_json` / `p.analysis_json` in the SELECT.
- `Save()` persists `analysis_json`.

### What should already work

Because the model and store changes were done in Phase 2, `/api/search` should **already** return `analysis` in each promotion object — no code changes needed in the search handler. Phase 4 is a verification step.

## Your task

### 1. Code review — confirm no changes needed

Read through the following files and confirm the analysis field flows through correctly:

- `internal/store/store.go` — verify `Search()` and `List()` SQL queries include `analysis_json`, and `scanPromotions()` scans it.
- `internal/handler/search.go` — verify it returns `[]store.Promotion` without filtering or transforming fields.

If any of these are missing `analysis_json`, fix them. But based on Phase 2 work, they should be correct.

### 2. Build verification

```bash
go build ./...
```

Fix any compile errors.

### 3. Smoke test — search returns analysis

Start the server with both agents configured:

```bash
AGENT_ENDPOINT="<your-promotion-endpoint>" \
AGENT_ACCESS_KEY="<your-promotion-key>" \
ANALYSIS_AGENT_ENDPOINT="<your-analysis-endpoint>" \
ANALYSIS_AGENT_ACCESS_KEY="<your-analysis-key>" \
go run ./cmd/server
```

#### Test A — Generate a promotion with analysis

If you don't already have a recent promotion with analysis in the DB, generate one:

```bash
curl -s -X POST http://localhost:8080/api/generate \
  -H "Content-Type: application/json" \
  -d '{"repo_url":"https://github.com/AcaciaMan/acacia-log","target_channel":"twitter"}' | jq '.analysis'
```

Confirm the response includes a populated `analysis` object.

#### Test B — List returns analysis

```bash
curl -s "http://localhost:8080/api/search?limit=3" | jq '.results[0].analysis'
```

**Expected:** The first result's `analysis` field is a JSON object (not null, not a string) with fields like `primary_value_proposition`, `ideal_audience`, `key_features`, etc.

#### Test C — Search returns analysis

```bash
curl -s "http://localhost:8080/api/search?q=acacia&limit=3" | jq '.results[0].analysis'
```

**Expected:** Same as Test B — analysis is present as a nested object.

#### Test D — Legacy rows without analysis

If the database has older promotions that were generated before the analysis feature:

```bash
curl -s "http://localhost:8080/api/search?limit=10" | jq '[.results[] | {repo_name, has_analysis: (.analysis != null)}]'
```

**Expected:** Older rows show `has_analysis: false`, newer rows show `has_analysis: true`. Both are returned without errors.

#### Test E — Frontend backward compatibility

Open `http://localhost:8080` in a browser:
1. Click on the search/browse section.
2. Verify existing promotions still render correctly.
3. The `analysis` field won't be visible in the UI yet (Phase 5), but confirm nothing breaks — no JavaScript errors in the console, cards still display.

### 4. Verify JSON shape

Confirm the full response shape matches expectations:

```bash
curl -s "http://localhost:8080/api/search?limit=1" | jq '.results[0] | keys'
```

**Expected keys** (order may vary):
```json
["analysis", "call_to_action", "clones_14d_total", "clones_14d_unique", "created_at", "headline", "id", "key_benefits", "linkedin_post", "repo_name", "repo_url", "summary", "tags", "target_audience", "target_channel", "twitter_posts", "views_14d_total", "views_14d_unique"]
```

The `analysis` key should be present in every result — either as a JSON object or `null`.

## What NOT to change

- Do not modify the search handler unless `analysis` is demonstrably missing from responses.
- Do not modify the frontend — Phase 5 handles UI changes.
- Do not modify the store — Phase 2 already handled persistence.
- Do not add new endpoints.

## Summary

Phase 4 is a **verification-only** phase. The store and model changes from Phase 2 should have already wired analysis through to `/api/search`. If everything checks out, simply confirm that Phase 4 is complete. If something is broken, fix the specific issue and re-test.
