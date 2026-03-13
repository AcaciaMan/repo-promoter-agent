# Prompt: Rate Limiter — Core Types and Package

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. All previous phases (prompts 01–43) are complete. Now I'm adding an **in-memory rate limiter** to protect the `POST /api/generate` and `GET /api/search` endpoints from abuse.

The full intent document is at `docs/intent-for-rate-limiter.md` — read it for high-level context, but note the following **deviations from the intent doc** that were decided during strategic review:

1. **`X-Forwarded-For` parsing is Phase 1** (moved from the intent's Phase 3) because the app runs behind DigitalOcean App Platform's reverse proxy. Without it, `RemoteAddr` returns the proxy IP and all clients share one rate limit.
2. **Stale client cleanup** is Phase 1 (missing from the intent doc entirely). Without it, the in-memory map grows unbounded.
3. **`Retry-After` header is mandatory**, not optional.

This prompt (44) creates the **core types and package skeleton**. Subsequent prompts will add:
- 45: client key extraction from request headers
- 46: allow/deny time-window logic
- 47: stale entry cleanup goroutine
- 48: HTTP middleware
- 49: wiring into `main.go`
- 50: unit tests

## Existing project state

```
cmd/server/main.go              # entry point — routes registered here
internal/agent/                  # agent clients
internal/github/                 # GitHub API client
internal/handler/generate.go     # POST /api/generate handler
internal/handler/search.go       # GET /api/search handler
internal/store/                  # SQLite persistence
```

The rate limiter will live in a **new package**: `internal/ratelimit/`.

## Your task

Create `internal/ratelimit/ratelimit.go` with the core type definitions. This file defines the structures and constructor only — **no logic yet** (that comes in prompts 45–47).

## Requirements

### 1. `BucketConfig` struct

```go
type BucketConfig struct {
    Max    int           // maximum requests allowed in the window
    Window time.Duration // rolling window duration
}
```

### 2. `clientState` struct (unexported)

Holds per-client, per-bucket timestamp slices:

```go
type clientState struct {
    mu      sync.Mutex
    buckets map[string][]time.Time // key = bucket name ("generate", "search")
}
```

Use a `map[string][]time.Time` so bucket names are flexible (not hardcoded to just two).

### 3. `Limiter` struct

```go
type Limiter struct {
    mu      sync.Mutex
    clients map[string]*clientState // key = client identifier (IP)
    configs map[string]BucketConfig // key = bucket name
    nowFunc func() time.Time        // injectable clock for testing; defaults to time.Now
}
```

Key design decisions:
- **`nowFunc`**: allows tests to inject a fake clock without modifying real time. Default to `time.Now`.
- **Two-level locking**: global `mu` protects the `clients` map. Per-client `clientState.mu` protects the timestamp slices. This avoids holding the global lock during the timestamp pruning/append work.
- **`configs`**: maps bucket names to their limits, set at construction time.

### 4. Constructor

```go
func NewLimiter(configs map[string]BucketConfig) *Limiter
```

- Stores the configs.
- Initializes the `clients` map.
- Sets `nowFunc` to `time.Now`.

### 5. Stub methods

Add these method signatures with `// TODO: implement in prompt XX` comments so the package compiles and the next prompts can fill them in:

```go
// Allow checks whether a request from the given client for the given bucket
// should be allowed. Returns true if allowed, or false with a retryAfter
// duration indicating when the client can retry.
// TODO: implement in prompt 46
func (l *Limiter) Allow(clientKey, bucket string) (bool, time.Duration) {
    return true, 0
}

// Middleware returns an http.Handler middleware that enforces rate limits
// for the given bucket.
// TODO: implement in prompt 48
func (l *Limiter) Middleware(bucket string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler { return next }
}

// StartCleanup begins a background goroutine that periodically removes
// stale client entries. Returns a stop function.
// TODO: implement in prompt 47
func (l *Limiter) StartCleanup(interval time.Duration) (stop func()) {
    return func() {}
}
```

## Verification

- `go build ./internal/ratelimit/` compiles with no errors.
- `go vet ./internal/ratelimit/` reports no issues.
- The package imports only `net/http`, `sync`, and `time` (no external dependencies).
