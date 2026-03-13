# Prompt: Phase 1 Smoke Test — Verify Traffic Metrics End-to-End

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. I just finished **Phase 1** of adding GitHub traffic metrics (views & clones) for AcaciaMan repos.

The previous prompts implemented:

- Prompt 16: GitHub token support, `RepoOwner()`, `newGitHubRequest()`.
- Prompt 17: `FetchTrafficMetrics()`, `TrafficMetrics` type, `HasToken()`.
- Prompt 18: Schema migration (4 new columns on `promotions`), extended `Promotion` struct, updated `Save`/`Search`/`List`.
- Prompt 19: Generate handler wired to fetch and store traffic metrics for AcaciaMan repos.

The full intent document is at `docs/intent-for-views-clones.md`.

## Your task

Verify that Phase 1 is working correctly. This is a **test and fix** prompt — your goal is to make sure everything compiles, runs, and behaves as expected.

### Step 1: Compile check

Run:

```bash
go build ./...
```

Fix any compilation errors. Common issues to watch for:

- Import paths (the github package may need new imports for the traffic types).
- `NewClient` signature change (now takes `token string`) — make sure `cmd/server/main.go` passes the token.
- Column count mismatches in SQL queries — all `SELECT` statements in the store must list the same columns that `scanPromotions` scans.

### Step 2: Fresh database test

1. Delete the existing `promotions.db` file (if any).
2. Start the server: `go run cmd/server/main.go` (with `GITHUB_TOKEN` set in `.env` or environment).
3. Verify the server starts without errors.
4. Check logs for the token status message.

### Step 3: Existing database migration test

1. If you have a backed-up `promotions.db` from before the schema change, restore it.
2. Start the server and verify the migration adds columns without errors.
3. If you don't have an old DB, create one without the new columns by temporarily reverting the migration, inserting a row, then re-applying. Or simply trust the `ALTER TABLE` fallback and note it as tested.

### Step 4: Generate with AcaciaMan repo

Run:

```bash
curl -s -X POST http://localhost:8080/api/generate \
  -H "Content-Type: application/json" \
  -d '{"repo_url": "https://github.com/AcaciaMan/village-square"}' | jq .
```

Verify:

- Response includes `views_14d_total`, `views_14d_unique`, `clones_14d_total`, `clones_14d_unique`.
- Values are integers (may be 0 if the repo has no traffic, but fields must be present).
- Server logs show the traffic fetch happening (no error warnings).

### Step 5: Generate with non-AcaciaMan repo

Run:

```bash
curl -s -X POST http://localhost:8080/api/generate \
  -H "Content-Type: application/json" \
  -d '{"repo_url": "https://github.com/golang/go"}' | jq .
```

Verify:

- Response includes traffic fields, all set to `0`.
- Server logs do NOT show any traffic API calls.
- All other fields (headline, summary, tweets, etc.) are populated normally.

### Step 6: Search returns metrics

Run:

```bash
curl -s "http://localhost:8080/api/search?q=village" | jq '.results[0] | {repo_name, views_14d_total, clones_14d_total}'
```

Verify that the stored promotion includes the traffic fields.

### Step 7: No token scenario

1. Remove or unset `GITHUB_TOKEN`.
2. Restart the server.
3. Verify log says "No GITHUB_TOKEN set...".
4. Run the AcaciaMan generate curl again — traffic fields should be 0, no errors.

## What to fix

If any step fails, diagnose and fix the issue. Common problems:

- **SQL column count mismatch:** The number of columns in SELECT must match the number of `Scan` targets in `scanPromotions`.
- **Import cycles:** If the handler imports `github.TrafficMetrics`, make sure the import path is correct.
- **Token not passed:** `cmd/server/main.go` must read `GITHUB_TOKEN` and pass it to `github.NewClient(token)`.
- **Migration errors on fresh DB:** `ALTER TABLE ADD COLUMN` fails on a fresh DB because the table was just created with the columns. The "duplicate column" error handler should catch this — verify it does.

## Deliverable

After this prompt, Phase 1 is **complete and verified**:

- Traffic metrics are fetched from the GitHub API for AcaciaMan repos when a token is available.
- Metrics are stored in SQLite alongside the promotion.
- Metrics appear in the API response JSON (but the frontend doesn't render them yet — that's Phase 2).
- Everything degrades gracefully when no token is set or repo is not AcaciaMan.
