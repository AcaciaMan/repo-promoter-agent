# Prompt: Wire Analysis Agent Env Vars into main.go

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. I've provisioned an Analysis Agent on Gradient and documented its env vars in `.env.example` and `README.md` (see prompts 28 and 29).

Now I need to update `cmd/server/main.go` to read the new optional env vars and create the Analysis Agent client when configured. The analysis client doesn't need to be wired to any handler yet — that happens in a later phase.

The full intent document is at `docs/intent-for-analysis-agent.md` — read it for high-level context.

## Current `cmd/server/main.go`

```go
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	"repo-promoter-agent/internal/agent"
	"repo-promoter-agent/internal/github"
	"repo-promoter-agent/internal/handler"
	"repo-promoter-agent/internal/store"
	"repo-promoter-agent/static"
)

func main() {
	// Load .env file (fail gracefully if it doesn't exist).
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables directly")
	}

	// Required env vars — fail fast if missing.
	endpoint := mustEnv("AGENT_ENDPOINT")
	accessKey := mustEnv("AGENT_ACCESS_KEY")

	// Optional env vars.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "promotions.db"
	}

	// Create store.
	st, err := store.New(dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer st.Close()

	// Create clients.
	agentClient := agent.NewClient(endpoint, accessKey)

	ghToken := os.Getenv("GITHUB_TOKEN")
	if ghToken != "" {
		log.Println("GitHub token configured — authenticated API access enabled")
	} else {
		log.Println("No GITHUB_TOKEN set — using unauthenticated GitHub API (60 req/hr limit)")
	}
	githubClient := github.NewClient(ghToken)

	// Set up routes.
	mux := http.NewServeMux()
	mux.Handle("/api/generate", handler.NewGenerateHandler(agentClient, githubClient, st))
	mux.Handle("/api/search", handler.NewSearchHandler(st))
	mux.Handle("/", noCacheHandler(http.FileServerFS(static.Files)))

	addr := ":" + port
	log.Printf("Server listening on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
```

## Current agent client (`internal/agent/client.go`)

The existing `agent.Client` is constructed with:

```go
func NewClient(endpoint, accessKey string) *Client {
    return &Client{
        endpoint:   endpoint,
        accessKey:  accessKey,
        httpClient: &http.Client{Timeout: 60 * time.Second},
    }
}
```

The Analysis Agent client does not exist yet — it will be created in Phase 1. For this prompt, we only need to **read the env vars** and store them for later use.

## Your task

### 1. Read the optional Analysis Agent env vars in `main.go`

After the existing `githubClient` creation block, add a new block that reads the Analysis Agent env vars:

```go
// Analysis Agent (optional — enables repo analysis feature).
analysisEndpoint := os.Getenv("ANALYSIS_AGENT_ENDPOINT")
analysisKey := os.Getenv("ANALYSIS_AGENT_ACCESS_KEY")
if analysisEndpoint != "" && analysisKey != "" {
    log.Println("Analysis Agent configured — repo analysis feature enabled")
} else {
    log.Println("Analysis Agent not configured — repo analysis feature disabled (set ANALYSIS_AGENT_ENDPOINT and ANALYSIS_AGENT_ACCESS_KEY to enable)")
}
```

### 2. Create the analysis client conditionally

Create an `*agent.Client` for the Analysis Agent only when both env vars are set. Use `nil` to represent "not configured":

```go
var analysisClient *agent.Client
if analysisEndpoint != "" && analysisKey != "" {
    analysisClient = agent.NewClient(analysisEndpoint, analysisKey)
}
```

**Important:** The Analysis Agent will eventually use a dedicated client type with a 30-second timeout (created in Phase 1). For now, reuse `agent.NewClient` as a placeholder. The variable name `analysisClient` is established here and will be updated to the proper type in Phase 1 without changing `main.go` structure.

### 3. Pass `analysisClient` to the generate handler (prepare the signature, do NOT break compilation)

The `GenerateHandler` needs to accept the analysis client. Update `NewGenerateHandler` to accept an optional analysis client parameter:

**In `internal/handler/generate.go`:**

- Add an `analysisClient *agent.Client` field to the `GenerateHandler` struct.
- Update `NewGenerateHandler` to accept a fourth parameter: `analysisClient *agent.Client` (can be `nil`).
- Do NOT use the analysis client anywhere in the handler logic yet — just store it.

**In `cmd/server/main.go`:**

- Pass `analysisClient` (which may be `nil`) to `NewGenerateHandler`:

```go
mux.Handle("/api/generate", handler.NewGenerateHandler(agentClient, githubClient, st, analysisClient))
```

## What NOT to do

- Do NOT implement the analysis client type or `CallAnalysisAgent` function (that's Phase 1).
- Do NOT add any analysis logic to the generate handler flow.
- Do NOT modify the search handler, store, or frontend.
- Do NOT make the Analysis Agent env vars required — the app must start and work exactly as before when they're not set.

## Verification

After implementation:

1. `go build ./...` compiles without errors.
2. With no `ANALYSIS_AGENT_ENDPOINT` set, the app starts normally with log message: "Analysis Agent not configured — repo analysis feature disabled..."
3. With both `ANALYSIS_AGENT_ENDPOINT` and `ANALYSIS_AGENT_ACCESS_KEY` set, the app starts with log message: "Analysis Agent configured — repo analysis feature enabled"
4. `/api/generate` and `/api/search` work exactly as before — no behavioral changes.
5. The `GenerateHandler` struct has an `analysisClient` field but doesn't use it yet.
