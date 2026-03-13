# Prompt: Wire Traffic Metrics into the Generate Handler

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. I'm adding GitHub traffic metrics (views & clones) for AcaciaMan repos.

This is **Phase 1, Step 4**. The previous prompts added:

- Prompt 16: GitHub token support (`NewClient(token)`), `RepoOwner()` helper, `newGitHubRequest()`.
- Prompt 17: `FetchTrafficMetrics(owner, repo)` → `TrafficMetrics`, `HasToken()`.
- Prompt 18: Schema migration (4 new columns), `Promotion` struct extended with traffic fields, `Save`/`Search`/`List` updated.

The full intent document is at `docs/intent-for-views-clones.md`.

## Current generate handler (`internal/handler/generate.go`)

```go
type GenerateHandler struct {
    agentClient  *agent.Client
    githubClient *github.Client
    store        *store.Store
}

func (h *GenerateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // 1. Parse request (repo_url, target_channel, target_audience)
    // 2. Normalize channel
    // 3. FetchRepo from GitHub (or use hardcoded fallback)
    // 4. Call agent
    // 5. Parse agent output into store.Promotion
    // 6. Save to store (best-effort)
    // 7. Return promotion JSON
}
```

Key detail: after step 5 (parsing agent output into `promo`), the handler sets `promo.TargetChannel` and `promo.TargetAudience`, then calls `h.store.Save(ctx, &promo)`.

## GitHub client state (after prompts 16–17)

```go
github.RepoOwner(repoURL string) string       // returns owner or ""
client.HasToken() bool                          // true if token configured
client.FetchTrafficMetrics(ctx, owner, repo) (TrafficMetrics, error)
```

## Your task

Modify `internal/handler/generate.go` to fetch and store traffic metrics for eligible repos.

### Integration logic

After the agent call succeeds and the agent output is parsed into `promo`, but **before** `h.store.Save()`:

1. **Check eligibility:**
   ```go
   owner := github.RepoOwner(promo.RepoURL)
   if owner == "AcaciaMan" && h.githubClient.HasToken() {
   ```

2. **Fetch traffic metrics:**
   ```go
   _, repoName, _ := // extract from URL — or use parseGitHubURL if accessible
   metrics, err := h.githubClient.FetchTrafficMetrics(r.Context(), owner, repoName)
   ```

   Note: `parseGitHubURL` is unexported. Use `github.RepoOwner` to get the owner, and for the repo name, extract it from the URL similarly. Options:
   - Add an exported `ParseRepoURL(url) (owner, repo, error)` to the github package (preferred — it wraps the existing `parseGitHubURL`).
   - Or split the URL in the handler (less clean).

   **Recommended:** Add a small exported wrapper in `internal/github/client.go`:
   ```go
   // ParseRepoURL extracts owner and repo from a GitHub URL.
   func ParseRepoURL(rawURL string) (owner, repo string, err error) {
       return parseGitHubURL(rawURL)
   }
   ```

3. **Handle errors gracefully:**
   - If `FetchTrafficMetrics` fails, log a warning and continue — **do not fail the request**.
   - The promotion still gets saved with zero metrics.

   ```go
   if err != nil {
       log.Printf("WARNING: traffic metrics fetch failed for %s/%s: %v", owner, repoName, err)
   } else {
       promo.Views14dTotal = metrics.Views14dTotal
       promo.Views14dUnique = metrics.Views14dUnique
       promo.Clones14dTotal = metrics.Clones14dTotal
       promo.Clones14dUnique = metrics.Clones14dUnique
   }
   ```

4. **Proceed with `h.store.Save()`** as before — the promotion now includes traffic metrics (or zeros if fetch failed / not eligible).

### What the flow looks like after this change

```
POST /api/generate
  ├── Parse request
  ├── Normalize channel
  ├── Fetch repo data from GitHub (public API)
  ├── Call AI agent → promotional content
  ├── Parse agent output into Promotion
  ├── IF owner == "AcaciaMan" AND token present:
  │     └── Fetch traffic metrics (best-effort)
  │           └── Set metrics on Promotion (or log warning and continue)
  ├── Save to SQLite (best-effort)
  └── Return Promotion JSON (now includes traffic fields)
```

### Important: the response JSON now exposes metrics

Even though this is "Phase 1 — no UI changes," the API response will now include `views_14d_total`, `views_14d_unique`, `clones_14d_total`, `clones_14d_unique` in the JSON. This is fine — the frontend will simply ignore fields it doesn't render yet. The fields will be zeros for non-AcaciaMan repos or when no token is set.

## What NOT to do

- Do NOT send traffic metrics to the AI agent yet (that's Phase 3).
- Do NOT modify the frontend.
- Do NOT modify the agent client or prompt template.
- Do NOT fail the generate request if traffic metrics fetch fails.
- Do NOT change any existing behavior for non-AcaciaMan repos or when no token is set.

## Verification

After implementation:

1. `go build ./...` compiles without errors.
2. **No token set:** Generate works exactly as before — traffic fields are 0 in the response.
3. **Token set, non-AcaciaMan repo:** Generate works as before — traffic fields are 0.
4. **Token set, AcaciaMan repo:** Generate fetches traffic metrics and includes them in the response JSON. Example:
   ```bash
   curl -X POST http://localhost:8080/api/generate \
     -H "Content-Type: application/json" \
     -d '{"repo_url": "https://github.com/AcaciaMan/village-square"}'
   ```
   Response should include non-zero `views_14d_total` etc. (assuming the repo has traffic).
5. **Token set, AcaciaMan repo, traffic API fails** (e.g., token without proper permissions): Generate still succeeds — log shows a warning, traffic fields are 0.
