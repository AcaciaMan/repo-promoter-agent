# Prompt: Implement the Analysis Agent Client Function

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. In the previous prompt (31), I defined the Go types and prompt template for the Analysis Agent in `internal/agent/analysis.go`.

Now I need to implement the actual client — a dedicated `AnalysisClient` struct with an `Analyze()` method that calls the Gradient agent and returns the parsed analysis output.

The full intent document is at `docs/intent-for-analysis-agent.md` — read it for high-level context.

## Current state

### `internal/agent/analysis.go` (created in prompt 31)

Contains:
- `AnalysisInput` struct — repo data input for the analysis agent
- `AnalysisMetrics` struct — stars, forks, watchers, 14-day traffic
- `AnalysisOutput` struct — the structured analysis result
- `analysisPromptTemplate` const + `analysisOutputSchema` const
- `analysisPromptTmpl` compiled template

### `internal/agent/client.go` (existing, unchanged)

Contains:
- `Client` struct (promotion agent) — endpoint, accessKey, httpClient (60s timeout)
- `chatRequest`, `chatMessage`, `chatCompletion`, `chatChoice` types (reusable)
- `stripMarkdownFences()` helper (reusable)

### `cmd/server/main.go` (from prompt 30)

Currently creates:
```go
var analysisClient *agent.Client  // reuses promotion Client as placeholder
if analysisEndpoint != "" && analysisKey != "" {
    analysisClient = agent.NewClient(analysisEndpoint, analysisKey)
}
```

This will need to change to use the new `AnalysisClient` type after this prompt.

## Your task

### 1. Add `AnalysisClient` struct to `internal/agent/analysis.go`

```go
// AnalysisClient calls the Analysis Agent's chat completion endpoint.
type AnalysisClient struct {
    endpoint   string
    accessKey  string
    httpClient *http.Client
}

// NewAnalysisClient creates an AnalysisClient with a 30-second timeout.
func NewAnalysisClient(endpoint, accessKey string) *AnalysisClient {
    return &AnalysisClient{
        endpoint:  endpoint,
        accessKey: accessKey,
        httpClient: &http.Client{
            Timeout: 30 * time.Second,
        },
    }
}
```

**Key difference from `Client`:** 30-second timeout instead of 60. Analysis is supplementary; it should fail fast.

### 2. Implement `Analyze()` method

```go
func (c *AnalysisClient) Analyze(ctx context.Context, input AnalysisInput) (*AnalysisOutput, error)
```

Follow the same pattern as `Client.Generate()` in `client.go`, but:

1. **Marshal** `AnalysisInput` to JSON.
2. **Render** the analysis prompt template (inject repo data + output schema).
3. **Build** a `chatRequest` with the rendered prompt as a user message.
4. **POST** to `{endpoint}/api/v1/chat/completions` with Bearer auth.
5. **Parse** the `chatCompletion` response envelope.
6. **Strip** markdown fences using the existing `stripMarkdownFences()` helper.
7. **Validate** JSON (same fallback extraction as `Generate()`).
8. **Unmarshal** the JSON into `AnalysisOutput` struct (unlike `Generate()` which returns raw JSON, we parse into the typed struct).
9. **Return** `(*AnalysisOutput, error)`.

### 3. Add structured logging

Add logging around the agent call for observability. Use `log.Printf` (standard library), consistent with the rest of the codebase. Log:

**Before the call:**
```go
log.Printf("[analysis] calling agent for %s (input size: %d bytes)", input.RepoName, len(inputJSON))
```

**On success:**
```go
log.Printf("[analysis] success for %s (duration: %dms, output size: %d bytes)", input.RepoName, durationMs, len(content))
```

**On failure (at each error return point):**
```go
log.Printf("[analysis] failed for %s (duration: %dms, error: %v)", input.RepoName, durationMs, err)
```

Use `time.Since(start).Milliseconds()` for duration. Capture `start := time.Now()` at the top of the function.

### 4. Update `main.go` to use `AnalysisClient`

Change the analysis client creation in `cmd/server/main.go`:

**Before (from prompt 30):**
```go
var analysisClient *agent.Client
if analysisEndpoint != "" && analysisKey != "" {
    analysisClient = agent.NewClient(analysisEndpoint, analysisKey)
}
```

**After:**
```go
var analysisClient *agent.AnalysisClient
if analysisEndpoint != "" && analysisKey != "" {
    analysisClient = agent.NewAnalysisClient(analysisEndpoint, analysisKey)
}
```

### 5. Update `GenerateHandler` to accept `*agent.AnalysisClient`

In `internal/handler/generate.go`:

- Change the `analysisClient` field type from `*agent.Client` to `*agent.AnalysisClient`.
- Update `NewGenerateHandler` parameter type accordingly.
- Still do **not** use the analysis client in the handler logic (that's Phase 3).

## What NOT to do

- Do NOT modify `client.go` — the existing Promotion Agent client stays untouched.
- Do NOT add any analysis logic to the generate handler flow (that's Phase 3).
- Do NOT write tests (that's prompt 33).
- Do NOT modify the store, search handler, or frontend.
- Do NOT change anything about how `chatRequest`, `chatMessage`, `chatCompletion`, `chatChoice`, or `stripMarkdownFences` work — reuse them from `client.go`.

## Verification

1. `go build ./...` compiles without errors.
2. `internal/agent/analysis.go` has `AnalysisClient` struct with `NewAnalysisClient()` and `Analyze()`.
3. `Analyze()` follows the same HTTP call pattern as `Generate()` but:
   - Uses the analysis prompt template.
   - Returns `*AnalysisOutput` (parsed) instead of `json.RawMessage`.
   - Logs timing/size/errors with `[analysis]` prefix.
   - Uses 30-second HTTP client timeout.
4. `main.go` creates `*agent.AnalysisClient` (not `*agent.Client`) for the analysis agent.
5. `GenerateHandler` stores `*agent.AnalysisClient` but doesn't call it yet.
6. App starts and works exactly as before with both env var scenarios (set / not set).
