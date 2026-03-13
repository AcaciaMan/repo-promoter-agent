# Prompt: Phase 3 Smoke Test — Verify Agent Tone Adjustment with Traffic Metrics

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. I just finished **Phase 3** which makes traffic metrics influence the AI agent's promotional tone:

- Prompt 24: Extended `RepoMetrics` with traffic fields + updated prompt template with tone rules.
- Prompt 25: Moved traffic fetch before the agent call so metrics are included in agent input.

The full intent document is at `docs/intent-for-views-clones.md`.

## Your task

Verify that Phase 3 is working correctly. This is a **test and fix** prompt — compile, run, and check three scenarios.

### Step 1: Compile and start

```bash
go build ./...
go run cmd/server/main.go
```

Make sure the server starts with `GITHUB_TOKEN` set.

### Step 2: Scenario A — AcaciaMan repo (traffic metrics present)

Run:

```bash
curl -s -X POST http://localhost:8080/api/generate \
  -H "Content-Type: application/json" \
  -d '{"repo_url": "https://github.com/AcaciaMan/village-square", "target_channel": "general"}' | jq '{headline, summary, views_14d_total, clones_14d_total}'
```

**Verify:**
1. `views_14d_total` and `clones_14d_total` are present in the response (may be 0 if the repo has no traffic, that's okay).
2. Check server logs: the traffic fetch log line should appear **before** the "Agent call succeeded" log line. This confirms metrics were fetched before calling the agent.
3. If the repo has non-zero traffic: the `summary` or `headline` may include softer language about the project attracting interest. This is subtle — don't expect dramatic changes, just verify it doesn't make false claims.
4. If the repo has zero traffic: the tone should be neutral, with no claims about popularity.

### Step 3: Scenario B — Non-AcaciaMan repo (no traffic metrics)

Run:

```bash
curl -s -X POST http://localhost:8080/api/generate \
  -H "Content-Type: application/json" \
  -d '{"repo_url": "https://github.com/golang/go", "target_channel": "twitter"}' | jq '{headline, summary, views_14d_total, clones_14d_total}'
```

**Verify:**
1. `views_14d_total` and `clones_14d_total` are `0`.
2. Server logs show NO traffic fetch attempt.
3. The generated content does not reference traffic, views, or discovery metrics.

### Step 4: Scenario C — No token

1. Remove or unset `GITHUB_TOKEN`.
2. Restart the server.
3. Generate for an AcaciaMan repo.

**Verify:**
1. No traffic fetch attempt (log says "No GITHUB_TOKEN set...").
2. Traffic fields are `0` in the response.
3. Generated content has neutral tone regarding popularity.

### Step 5: Check the agent input (optional but recommended)

To see exactly what the agent receives, temporarily add a log line in the generate handler that prints the marshaled `input` JSON before calling the agent. Check:

- For AcaciaMan repos with traffic: `metrics` includes `views_14d_total`, `views_14d_unique`, `clones_14d_total`, `clones_14d_unique`.
- For others: `metrics` only has `stars`, `forks`, `watchers`, `open_issues` (traffic fields are omitted due to `omitempty`).

Remove the debug log line after verifying.

### Step 6: Compare generate output — before vs after

Run two generates for the same AcaciaMan repo and compare:
- The tone should be consistent — traffic metrics provide a subtle influence, not a dramatic rewrite.
- The `headline`, `key_benefits`, `twitter_posts` should still focus on the project's features, not its traffic numbers.
- No generated text should cite specific view/clone counts.

## Common issues to fix

- **`omitempty` not working:** If traffic fields show as `0` in the agent input JSON (instead of being omitted), check that the struct tags include `omitempty`. Note: `omitempty` omits zero-value ints.
- **Traffic fetch after agent call:** If server logs show the traffic fetch line AFTER "Agent call succeeded", the handler change from prompt 25 wasn't applied correctly. The traffic fetch block must be before the `h.agentClient.Generate()` call.
- **Agent ignoring metrics:** The prompt rules are guidance, not guarantees. The LLM may not always visibly adjust tone. This is acceptable — the important thing is that the data reaches the agent.
- **Compile errors:** `RepoMetrics` field names must match between `internal/agent/client.go` and the handler code that sets them.

## Deliverable

After this prompt, Phase 3 is **complete and verified**:

- `RepoMetrics` includes traffic fields sent to the agent via `omitempty`.
- The prompt template instructs the agent to subtly adjust tone based on traffic levels.
- Traffic is fetched before the agent call so it's available as input context.
- Generated content is truthful — no fabricated popularity claims.
- Everything degrades gracefully when metrics are unavailable.
