# Prompt: Rate Limiter — Wire into Main

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. I'm adding an in-memory rate limiter (Phase 1). The full intent is in `docs/intent-for-rate-limiter.md`.

- Prompts 44–48 built the rate limiter package (`internal/ratelimit/`):
  - `NewLimiter(configs)` constructor
  - `ClientKeyFromRequest(r)` for IP extraction behind proxies
  - `Allow(clientKey, bucket)` rolling-window logic
  - `StartCleanup(interval)` stale entry eviction
  - `Middleware(bucket)` HTTP middleware returning 429 with JSON + `Retry-After`
- This prompt (49) wires the limiter into `cmd/server/main.go`.

## Existing `cmd/server/main.go`

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
	// ... env loading, client creation (omitted for brevity) ...

	// Set up routes.
	mux := http.NewServeMux()
	mux.Handle("/api/generate", handler.NewGenerateHandler(agentClient, githubClient, st, analysisClient))
	mux.Handle("/api/search", handler.NewSearchHandler(st))
	mux.Handle("/", noCacheHandler(http.FileServerFS(static.Files)))

	addr := ":" + port
	log.Printf("Server listening on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
```

## Your task

Modify `cmd/server/main.go` to create the rate limiter, start its cleanup goroutine, and wrap the existing handlers with rate limit middleware.

## Requirements

### 1. Import the ratelimit package

```go
import "repo-promoter-agent/internal/ratelimit"
```

### 2. Create the limiter (after client creation, before route setup)

```go
limiter := ratelimit.NewLimiter(map[string]ratelimit.BucketConfig{
    "generate": {Max: 5, Window: 5 * time.Minute},
    "search":   {Max: 100, Window: 5 * time.Minute},
})
```

Add `"time"` to imports.

### 3. Start cleanup goroutine

```go
stopCleanup := limiter.StartCleanup(10 * time.Minute)
defer stopCleanup()
```

Place this right after limiter creation, before route setup.

### 4. Wrap handlers with middleware

Change the route registration to wrap each handler:

```go
mux.Handle("/api/generate", limiter.Middleware("generate")(handler.NewGenerateHandler(agentClient, githubClient, st, analysisClient)))
mux.Handle("/api/search", limiter.Middleware("search")(handler.NewSearchHandler(st)))
```

**Do NOT wrap the static file route** (`/`). Serving HTML/CSS/JS doesn't need rate limiting.

### 5. Log the rate limit configuration

After creating the limiter, log:

```
Rate limiter enabled: generate=5/5m0s, search=100/5m0s
```

This makes it visible in server startup logs that rate limiting is active, and with what thresholds.

### 6. No other changes

- Don't modify the handler package.
- Don't modify the ratelimit package.
- Don't add environment variable overrides for rate limits yet (that's Phase 2).
- Keep the existing `noCacheHandler` on the static route unchanged.

## Verification

- `go build ./cmd/server/` compiles cleanly.
- `go build ./...` compiles cleanly (full project).
- Starting the server (`go run ./cmd/server/main.go`) shows the rate limiter log line.
- The static file route `/` is NOT wrapped with rate limiting.
