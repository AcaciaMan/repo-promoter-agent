# Prompt: Evolve the /api/generate Handler — Wire GitHub + Storage

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. Phase 1 is done. I'm completing Phase 2 by wiring the new components into the handler.

## Current project state

```
cmd/server/main.go
internal/agent/client.go         # Gradient agent client (done)
internal/github/client.go        # GitHub API client (prompt 09)
internal/handler/generate.go     # THIS FILE — needs evolution
internal/store/store.go          # SQLite storage (prompt 10)
static/index.html
```

### Current handler code (`internal/handler/generate.go`)

```go
package handler

import (
    "encoding/json"
    "log"
    "net/http"
    "time"
    "repo-promoter-agent/internal/agent"
)

type GenerateHandler struct {
    agentClient *agent.Client
}

func NewGenerateHandler(agentClient *agent.Client) *GenerateHandler {
    return &GenerateHandler{agentClient: agentClient}
}

type generateRequest struct {
    RepoURL        string `json:"repo_url"`
    TargetChannel  string `json:"target_channel"`
    TargetAudience string `json:"target_audience"`
}

func (h *GenerateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        writeError(w, http.StatusMethodNotAllowed, "method not allowed")
        return
    }
    log.Printf("POST /api/generate from %s", r.RemoteAddr)

    input := defaultRepoInput()
    var req generateRequest
    if r.Body != nil {
        if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
            if req.RepoURL != "" {
                input.RepoURL = req.RepoURL
            }
            if req.TargetChannel != "" {
                input.TargetChannel = req.TargetChannel
            }
            if req.TargetAudience != "" {
                input.TargetAudience = req.TargetAudience
            }
        }
    }

    start := time.Now()
    result, err := h.agentClient.Generate(r.Context(), input)
    elapsed := time.Since(start)
    if err != nil {
        log.Printf("Agent call failed after %s: %v", elapsed, err)
        writeError(w, http.StatusBadGateway, "agent request failed: "+err.Error())
        return
    }
    log.Printf("Agent call succeeded in %s (%d bytes)", elapsed, len(result))

    w.Header().Set("Content-Type", "application/json")
    w.Write(result)
}

func defaultRepoInput() agent.RepoInput { /* hardcoded sample */ }

func writeError(w http.ResponseWriter, statusCode int, message string) { /* ... */ }
```

## Your task

Evolve the handler to:
1. **Fetch real repo data** from GitHub when a `repo_url` is provided.
2. **Store the generated promotion** in SQLite after a successful agent call.
3. **Keep the hardcoded fallback** when no `repo_url` is provided (for quick testing).

## Requirements

### 1. Updated `GenerateHandler` struct

```go
type GenerateHandler struct {
    agentClient  *agent.Client
    githubClient *github.Client
    store        *store.Store
}
```

New constructor:
```go
func NewGenerateHandler(agentClient *agent.Client, githubClient *github.Client, store *store.Store) *GenerateHandler
```

### 2. Updated `ServeHTTP` flow

```
1. Parse request body (repo_url, target_channel, target_audience)
2. IF repo_url is provided:
     a. Call githubClient.FetchRepo(ctx, repoURL) to get real data
     b. If GitHub fetch fails, return 422 with clear error
     c. Set target_channel and target_audience from request
   ELSE:
     a. Use defaultRepoInput() (hardcoded sample for testing)
3. Call agentClient.Generate(ctx, input)
4. Parse agent's json.RawMessage into store.Promotion
5. Set target_channel and target_audience on the Promotion
6. Call store.Save(ctx, &promotion)
7. Return the promotion as JSON (now enriched with id and created_at)
```

### 3. Response shape change

The response should now return the **stored promotion** (with `id` and `created_at`) instead of raw agent output. This is a small but important change — the frontend can now reference promotions by ID.

Before (Phase 1): returns raw `json.RawMessage` from agent.
After (Phase 2): returns `store.Promotion` marshaled to JSON (includes `id`, `created_at`, and all content fields).

### 4. Input validation

- `repo_url`: if provided, must look like a GitHub URL. The `github.Client.FetchRepo` will do the real validation, but reject obviously wrong input early (e.g., empty string after whitespace trim is fine — means "use default").
- `target_channel`: if provided, should be one of `"twitter"`, `"linkedin"`, `"general"`, or empty (default to `"general"`).
- `target_audience`: free text, optional, no validation needed.

### 5. Store save failure handling

If the agent call succeeds but the store save fails:
- Log the error.
- Still return the promotional JSON to the user (don't lose the generated content).
- Include a warning in the response or log. Decide the best approach.

### 6. Updated `main.go`

Show the full updated `main.go` that:
- Creates the GitHub client.
- Creates the store (with `DB_PATH` env var).
- Passes all three dependencies to `NewGenerateHandler`.
- Adds `defer store.Close()`.
- Updates `.env.example` with the new `DB_PATH` variable.

## Deliverables

1. **Updated `internal/handler/generate.go`** — full file, not just a diff.
2. **Updated `cmd/server/main.go`** — full file, not just a diff.
3. **Updated `.env.example`** — with `DB_PATH` added.
4. **Verification** — `go build ./...` must succeed.

## Constraints

- Keep `writeError` and `defaultRepoInput` in the same file.
- Don't restructure other files — only modify `generate.go` and `main.go`.
- The handler should still work if `repo_url` is omitted (backward compatible with the Phase 1 HTML page).
- The store save is best-effort — never fail the request because of a DB error.
- Use `r.Context()` consistently for all downstream calls (GitHub, agent, store).
