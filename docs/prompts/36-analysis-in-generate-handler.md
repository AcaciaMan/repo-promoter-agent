# Prompt: Wire Analysis Agent into the Generate Handler

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. In Phase 1 (prompts 31–33) I built the `AnalysisClient` with `Analyze()` method and tests. In Phase 2 (prompts 34–35) I added `analysis_json` to the DB schema and the `Promotion` struct. Now I'm starting **Phase 3** — wiring the analysis call into the actual `/api/generate` flow.

The full intent document is at `docs/intent-for-analysis-agent.md` — read it for high-level context.

## Current state

### `internal/handler/generate.go`

The `GenerateHandler` struct already has an `analysisClient *agent.AnalysisClient` field (added in prompt 30), but it is **not used yet**. The current flow is:

1. Parse request body (`repo_url`, `target_channel`, `target_audience`).
2. Normalize `target_channel`.
3. Fetch repo metadata from GitHub (or use hardcoded fallback).
4. Set `input.TargetChannel` and `input.TargetAudience`.
5. Fetch traffic metrics for AcaciaMan repos (best-effort).
6. Copy traffic metrics into `input.Metrics`.
7. Call `h.agentClient.Generate(ctx, input)` — the **promotion agent**.
8. Parse result into `store.Promotion`.
9. Copy traffic metrics to promotion.
10. Save to DB (best-effort).
11. Return JSON response.

### `internal/agent/analysis.go` (from prompts 31–32)

- `AnalysisClient` struct with `Analyze(ctx, AnalysisInput) (*AnalysisOutput, error)`.
- `AnalysisInput` struct — `RepoURL`, `RepoName`, `ShortDescription`, `ReadmeText`, `Topics`, `Metrics` (`AnalysisMetrics`), `TargetAudience`.
- `AnalysisOutput` struct — `RepoURL`, `RepoName`, `PrimaryValueProposition`, `IdealAudience`, `KeyFeatures`, `Differentiators`, `RiskOrLimitations`, `SocialProofSignals`, `RecommendedPositioningAngle`.

### `internal/agent/client.go`

- `RepoInput` struct — input to the promotion agent. Fields: `RepoURL`, `RepoName`, `ShortDescription`, `ReadmeSummary`, `PrimaryLanguage`, `Topics`, `Metrics` (`RepoMetrics`), `TargetChannel`, `TargetAudience`. **No `Analysis` field yet.**

### `internal/store/store.go` (from prompt 34)

- `Promotion` struct now has `AnalysisJSON json.RawMessage` field at the end.
- `Save()`, `Search()`, `List()`, `scanPromotions()` all handle `analysis_json`.

### `internal/github/client.go`

- `FetchRepo(ctx, repoURL) (agent.RepoInput, error)` — returns `RepoInput` (which has `ReadmeSummary`).
- Note: `ReadmeSummary` in `RepoInput` is the full/truncated README text from GitHub.

## Your task

### 1. Add `Analysis` field to `RepoInput` in `client.go`

Add an optional analysis field to the promotion agent's input struct so the promotion agent can use analysis insights:

```go
type RepoInput struct {
    RepoURL          string           `json:"repo_url"`
    RepoName         string           `json:"repo_name"`
    ShortDescription string           `json:"short_description"`
    ReadmeSummary    string           `json:"readme_summary"`
    PrimaryLanguage  string           `json:"primary_language,omitempty"`
    Topics           []string         `json:"topics,omitempty"`
    Metrics          RepoMetrics      `json:"metrics"`
    TargetChannel    string           `json:"target_channel,omitempty"`
    TargetAudience   string           `json:"target_audience,omitempty"`
    Analysis         *AnalysisOutput  `json:"analysis,omitempty"`
}
```

`*AnalysisOutput` pointer so it's `omitempty`-able — when nil, it's omitted from the JSON sent to the promotion agent.

### 2. Call the Analysis Agent before the Promotion Agent

In `generate.go`, add the analysis call **after** traffic metrics fetch and **before** the `agentClient.Generate()` call. Insert this block between the traffic metrics section and the "Call the agent" section:

