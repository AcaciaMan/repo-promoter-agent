# Prompt: Move Traffic Fetch Before Agent Call in Generate Handler

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. I'm working on **Phase 3** — making traffic metrics influence AI-generated promotional tone.

The previous prompt (24) extended `RepoMetrics` with traffic fields and updated the prompt template. But the generate handler still fetches traffic metrics **after** the agent call (only for storage). Now I need to move the fetch **before** the agent call so the metrics are included in the `RepoInput` sent to the agent.

The full intent document is at `docs/intent-for-views-clones.md`.

## Current generate handler flow (`internal/handler/generate.go`)

```go
func (h *GenerateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // 1. Parse request
    // 2. Normalize channel
    // 3. Fetch repo data from GitHub → input (agent.RepoInput)
    // 4. Set TargetChannel, TargetAudience on input
    // 5. Call agent with input → result
    // 6. Parse result into promo (store.Promotion)
    // 7. Fetch traffic metrics (AFTER agent) → set on promo    ← PROBLEM
    // 8. Save promo to store
    // 9. Return promo JSON
}
```

The traffic fetch currently happens at step 7 — between parsing the agent output and saving to the store:

```go
    // Fetch traffic metrics for AcaciaMan repos (best-effort).
    owner := github.RepoOwner(promo.RepoURL)
    if owner == "AcaciaMan" && h.githubClient.HasToken() {
        _, repoName, parseErr := github.ParseRepoURL(promo.RepoURL)
        if parseErr == nil {
            metrics, mErr := h.githubClient.FetchTrafficMetrics(r.Context(), owner, repoName)
            if mErr != nil {
                log.Printf("WARNING: traffic metrics fetch failed for %s/%s: %v", owner, repoName, mErr)
            } else {
                promo.Views14dTotal = metrics.Views14dTotal
                promo.Views14dUnique = metrics.Views14dUnique
                promo.Clones14dTotal = metrics.Clones14dTotal
                promo.Clones14dUnique = metrics.Clones14dUnique
            }
        }
    }
```

## Updated `RepoMetrics` (from prompt 24)

```go
type RepoMetrics struct {
    Stars           int `json:"stars"`
    Forks           int `json:"forks"`
    Watchers        int `json:"watchers"`
    OpenIssues      int `json:"open_issues"`
    Views14dTotal   int `json:"views_14d_total,omitempty"`
    Views14dUnique  int `json:"views_14d_unique,omitempty"`
    Clones14dTotal  int `json:"clones_14d_total,omitempty"`
    Clones14dUnique int `json:"clones_14d_unique,omitempty"`
}
```

## Your task

Restructure the generate handler to fetch traffic metrics **before** the agent call, so they flow into the agent via `input.Metrics`.

### New flow

```
1. Parse request
2. Normalize channel
3. Fetch repo data from GitHub → input (agent.RepoInput)
4. Set TargetChannel, TargetAudience on input
5. IF AcaciaMan + token: fetch traffic metrics (best-effort)  ← MOVED HERE
   → Set on input.Metrics (so agent sees them)
   → Also store in local vars for setting on promo later
6. Call agent with input → result
7. Parse result into promo (store.Promotion)
8. Copy traffic metrics onto promo (for storage/UI)           ← STILL NEEDED
9. Save promo to store
10. Return promo JSON
```

### Implementation details

1. **Move traffic fetch to before the agent call.** After building `input` (step 3) and setting channel/audience (step 4), add:

```go
    // Fetch traffic metrics for AcaciaMan repos (best-effort) — needed
    // both for agent input (tone adjustment) and for storage/UI.
    var trafficMetrics github.TrafficMetrics
    if repoURL != "" {
        owner := github.RepoOwner(repoURL)
        if owner == "AcaciaMan" && h.githubClient.HasToken() {
            _, repoName, parseErr := github.ParseRepoURL(repoURL)
            if parseErr == nil {
                tm, mErr := h.githubClient.FetchTrafficMetrics(r.Context(), owner, repoName)
                if mErr != nil {
                    log.Printf("WARNING: traffic metrics fetch failed for %s/%s: %v", owner, repoName, mErr)
                } else {
                    trafficMetrics = tm
                    // Include in agent input so they influence generated tone.
                    input.Metrics.Views14dTotal = tm.Views14dTotal
                    input.Metrics.Views14dUnique = tm.Views14dUnique
                    input.Metrics.Clones14dTotal = tm.Clones14dTotal
                    input.Metrics.Clones14dUnique = tm.Clones14dUnique
                }
            }
        }
    }
```

2. **After parsing agent output into `promo`**, set traffic metrics on the promotion for storage:

```go
    // Copy traffic metrics to promotion for DB storage and UI display.
    promo.Views14dTotal = trafficMetrics.Views14dTotal
    promo.Views14dUnique = trafficMetrics.Views14dUnique
    promo.Clones14dTotal = trafficMetrics.Clones14dTotal
    promo.Clones14dUnique = trafficMetrics.Clones14dUnique
```

3. **Remove the old traffic fetch block** that was between promo parsing and store save.

### Why this structure

- Traffic metrics are fetched **once** and used **twice**: once as agent input (step 5), once for storage (step 8).
- The `trafficMetrics` variable is declared outside the if-block so it's accessible later even when traffic isn't fetched (it stays zero-valued).
- Using `repoURL` (from the request) for the owner check instead of `promo.RepoURL` (from agent output) is more reliable — it's available before the agent call.

## What NOT to do

- Do NOT modify the agent client, store, GitHub client, or frontend.
- Do NOT change the traffic fetch logic itself (error handling, owner check, etc.) — only move it.
- Do NOT remove the metric-setting on `promo` — it's still needed for storage and UI display.
- Do NOT change any other handler behavior.

## Verification

After implementation:

1. `go build ./...` compiles without errors.
2. **AcaciaMan repo with token:** 
   - Server log shows traffic fetch happening BEFORE "Agent call succeeded" (check log timestamps/order).
   - The generated promotional content should subtly reflect traffic (e.g., mentions of "gaining interest" or "actively explored" if views are non-zero).
   - The response JSON still includes `views_14d_total` etc. for UI display.
3. **Non-AcaciaMan repo:** No traffic fetch, agent generates content as before, traffic fields are 0 in response.
4. **No token:** No traffic fetch, same as before.
5. **Traffic fetch fails:** Warning logged, agent still called (without traffic metrics), promotion saved with 0 metrics.
