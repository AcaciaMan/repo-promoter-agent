# Prompt: Rate Limiter Polish — Environment Variable Config

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. Phase 1 of the rate limiter is complete (prompts 44–50). Prompt 51 enhanced logging and changed `Allow` to return an `AllowResult` struct. This is **Phase 2, prompt 2 of 3**.

The full intent is in `docs/intent-for-rate-limiter.md`.

## Current state

### `cmd/server/main.go` — hardcoded rate limit config

```go
limiter := ratelimit.NewLimiter(map[string]ratelimit.BucketConfig{
    "generate": {Max: 5, Window: 5 * time.Minute},
    "search":   {Max: 100, Window: 5 * time.Minute},
})
stopCleanup := limiter.StartCleanup(10 * time.Minute)
defer stopCleanup()
log.Println("Rate limiter enabled: generate=5/5m0s, search=100/5m0s")
```

The limits are hardcoded. If we want to tune them during a hackathon demo (e.g., increase `generate` to 20 for testing), we'd have to recompile.

## Your task

Add optional environment variable overrides for rate limit maximums, so they can be tuned without recompilation.

## Requirements

### 1. Environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `RATE_LIMIT_GENERATE_MAX` | `5` | Max generate requests per client per 5 minutes |
| `RATE_LIMIT_SEARCH_MAX` | `100` | Max search requests per client per 5 minutes |

- Only the **max count** is configurable via env var. The **window duration** (5 minutes) stays hardcoded — changing the window is a more fundamental behavioral change that shouldn't be tweaked casually.
- If the env var is set but contains an invalid integer (e.g., `"abc"`), **log a warning and use the default**. Do not crash the server over a bad rate limit config.
- If the env var is set to `0`, treat it as "disable rate limiting for this bucket" — set `Max` to a very high number (e.g., `math.MaxInt32`) rather than literally zero (which would block everything).

### 2. Implementation location

Add the parsing logic in `cmd/server/main.go`, **not** in the ratelimit package. The ratelimit package should remain configuration-agnostic — it just takes a `BucketConfig` map.

Add a small helper function in `main.go`:

```go
// envIntOr reads an integer from the named env var, or returns the default.
// Logs a warning if the value is set but not a valid integer.
func envIntOr(key string, defaultVal int) int
```

### 3. Update limiter creation

Replace the hardcoded config with:

```go
generateMax := envIntOr("RATE_LIMIT_GENERATE_MAX", 5)
searchMax := envIntOr("RATE_LIMIT_SEARCH_MAX", 100)

limiter := ratelimit.NewLimiter(map[string]ratelimit.BucketConfig{
    "generate": {Max: generateMax, Window: 5 * time.Minute},
    "search":   {Max: searchMax, Window: 5 * time.Minute},
})
```

### 4. Update the startup log line

Make the log line reflect actual configured values (not always "5" and "100"):

```go
log.Printf("Rate limiter enabled: generate=%d/5m0s, search=%d/5m0s", generateMax, searchMax)
```

### 5. No changes to the ratelimit package

All changes are in `cmd/server/main.go` only.

## Verification

- `go build ./cmd/server/` compiles cleanly.
- Starting the server **without** env vars shows: `Rate limiter enabled: generate=5/5m0s, search=100/5m0s` (defaults).
- Starting with `RATE_LIMIT_GENERATE_MAX=20` shows: `Rate limiter enabled: generate=20/5m0s, search=100/5m0s`.
- Starting with `RATE_LIMIT_GENERATE_MAX=abc` shows a warning and uses default 5.
- Starting with `RATE_LIMIT_GENERATE_MAX=0` effectively disables generate rate limiting.
- `go test ./...` — all tests pass (no test changes needed — tests use `NewLimiter` directly).
