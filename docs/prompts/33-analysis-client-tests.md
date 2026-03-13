# Prompt: Unit Tests for the Analysis Agent Client

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. In prompts 31–32, I created:

- `internal/agent/analysis.go` — `AnalysisInput`, `AnalysisOutput`, `AnalysisClient`, `Analyze()` method with structured logging, 30-second timeout, and analysis prompt template.

Now I need comprehensive unit tests using a fake HTTP server, covering both the happy path and edge cases outlined in the Phase 1 spec.

The full intent document is at `docs/intent-for-analysis-agent.md` — read it for high-level context.

## Current state

### `internal/agent/analysis.go` (from prompts 31–32)

Key function signature:
```go
func (c *AnalysisClient) Analyze(ctx context.Context, input AnalysisInput) (*AnalysisOutput, error)
```

Behavior:
- Marshals `AnalysisInput` to JSON, renders prompt, POSTs to `{endpoint}/api/v1/chat/completions`.
- Parses `chatCompletion` response envelope, strips markdown fences, unmarshals into `AnalysisOutput`.
- Logs `[analysis]` prefixed messages for timing/size/errors.
- 30-second timeout on the HTTP client.

### Existing tests in project

There are currently no test files in the project. This will be the first `_test.go` file.

## Your task

Create `internal/agent/analysis_test.go` with tests using `net/http/httptest` to mock the Gradient agent endpoint.

### Test helper

Create a helper to build a fake Gradient response:

```go
func fakeAnalysisResponse(content string) string {
    // Returns a valid chatCompletion JSON envelope wrapping `content` 
    // in choices[0].message.content
    resp := map[string]interface{}{
        "choices": []map[string]interface{}{
            {
                "message": map[string]interface{}{
                    "role":    "assistant",
                    "content": content,
                },
            },
        },
    }
    b, _ := json.Marshal(resp)
    return string(b)
}
```

Also create a standard valid `AnalysisInput` helper to avoid repetition:

```go
func sampleAnalysisInput() AnalysisInput {
    return AnalysisInput{
        RepoURL:          "https://github.com/testowner/testrepo",
        RepoName:         "testrepo",
        ShortDescription: "A test repository for unit testing",
        ReadmeText:       "# Test Repo\n\nThis is a test repo with some features.",
        Topics:           []string{"go", "testing"},
        Metrics: AnalysisMetrics{
            Stars:    42,
            Forks:    5,
            Watchers: 3,
        },
        TargetAudience: "Go developers",
    }
}
```

### Required test cases

#### 1. `TestAnalyze_HappyPath`

- Start `httptest.NewServer` that returns HTTP 200 with a valid analysis JSON response.
- Create `AnalysisClient` pointing at the test server.
- Call `Analyze()` with `sampleAnalysisInput()`.
- Assert: no error, `AnalysisOutput` fields populated correctly:
  - `RepoURL` matches input.
  - `RepoName` matches input.
  - `PrimaryValueProposition` is non-empty.
  - `IdealAudience` has at least 1 item.
  - `KeyFeatures` has at least 1 item.
  - `Differentiators` has at least 1 item.

Use a realistic analysis JSON as the mock response:
```json
{
  "repo_url": "https://github.com/testowner/testrepo",
  "repo_name": "testrepo",
  "primary_value_proposition": "Helps Go developers test their code more efficiently.",
  "ideal_audience": ["Go developers writing unit tests", "Teams adopting TDD practices"],
  "key_features": ["Fast test execution", "Simple assertion API"],
  "differentiators": ["Minimal dependencies compared to testify"],
  "risk_or_limitations": ["Early-stage project with limited documentation"],
  "social_proof_signals": ["Modest traction with 42 stars"],
  "recommended_positioning_angle": ["Lightweight alternative to heavy test frameworks"]
}
```

#### 2. `TestAnalyze_MarkdownFences`

- Server returns the analysis JSON wrapped in `` ```json ... ``` `` markdown fences.
- Assert: no error, JSON is still parsed correctly after fence stripping.

#### 3. `TestAnalyze_EmptyReadme`

- Call `Analyze()` with `ReadmeText: ""` (empty string).
- Server returns a valid analysis response (with a value proposition acknowledging limited info).
- Assert: no error, `AnalysisOutput` is populated. This verifies the client handles sparse inputs without crashing.

#### 4. `TestAnalyze_EmptyTopics`

- Call `Analyze()` with `Topics: []string{}` (empty array).
- Server returns a valid analysis response.
- Assert: no error, `AnalysisOutput` is populated.

#### 5. `TestAnalyze_InvalidJSON`

- Server returns HTTP 200 with `choices[0].message.content` set to `"This is not JSON at all"`.
- Assert: error is returned, error message contains "not valid JSON".

#### 6. `TestAnalyze_PartialJSON`

- Server returns HTTP 200 with content that has extra text around JSON: `"Here is the analysis: {valid json} Hope that helps!"`.
- Assert: no error (the fallback `{...}` extraction should rescue this), `AnalysisOutput` is populated. This mirrors the existing behavior in `Generate()`.

#### 7. `TestAnalyze_AgentHTTPError`

- Server returns HTTP 500 with body `"Internal Server Error"`.
- Assert: error is returned, error message contains "status 500".

#### 8. `TestAnalyze_EmptyChoices`

- Server returns HTTP 200 with a valid envelope but empty `choices` array: `{"choices": []}`.
- Assert: error is returned, error message contains "no choices".

#### 9. `TestAnalyze_ContextCancelled`

- Create a context and cancel it immediately before calling `Analyze()`.
- Assert: error is returned (context cancelled).
- This tests the timeout/cancellation path.

#### 10. `TestAnalyze_ExtraFieldsIgnored`

- Server returns analysis JSON with extra fields not in the schema (e.g., `"extra_field": "value"`).
- Assert: no error, `AnalysisOutput` is populated, extra fields are silently ignored.

#### 11. `TestAnalyze_MissingOptionalFields`

- Server returns analysis JSON with only `repo_url`, `repo_name`, and `primary_value_proposition` — all array fields omitted.
- Assert: no error, `AnalysisOutput` has the three fields populated, array fields are nil/empty (zero values).
- This validates that the client doesn't fail when the agent returns fewer fields.

#### 12. `TestAnalyze_RequestFormat`

- Use the test server to capture the incoming request.
- Assert:
  - Method is POST.
  - Path is `/api/v1/chat/completions`.
  - `Content-Type` header is `application/json`.
  - `Authorization` header is `Bearer test-key`.
  - Request body is valid JSON with `messages` array containing one user message.
  - The user message content contains the repo data (e.g., `"testrepo"` appears in the prompt).

### Test structure

Use standard Go testing with `testing.T`. No external test libraries — keep it simple with `t.Errorf` / `t.Fatalf` / `t.Run` for subtests where appropriate.

Each test should create its own `httptest.NewServer` (or use `t.Run` sub-tests with a shared server where it makes sense) and close it with `defer server.Close()`.

## What NOT to do

- Do NOT modify `analysis.go` or `client.go` — tests only.
- Do NOT add external test dependencies / frameworks (no testify, no gomock).
- Do NOT modify any other files.
- Do NOT test the `Generate()` method (that's the promotion agent — already working).

## Verification

1. `go test ./internal/agent/... -v` — all tests pass.
2. `go build ./...` — still compiles.
3. Test file is `internal/agent/analysis_test.go` with package `agent`.
4. All 12 test cases listed above are implemented.
5. No test relies on external network calls (all use `httptest.NewServer`).
