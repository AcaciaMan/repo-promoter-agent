# Prompt: Rate Limiter — Stale Entry Cleanup

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. I'm adding an in-memory rate limiter (Phase 1). The full intent is in `docs/intent-for-rate-limiter.md`.

- Prompt 44 created core types (including a `StartCleanup` stub).
- Prompt 46 implemented the `Allow` method with rolling-window logic.
- This prompt (47) replaces the `StartCleanup` stub with a real **background cleanup goroutine**.

**Why this is needed**: The `Allow` method prunes timestamps for a client when that client makes a request. But clients who stop making requests leave stale entries in the `clients` map forever. Over time (even during a hackathon demo day with many users), the map grows unbounded. A periodic goroutine removes these dead entries.

## Existing state

### `internal/ratelimit/ratelimit.go`

```go
type Limiter struct {
    mu      sync.Mutex
    clients map[string]*clientState
    configs map[string]BucketConfig
    nowFunc func() time.Time
}

// Stub from prompt 44:
func (l *Limiter) StartCleanup(interval time.Duration) (stop func()) {
    return func() {}
}
```

The `clientState` struct has:
```go
type clientState struct {
    mu      sync.Mutex
    buckets map[string][]time.Time
}
```

## Your task

Replace the `StartCleanup` stub with a real implementation.

## Requirements

### 1. Method signature (unchanged)

```go
func (l *Limiter) StartCleanup(interval time.Duration) (stop func())
```

### 2. Cleanup goroutine behavior

1. Start a goroutine that runs every `interval` (e.g., every 10 minutes).
2. Use `time.NewTicker(interval)` and a `done` channel.
3. On each tick:
   - Lock `l.mu`.
   - Iterate over all entries in `l.clients`.
   - For each client, lock `cs.mu` and check all bucket slices.
   - Prune timestamps older than `now - window` (use the corresponding `BucketConfig` window for each bucket).
   - If **all** bucket slices for a client are empty after pruning, **delete** the client entry from the map.
   - Unlock `cs.mu` before moving to the next client.
   - Unlock `l.mu` after the full sweep.
4. Log the number of evicted clients if any were removed (for debuggability).

### 3. Stop function

The returned `stop` function should:
- Close the `done` channel (use `sync.Once` to make it safe to call multiple times).
- The goroutine exits when `done` is closed.

### 4. Locking order

Same discipline as `Allow`:
- Acquire `l.mu` first, then `cs.mu` per client.
- Never hold `cs.mu` while acquiring `l.mu`.
- To avoid holding `l.mu` for too long during a large sweep, consider collecting keys to delete during iteration and deleting them in a second pass (still under `l.mu`). Or keep it simple since the map will be small during a hackathon. Use your judgment — simplicity wins.

### 5. Caller responsibility

The caller (`main.go` in prompt 49) will call:
```go
stopCleanup := limiter.StartCleanup(10 * time.Minute)
defer stopCleanup()
```

## Verification

- `go build ./internal/ratelimit/` compiles cleanly.
- The `StartCleanup` method no longer returns a no-op stub.
- Only standard library imports (`sync`, `time`, `log`).
- No changes outside `internal/ratelimit/`.
