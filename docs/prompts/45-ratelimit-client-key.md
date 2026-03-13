# Prompt: Rate Limiter — Client Key Extraction

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. I'm adding an in-memory rate limiter (Phase 1). The full intent is in `docs/intent-for-rate-limiter.md`.

**Critical context**: This app runs behind **DigitalOcean App Platform's reverse proxy**. `r.RemoteAddr` returns the proxy's IP, not the real client's IP. Without proper header parsing, all users share a single rate limit bucket.

Prompt 44 created the core types in `internal/ratelimit/ratelimit.go`. This prompt (45) adds the **client key extraction** function. It has **no dependency** on the Allow logic (prompt 46) — it's a standalone utility.

## Existing state

### `internal/ratelimit/ratelimit.go` (created in prompt 44)

Contains:
- `Limiter` struct with `clients`, `configs`, `nowFunc`, and `mu`
- `clientState` struct with per-bucket timestamp slices
- `BucketConfig` struct
- `NewLimiter(configs)` constructor
- Stub methods: `Allow`, `Middleware`, `StartCleanup`

### How the existing codebase extracts client info

In `internal/handler/generate.go`:
```go
log.Printf("POST /api/generate from %s", r.RemoteAddr)
```
This uses raw `r.RemoteAddr` (IP:port) — fine for logging, but wrong for rate limiting behind a proxy.

## Your task

Add a `ClientKeyFromRequest` function to `internal/ratelimit/ratelimit.go` (or a new file `internal/ratelimit/clientkey.go` if you prefer separation).

## Requirements

### 1. Function signature

```go
// ClientKeyFromRequest extracts a client identifier from the request,
// suitable for use as a rate-limiting key.
func ClientKeyFromRequest(r *http.Request) string
```

Exported so it can be tested directly and potentially reused.

### 2. Header priority

Check headers in this order, using the **first non-empty value** found:

1. `X-Forwarded-For` — take the **leftmost (first) IP** in the comma-separated list. This is the original client IP set by the first proxy. Trim whitespace.
2. `X-Real-IP` — common alternative set by some proxies. Use as-is after trimming.
3. `r.RemoteAddr` — fallback for direct connections (local development).

### 3. Port stripping

`r.RemoteAddr` includes a port (e.g., `192.168.1.1:54321`). Strip the port using `net.SplitHostPort`. If `SplitHostPort` fails (e.g., the value is already just an IP), use the value as-is.

For `X-Forwarded-For` and `X-Real-IP`, IPs typically don't include ports, but strip the port defensively if present.

### 4. Edge cases

- Empty `X-Forwarded-For` with whitespace: treat as empty, fall through.
- `X-Forwarded-For: , , ` (all empty entries): fall through to `X-Real-IP`.
- IPv6 addresses in `RemoteAddr` are wrapped in brackets (e.g., `[::1]:8080`). `net.SplitHostPort` handles this correctly.

### 5. No validation

Do **not** validate that the extracted string is a real IP address. Just use it as a map key. Validation would add complexity with no benefit for rate limiting.

## Verification

- `go build ./internal/ratelimit/` compiles cleanly.
- The function uses only `net` and `net/http` from the standard library (plus `strings` for trimming/splitting).
- No changes to any file outside `internal/ratelimit/`.
