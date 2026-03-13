# Prompt: Rate Limiter — Allow/Deny Time-Window Logic

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. I'm adding an in-memory rate limiter (Phase 1). The full intent is in `docs/intent-for-rate-limiter.md`.

- Prompt 44 created core types (`Limiter`, `clientState`, `BucketConfig`, `NewLimiter`).
- Prompt 45 added `ClientKeyFromRequest`.
- This prompt (46) replaces the `Allow` stub with the **real rolling-window logic**.

## Existing state

### `internal/ratelimit/ratelimit.go` (from prompt 44)

```go
type BucketConfig struct {
    Max    int
    Window time.Duration
}

type clientState struct {
    mu      sync.Mutex
    buckets map[string][]time.Time
}

type Limiter struct {
    mu      sync.Mutex
    clients map[string]*clientState
    configs map[string]BucketConfig
    nowFunc func() time.Time
}

// Stub from prompt 44:
func (l *Limiter) Allow(clientKey, bucket string) (bool, time.Duration) {
    return true, 0
}
```

## Your task

Replace the `Allow` stub with the real implementation.

## Requirements

### 1. Method signature (unchanged)

```go
func (l *Limiter) Allow(clientKey, bucket string) (bool, time.Duration)
```

### 2. Algorithm — rolling window with timestamp list

1. Look up the `BucketConfig` for `bucket`. If no config exists for this bucket, **fail open** — return `(true, 0)`. Log a warning on the first occurrence.
2. Get or create a `clientState` for `clientKey`:
   - Lock `l.mu` to access `l.clients`.
   - If no entry exists, create one with an initialized `buckets` map and insert it.
   - Unlock `l.mu` as soon as you have the `*clientState` pointer.
3. Lock `cs.mu` (client state mutex) to work with timestamp slices.
4. Get `now` from `l.nowFunc()`.
5. **Prune**: remove all timestamps older than `now - window` from the bucket's slice. Use an efficient approach — since timestamps are appended in order, find the first valid index and reslice.
6. **Check**: if `len(timestamps) >= config.Max`, the request is **denied**.
   - Compute `retryAfter = timestamps[0] + window - now` (time until the oldest entry exits the window).
   - Return `(false, retryAfter)`.
7. **Accept**: append `now` to the slice and return `(true, 0)`.
8. Unlock `cs.mu`.

### 3. Locking discipline

- **Global lock** (`l.mu`): held only to read/write the `clients` map. Never held during timestamp operations.
- **Per-client lock** (`cs.mu`): held during prune + check + append on that client's slices.
- This two-level scheme allows different clients to be rate-checked concurrently.

### 4. Fail-open policy

If the bucket name is not in `l.configs`, **allow the request**. During a hackathon demo, a misconfiguration should not block legitimate traffic. Log a warning so it's debuggable.

### 5. No allocations on hot path when possible

- Reuse the slice by reslicing (e.g., `timestamps = timestamps[i:]`) rather than allocating a new slice on each request.
- Don't copy the slice — reslice in place.

## Verification

- `go build ./internal/ratelimit/` compiles cleanly.
- The `Allow` method no longer returns the stub `(true, 0)` unconditionally.
- Only standard library imports (`sync`, `time`, `log`).
- No changes outside `internal/ratelimit/`.
