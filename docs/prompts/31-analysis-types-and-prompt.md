# Prompt: Define Analysis Agent Go Types and Prompt Template

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. In Phase 0 I provisioned an Analysis Agent on Gradient and wired `ANALYSIS_AGENT_ENDPOINT` / `ANALYSIS_AGENT_ACCESS_KEY` as optional env vars in `main.go`.

Now I'm starting **Phase 1** — building the Analysis Agent backend client. This first prompt defines the Go types and prompt template. The next prompts (32, 33) will implement the client function and tests.

The full intent document is at `docs/intent-for-analysis-agent.md` — read it for high-level context.  
The analysis agent's model instructions are at `docs/analysis-agent-model-instructions.md`.

## Current state

The existing Promotion Agent client is in `internal/agent/client.go`. It has:

- **`RepoInput`** struct — input to the promotion agent (repo metadata, metrics, target channel/audience).
- **`RepoMetrics`** struct — stars, forks, watchers, open_issues, 14-day traffic counts.
- **`Client`** struct — endpoint, accessKey, httpClient (60s timeout).
- **`Generate(ctx, RepoInput) (json.RawMessage, error)`** — sends prompt to Gradient, extracts JSON.
- Chat completion request/response types (`chatRequest`, `chatMessage`, `chatCompletion`, `chatChoice`).
- `promptTemplate` const + `outputSchema` const rendered via `text/template`.
- `stripMarkdownFences()` helper.

After Phase 0 (prompt 30), `main.go` creates an optional `analysisClient *agent.Client` and passes it to `GenerateHandler`. The handler struct stores it but doesn't use it yet.

## Your task

Create a new file `internal/agent/analysis.go` that defines the types and prompt template for the Analysis Agent. Do **not** implement the client function yet — just types, constants, and the template.

### 1. Analysis Agent input type

```go
// AnalysisInput is the structured repo data sent to the Analysis Agent.
type AnalysisInput struct {
    RepoURL          string          `json:"repo_url"`
    RepoName         string          `json:"repo_name"`
    ShortDescription string          `json:"short_description"`
    ReadmeText       string          `json:"readme_text"`
    Topics           []string        `json:"topics"`
    Metrics          AnalysisMetrics `json:"metrics"`
    TargetAudience   string          `json:"target_audience,omitempty"`
}
```

Note: this is **different** from `RepoInput` (used by the Promotion Agent):
- Field is `readme_text` not `readme_summary` (the analysis agent sees the full/truncated README).
- Field is `short_description` not matching promotion's field.
- No `target_channel` or `primary_language` — analysis doesn't need these.
- Uses its own `AnalysisMetrics` (see below).

### 2. Analysis metrics type

```go
// AnalysisMetrics holds repo metrics for the Analysis Agent input.
type AnalysisMetrics struct {
    Stars           int `json:"stars"`
    Forks           int `json:"forks"`
    Watchers        int `json:"watchers"`
    Views14dTotal   int `json:"views_14d_total,omitempty"`
    Views14dUnique  int `json:"views_14d_unique,omitempty"`
    Clones14dTotal  int `json:"clones_14d_total,omitempty"`
    Clones14dUnique int `json:"clones_14d_unique,omitempty"`
}
```

