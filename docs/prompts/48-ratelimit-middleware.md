# Prompt: Rate Limiter — HTTP Middleware

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. I'm adding an in-memory rate limiter (Phase 1). The full intent is in `docs/intent-for-rate-limiter.md`.

- Prompt 44 created core types (including a `Middleware` stub).
- Prompt 45 added `ClientKeyFromRequest`.
- Prompt 46 implemented the `Allow` method.
- This prompt (48) replaces the `Middleware` stub with the real HTTP middleware.

## Existing state

### `internal/ratelimit/ratelimit.go`

Key pieces available:

```go
// Extracts client IP from request headers (prompt 45)
func ClientKeyFromRequest(r *http.Request) string

// Checks rate limit, returns (allowed, retryAfter) (prompt 46)
func (l *Limiter) Allow(clientKey, bucket string) (bool, time.Duration)

// Stub from prompt 44:
func (l *Limiter) Middleware(bucket string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler { return next }
}
```

### Existing error response pattern

In `internal/handler/generate.go`:

```go
func writeError(w http.ResponseWriter, statusCode int, message string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(statusCode)
    json.NewEncoder(w).Encode(map[string]string{"error": message})
}
```

The rate limiter will define its own JSON response writer since it's in a different package, but the response format should be **aligned** with the existing pattern.

## Your task

Replace the `Middleware` stub with the real implementation.

## Requirements

### 1. Method signature (unchanged)

```go
func (l *Limiter) Middleware(bucket string) func(http.Handler) http.Handler
```

This returns standard Go middleware — a function that wraps an `http.Handler`.

### 2. Middleware behavior

For each incoming request:

1. **Skip `OPTIONS` requests** — browser CORS preflight requests should not consume rate limit budget. Call `next.ServeHTTP` directly.
2. **Extract client key** using `ClientKeyFromRequest(r)`.
3. **Call `l.Allow(clientKey, bucket)`**.
4. **If allowed**: call `next.ServeHTTP(w, r)` (pass through to the real handler).
5. **If denied**: write a `429 Too Many Requests` response:
   - Set `Content-Type: application/json`.
   - Set `Retry-After` header (value = `retryAfter` in whole seconds, **rounded up** — use `math.Ceil` or integer arithmetic).
   - Write status code `429`.
   - Write JSON body:
     ```json
     {
       "error": "rate limit exceeded",
       "retry_after_seconds": 42
     }
     ```
   - The `retry_after_seconds` field is an integer matching the `Retry-After` header value.

### 3. JSON response struct

Define a small unexported struct for the 429 response body:

```go
type rateLimitError struct {
    Error            string `json:"error"`
    RetryAfterSeconds int   `json:"retry_after_seconds"`
}
```

### 4. Logging

When a request is rate-limited, log a concise message:

```
rate limited: bucket=generate client=203.0.113.5 (count at limit)
```

Include bucket and client key. Keep it to one line — this will be useful for debugging during demos but shouldn't be noisy.

### 5. No dependency on handler package

The middleware is in `internal/ratelimit/`, not `internal/handler/`. It must **not** import the handler package. It writes its own JSON response for 429s.

## Verification

- `go build ./internal/ratelimit/` compiles cleanly.
- The `Middleware` method no longer returns a pass-through stub.
- Imports: `encoding/json`, `log`, `math`, `net/http`, `time` — all standard library.
- No changes outside `internal/ratelimit/`.
