# Prompt: Phase 3 Smoke Test — Analysis Agent in Generate Flow

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. In the previous prompt (36), I wired the Analysis Agent into the `/api/generate` handler. The handler now:

1. Fetches repo metadata from GitHub.
2. Fetches traffic metrics (for AcaciaMan repos).
3. **Calls the Analysis Agent** (if configured) — fail-soft on error.
4. Passes analysis output to the Promotion Agent.
5. Stores `analysis_json` in the DB.
6. Returns the promotion with nested `analysis` in the JSON response.

Now I need to verify everything works end-to-end.

The full intent document is at `docs/intent-for-analysis-agent.md` — read it for high-level context.

## Your task

Perform the following manual smoke tests and fix any issues found.

### Test 1 — Build succeeds

```bash
go build ./...
```

Fix any compile errors before proceeding.

### Test 2 — Analysis disabled (env vars not set)

Start the server **without** `ANALYSIS_AGENT_ENDPOINT` / `ANALYSIS_AGENT_ACCESS_KEY`:

```bash
AGENT_ENDPOINT="<your-promotion-endpoint>" \
AGENT_ACCESS_KEY="<your-promotion-key>" \
go run ./cmd/server
```

Then call the generate endpoint:

```bash
curl -s -X POST http://localhost:8080/api/generate \
  -H "Content-Type: application/json" \
  -d '{"repo_url":"https://github.com/AcaciaMan/acacia-log","target_channel":"twitter"}' | jq .
```

**Expected:**
- Server starts without errors or warnings about missing analysis env vars.
- Response is a valid promotion JSON with `"analysis": null`.
- No analysis-related log messages — the call is simply skipped.

### Test 3 — Analysis enabled, happy path

Start the server **with** both agent endpoints configured:

```bash
AGENT_ENDPOINT="<your-promotion-endpoint>" \
AGENT_ACCESS_KEY="<your-promotion-key>" \
ANALYSIS_AGENT_ENDPOINT="<your-analysis-endpoint>" \
ANALYSIS_AGENT_ACCESS_KEY="<your-analysis-key>" \
go run ./cmd/server
```

Call the generate endpoint:

```bash
curl -s -X POST http://localhost:8080/api/generate \
  -H "Content-Type: application/json" \
  -d '{"repo_url":"https://github.com/AcaciaMan/acacia-log","target_channel":"twitter"}' | jq .
```

**Expected:**
- Server log shows `[analysis]` lines: calling, success with duration and output size.
- Server log shows the promotion agent call succeeding after the analysis.
- Response JSON has a populated `analysis` object with fields like `primary_value_proposition`, `ideal_audience`, `key_features`, etc.
- Response JSON has the usual promotion fields (`headline`, `summary`, etc.).
- Total request time is roughly the sum of both agent call durations (expect 15–30s).

### Test 4 — Analysis stored in database

After Test 3, call the search endpoint to confirm the analysis was persisted:

```bash
curl -s "http://localhost:8080/api/search?limit=1" | jq '.[0].analysis'
```

Or:

```bash
curl -s "http://localhost:8080/api/search?limit=1" | jq '.results[0].analysis'
```

**Expected:**
- The `analysis` field is present and matches what was returned in the generate response.
- It's a nested JSON object, not a string.

### Test 5 — Default repo fallback (no repo_url)

```bash
curl -s -X POST http://localhost:8080/api/generate \
  -H "Content-Type: application/json" \
  -d '{"target_channel":"linkedin"}' | jq .
```

**Expected:**
- Uses the hardcoded Village Square fallback repo.
- Analysis is still called (if client is configured) with the fallback data.
- Response includes both promotion and analysis.

### Test 6 — Frontend check

Open `http://localhost:8080` in a browser:
1. Enter a repo URL and click Generate.
2. Verify the response renders without errors.
3. The `analysis` field won't be visible in the UI yet (Phase 5), but confirm it doesn't break anything.

## Common issues to watch for

1. **Import cycles** — `handler` imports `agent`, which is fine. Don't import `handler` from `agent`.
2. **Nil pointer** — If `analysisClient` is nil, the `if h.analysisClient != nil` guard must prevent the call. Verify this.
3. **JSON marshaling** — `json.RawMessage(nil)` marshals to `null`, which is correct. But if it marshals to `""` (empty string), there's a bug in how `analysisJSON` is set.
4. **Scan column count** — The `scanPromotions()` helper must scan exactly the right number of columns. If the query and scan disagree, you'll get runtime errors on search/list.
5. **Context cancellation** — If the analysis call takes too long (>30s timeout), it should fail and the promotion agent should still be called.

## Fix any issues found

If any test fails, diagnose and fix the issue. Common fixes:
- Missing import statement.
- Wrong column count in scan.
- nil check missing or in wrong place.
- `json.Marshal` returning error silently.

After fixes, re-run the failed tests to confirm they pass.