Differences from `RepoMetrics`: no `open_issues` field (analysis doesn't need it).

### 3. Analysis Agent output type

```go
// AnalysisOutput is the structured JSON returned by the Analysis Agent.
type AnalysisOutput struct {
    RepoURL                      string   `json:"repo_url"`
    RepoName                     string   `json:"repo_name"`
    PrimaryValueProposition      string   `json:"primary_value_proposition"`
    IdealAudience                []string `json:"ideal_audience"`
    KeyFeatures                  []string `json:"key_features"`
    Differentiators              []string `json:"differentiators"`
    RiskOrLimitations            []string `json:"risk_or_limitations"`
    SocialProofSignals           []string `json:"social_proof_signals"`
    RecommendedPositioningAngle  []string `json:"recommended_positioning_angle"`
}
```

### 4. Prompt template

Define a `const analysisPromptTemplate` and corresponding `analysisOutputSchema` following the same pattern as the existing promotion prompt, but tailored for analysis.

The prompt should:

1. Instruct the agent to analyze the repository data and produce the analysis JSON.
2. Inject the repo data JSON (will be rendered via `text/template`).
3. Include the output schema so the agent knows the exact structure.
4. Include rules mirroring the style constraints from `docs/analysis-agent-model-instructions.md`:
   - Output only JSON, no markdown fences or explanations.
   - Base statements strictly on input — do not invent features.
   - If something is unclear, say so.
   - Concise developer-friendly language, avoid buzzwords.
   - 1–2 short sentences per string item.
   - 2–4 items for each array field.
   - Interpret traffic metrics naturally (same tiered phrasing as promotion template).
   - Handle sparse inputs gracefully: "If README is empty or very short, acknowledge limited information rather than guessing about features."
   - Handle inactive repos: "If traffic metrics are zero or absent, describe as early-stage or limited public visibility — do not assume active usage."

5. Compile the template with `template.Must(template.New("analysis").Parse(...))` and assign to a package-level `var analysisPromptTmpl`.

Here's the template structure:

```go
const analysisPromptTemplate = `Analyze this GitHub repository and produce a structured marketing analysis.

REPO DATA:
{{.RepoDataJSON}}

Return a JSON object with exactly this structure:
{{.OutputSchemaJSON}}

RULES:
- Output ONLY the JSON object. No markdown fences, no preamble, no explanation.
- Echo repo_url and repo_name exactly from the input.
- Generate 2–4 items for each array field (ideal_audience, key_features, differentiators, risk_or_limitations, social_proof_signals, recommended_positioning_angle).
- Base every statement strictly on the provided input. Do not invent features, integrations, or capabilities not described.
- If something is uncertain (e.g., docs quality, test coverage), say it is unclear rather than guessing.
- Use concise, developer-friendly language. Avoid generic marketing buzzwords.
- Keep each string item to 1–2 short sentences.
- Tailor to target_audience if provided; otherwise infer from the repo's language and domain.
- For social_proof_signals: interpret stars and traffic metrics naturally:
  - Low activity (<50 views, few stars): "early-stage project with limited public visibility"
  - Moderate (50–200 views): "gaining some developer attention"
  - High (>200 views): "actively discovered by developers"
  - Zero or absent traffic: do not assume active usage; describe as early-stage or visibility unknown.
- If readme_text is empty or very short, acknowledge limited information. Do not fabricate features from nothing.
- For risk_or_limitations: if the repo appears well-documented and mature, say "none clearly indicated". Do not pad with generic risks.`
```

And the output schema:

```go
const analysisOutputSchema = `{
  "repo_url": "string",
  "repo_name": "string",
  "primary_value_proposition": "One sentence explaining what this repo helps users achieve.",
  "ideal_audience": ["Short description of audience segment"],
  "key_features": ["Feature written as a user-facing benefit"],
  "differentiators": ["What makes this repo special vs. alternatives"],
  "risk_or_limitations": ["Caveats or 'none clearly indicated'"],
  "social_proof_signals": ["Interpretation of stars/traffic"],
  "recommended_positioning_angle": ["Suggested marketing angle"]
}`
```

## What NOT to do

- Do NOT modify `client.go` — the existing Promotion Agent client stays untouched.
- Do NOT implement the `Analyze()` method yet (that's prompt 32).
- Do NOT write tests yet (that's prompt 33).
- Do NOT modify `main.go`, handlers, store, or frontend.
- Do NOT duplicate the `chatRequest`, `chatMessage`, `chatCompletion`, `chatChoice` types — they're already in `client.go` and shared within the `agent` package.

## Verification

1. `go build ./...` compiles without errors.
2. The new file `internal/agent/analysis.go` exists with package `agent`.
3. Types `AnalysisInput`, `AnalysisMetrics`, `AnalysisOutput` are defined.
4. Constants `analysisPromptTemplate` and `analysisOutputSchema` are defined.
5. Package-level var `analysisPromptTmpl` is compiled.
6. No changes to any other files.
