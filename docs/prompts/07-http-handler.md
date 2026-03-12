# Prompt: Implement the /api/generate Handler

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. This is **Phase 1** — a local Go service with hardcoded repo data.

The project structure (`main.go`) was created in prompt 05. The agent client (`internal/agent/client.go`) was implemented in prompt 06. This prompt fills in the HTTP handler.

## Existing project state

> **IMPORTANT**: Before running this prompt, paste your actual file contents for `main.go` and `client.go` here so the session has full context. For now, here's the expected structure:

```
cmd/server/main.go           # entry point (done — prompt 05)
internal/agent/client.go     # agent client (done — prompt 06)
internal/handler/generate.go # THIS FILE — implement now
static/index.html            # not yet created (prompt 08)
```

### Agent client API (from prompt 06)

The agent client exposes:

```go
// In internal/agent
type RepoInput struct { /* fields matching input contract */ }
func (c *Client) Generate(ctx context.Context, input RepoInput) (json.RawMessage, error)
```

## Finalized contracts (from previous sessions)

> **IMPORTANT**: Paste your finalized contracts here before running.

### `/api/generate` request contract (from prompt 04)
> Paste the decided request shape here (e.g., `POST` with `{"repo_url": "..."}` or no body).

### `/api/generate` response contract (from prompt 04)
> Paste the decided response shape here.

### Hardcoded sample input (from prompt 01)
> Paste the finalized hardcoded `RepoInput` values here.

## Your task

Implement `internal/handler/generate.go` — the HTTP handler for `POST /api/generate`.

## Requirements

### 1. Handler struct

```go
type GenerateHandler struct {
    agentClient *agent.Client
}

func NewGenerateHandler(agentClient *agent.Client) *GenerateHandler
```

### 2. `ServeHTTP` or handler function

The handler should:

1. **Validate the HTTP method** — return `405 Method Not Allowed` if not POST.
2. **Parse the request body** (if the contract from prompt 04 requires a body):
   - If the body contains `repo_url`, use it.
   - For phase 1, other fields are hardcoded regardless of what the body says.
   - If no body / body parsing fails, use the fully hardcoded sample.
3. **Build the `RepoInput`** — hardcoded sample data for phase 1. The `repo_url` may come from the request.
4. **Call `agentClient.Generate(ctx, input)`** with the request's context.
5. **On success**: return `200 OK` with `Content-Type: application/json` and the promotional JSON.
6. **On error**: return an appropriate status code with `{"error": "message"}`.

### 3. Hardcoded sample data

Use the finalized sample from prompt 01. It should be a package-level variable or a function that returns the default `RepoInput`:

```go
func defaultRepoInput() agent.RepoInput {
    return agent.RepoInput{
        RepoURL:          "https://github.com/AcaciaMan/village-square",
        RepoName:         "Village Square",
        ShortDescription: "...",
        ReadmeSummary:    "...",
        Topics:           []string{"go", "villagers", "cooperation"},
        Metrics:          agent.Metrics{Stars: 12, Forks: 3, ...},
        TargetChannel:    "twitter",
        TargetAudience:   "Villagers",
    }
}
```

### 4. Error response helper

Create a small helper to write JSON error responses:

```go
func writeError(w http.ResponseWriter, statusCode int, message string)
```

### 5. Logging

Add basic `log.Printf` statements for:
- Incoming request
- Agent call duration
- Errors

## Deliverables

1. **`internal/handler/generate.go`** — full, working Go code.
2. **Update to `main.go`** — if the handler wiring changed from the stub, show what to change.
3. **Verification command** — `go build ./...` should succeed after this prompt.

## Constraints

- Standard library only.
- The handler must compile with the existing `main.go` and `client.go`.
- Keep it simple — one file, no middleware, no routing library.
- Use `r.Context()` to pass context to the agent client (so request cancellation propagates).
- The hardcoded sample data is a temporary bridge — structure the code so replacing it with parsed request data in Phase 2 is a one-line change.