```go
// Call the analysis agent (if configured).
var analysisJSON json.RawMessage
if h.analysisClient != nil {
    analysisInput := agent.AnalysisInput{
        RepoURL:          input.RepoURL,
        RepoName:         input.RepoName,
        ShortDescription: input.ShortDescription,
        ReadmeText:       input.ReadmeSummary,
        Topics:           input.Topics,
        Metrics: agent.AnalysisMetrics{
            Stars:           input.Metrics.Stars,
            Forks:           input.Metrics.Forks,
            Watchers:        input.Metrics.Watchers,
            Views14dTotal:   input.Metrics.Views14dTotal,
            Views14dUnique:  input.Metrics.Views14dUnique,
            Clones14dTotal:  input.Metrics.Clones14dTotal,
            Clones14dUnique: input.Metrics.Clones14dUnique,
        },
        TargetAudience: req.TargetAudience,
    }

    analysisOutput, analysisErr := h.analysisClient.Analyze(r.Context(), analysisInput)
    if analysisErr != nil {
        log.Printf("WARNING: analysis agent failed, proceeding without analysis: %v", analysisErr)
        // Fail-soft: continue without analysis
    } else {
        // Pass analysis to promotion agent
        input.Analysis = analysisOutput
        // Serialize for DB storage
        if raw, err := json.Marshal(analysisOutput); err == nil {
            analysisJSON = raw
        }
    }
}
```

**Key design decisions:**
- **Fail-soft:** If analysis fails, log a warning and proceed — the promotion agent works fine without it.
- **`ReadmeText` mapping:** Use `input.ReadmeSummary` from the GitHub-fetched `RepoInput` as the analysis agent's `ReadmeText`. It's the same content, just named differently in each agent's contract.
- **`AnalysisMetrics` construction:** Map fields individually from `RepoMetrics` to `AnalysisMetrics` (they have slightly different fields — `AnalysisMetrics` omits `OpenIssues`).
- **`analysisJSON`:** Declared outside the `if` block so it's available when setting the promotion later.

### 3. Set `AnalysisJSON` on the Promotion before saving

After parsing the agent result into `promo` and before calling `h.store.Save()`, add:

```go
promo.AnalysisJSON = analysisJSON
```

Place this in the section where `promo.TargetChannel`, traffic metrics, etc. are set — after `json.Unmarshal(result, &promo)` and before `h.store.Save()`.

### 4. Add `"encoding/json"` to imports if not already present

The `json` package is already imported. But now you're also using `json.RawMessage` and `json.Marshal` directly in the handler. Verify the import is there (it should be from the existing `json.NewEncoder` usage).

### 5. Do NOT change the response format

The response is already `json.NewEncoder(w).Encode(promo)`. Since `Promotion.AnalysisJSON` marshals as a nested JSON object (or `null`), the frontend will now receive:

```json
{
  "id": 1,
  "repo_url": "...",
  "headline": "...",
  ...
  "analysis": {
    "repo_url": "...",
    "primary_value_proposition": "...",
    "ideal_audience": ["..."],
    ...
  }
}
```

Or when analysis is unavailable:

```json
{
  ...
  "analysis": null
}
```

No changes needed to the response encoding.

## What NOT to change

- Do not modify `search.go` — Phase 4 handles that (it should already work via the store/model changes).
- Do not modify the frontend — Phase 5 handles the UI.
- Do not add test files in this prompt — this is a wiring prompt; the behavior is verified by manual testing (next prompt).
- Do not change `AnalysisClient.Analyze()` or the analysis types.
- Do not change the DB schema or store functions.

## Verification

After making changes:

1. `go build ./...` compiles without errors.
2. Read through the handler flow and confirm:
   - When `analysisClient` is nil → analysis call is skipped, `analysisJSON` stays nil, everything works.
   - When `analysisClient` is set and analysis succeeds → `input.Analysis` is set for the promotion agent, `analysisJSON` is stored, response includes analysis.
   - When `analysisClient` is set but analysis fails → warning is logged, promotion agent is called without analysis, `analysisJSON` is nil.
3. Confirm the fallback `defaultRepoInput()` path also works (analysis call still attempts if client is configured).
