# Prompt: Rate Limiter — Unit Tests

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. I'm adding an in-memory rate limiter (Phase 1). The full intent is in `docs/intent-for-rate-limiter.md`.

Prompts 44–49 built and wired the complete rate limiter:
- `internal/ratelimit/ratelimit.go` — types, constructor, `Allow`, `Middleware`, `StartCleanup`, `ClientKeyFromRequest`
- `cmd/server/main.go` — wiring with `generate` (5/5min) and `search` (100/5min) buckets

This prompt (50) adds comprehensive **unit tests** in `internal/ratelimit/ratelimit_test.go`.

## Existing state

### Key APIs to test

```go
// Constructor
func NewLimiter(configs map[string]BucketConfig) *Limiter

// Core logic
func (l *Limiter) Allow(clientKey, bucket string) (bool, time.Duration)

// Client key extraction
func ClientKeyFromRequest(r *http.Request) string

// HTTP middleware
func (l *Limiter) Middleware(bucket string) func(http.Handler) http.Handler

// Background cleanup
func (l *Limiter) StartCleanup(interval time.Duration) (stop func())
```

### Testability hooks

- `l.nowFunc` — injectable clock. Tests can set this to a function that returns a controlled timestamp, then advance it to simulate time passing.

## Your task

Create `internal/ratelimit/ratelimit_test.go` with thorough unit tests.

## Required test cases

### 1. `TestAllow_GenerateBucket` — basic limit enforcement

- Create a limiter with `generate: {Max: 5, Window: 5 * time.Minute}`.
- Set `nowFunc` to return a fixed time.
- Call `Allow("client-a", "generate")` **5 times** — all should return `(true, 0)`.
- Call a 6th time — should return `(false, retryAfter)` where `retryAfter > 0`.
- Verify `retryAfter` is approximately 5 minutes (close to `window - 0` since all requests happened at the same fake "now").

### 2. `TestAllow_SearchBucket` — higher limit

- Create a limiter with `search: {Max: 100, Window: 5 * time.Minute}`.
- Call `Allow("client-a", "search")` **100 times** — all allowed.
- 101st call — denied with `retryAfter > 0`.

### 3. `TestAllow_WindowExpiry` — time-based recovery

- Create a limiter with `generate: {Max: 2, Window: 1 * time.Minute}` (small values for easy testing).
- Use a mutable `nowFunc`:
  ```go
  now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
  l.nowFunc = func() time.Time { return now }
  ```
- Make 2 calls (both allowed).
- 3rd call denied.
- Advance `now` by 61 seconds (`now = now.Add(61 * time.Second)`).
- 4th call should be **allowed** (old timestamps have fallen out of the window).

### 4. `TestAllow_ClientIsolation` — per-client independence

- Create a limiter with `generate: {Max: 2, Window: 5 * time.Minute}`.
- Exhaust `client-a`'s limit (2 calls, 3rd denied).
- `client-b` should still be allowed (first call returns `true`).

### 5. `TestAllow_UnknownBucket` — fail-open behavior

- Create a limiter with only `generate` configured.
- Call `Allow("client-a", "nonexistent")` — should return `(true, 0)`.

### 6. `TestAllow_RetryAfterAccuracy`

- Create a limiter with `generate: {Max: 2, Window: 1 * time.Minute}`.
- Use a mutable `nowFunc`.
- Make 1st call at `T+0`.
- Advance time by 20 seconds, make 2nd call.
- 3rd call denied. `retryAfter` should be approximately 40 seconds (until the 1st request at T+0 exits the window at T+60s, and current time is T+20s, so 60-20=40s).

### 7. `TestClientKeyFromRequest` — header parsing

Test the following scenarios by constructing `http.Request` objects with `httptest`:

| Scenario | Headers | RemoteAddr | Expected key |
|----------|---------|-----------|--------------|
| X-Forwarded-For single | `X-Forwarded-For: 203.0.113.5` | `10.0.0.1:1234` | `203.0.113.5` |
| X-Forwarded-For multiple | `X-Forwarded-For: 203.0.113.5, 70.41.3.18` | `10.0.0.1:1234` | `203.0.113.5` |
| X-Real-IP only | `X-Real-IP: 198.51.100.7` | `10.0.0.1:1234` | `198.51.100.7` |
| RemoteAddr fallback | (no proxy headers) | `192.168.1.1:54321` | `192.168.1.1` |
| IPv6 RemoteAddr | (no proxy headers) | `[::1]:8080` | `::1` |
| X-Forwarded-For with spaces | `X-Forwarded-For:  203.0.113.5 , 70.41.3.18 ` | `10.0.0.1:1234` | `203.0.113.5` |
| Empty X-Forwarded-For | `X-Forwarded-For: ` | `10.0.0.1:1234` | `10.0.0.1` |

Use table-driven tests.

### 8. `TestMiddleware_Returns429` — HTTP-level integration

- Create a limiter with `generate: {Max: 1, Window: 5 * time.Minute}`.
- Create a test handler (`httptest.NewServer` or `httptest.NewRecorder`).
- Wrap it with `limiter.Middleware("generate")`.
- Send 1st request — expect `200 OK`.
- Send 2nd request — expect:
  - Status `429`.
  - `Content-Type: application/json`.
  - `Retry-After` header present, value > 0.
  - Body contains `"error":"rate limit exceeded"` and `"retry_after_seconds"`.

### 9. `TestMiddleware_SkipsOptions`

- Create a limiter with `generate: {Max: 1, Window: 5 * time.Minute}`.
- Wrap a test handler with `Middleware("generate")`.
- Send an `OPTIONS` request — should pass through (200), not consume rate limit.
- Send a `POST` request — should be allowed (first real request).

### 10. `TestCleanup_RemovesStaleClients`

- Create a limiter with `generate: {Max: 5, Window: 1 * time.Minute}`.
- Use a mutable `nowFunc`.
- Make 1 request from `client-a` (to create an entry).
- Advance time by 2 minutes (past the window).
- Manually trigger cleanup by calling the unexported cleanup logic, or start the cleanup goroutine with a very short interval (e.g., 10ms) and wait briefly.
- Verify that `client-a` has been removed from the internal map.
  - If the `clients` map is unexported, verify indirectly: `Allow("client-a", "generate")` works fresh (the old state is gone).

### 11. `TestAllow_Concurrent` — race condition detection

- Create a limiter with `generate: {Max: 1000, Window: 5 * time.Minute}`.
- Launch 100 goroutines, each calling `Allow("client-X", "generate")` 10 times (total 1000 calls, exactly at the limit).
- Use `sync.WaitGroup` to coordinate.
- Verify:
  - No panics.
  - Exactly 1000 calls were allowed (the concurrent calls should serialize cleanly).
  - The test must **pass with `-race` flag** (`go test -race`).

## Test conventions

- Use standard `testing` package — no external test frameworks.
- Use `t.Run("subtest name", ...)` for grouping.
- Use table-driven tests where appropriate (especially for `ClientKeyFromRequest`).
- Helper function for creating a limiter with a fake clock:
  ```go
  func newTestLimiter(configs map[string]ratelimit.BucketConfig) (*ratelimit.Limiter, *time.Time) {
      l := ratelimit.NewLimiter(configs)
      now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
      l.NowFunc = func() time.Time { return now }  // if exported, or use a setter
      return l, &now
  }
  ```
  Note: if `nowFunc` is unexported, you may need to add a `SetNowFunc` method or export it. Decide what's cleanest — a simple exported field `NowFunc` is fine for a hackathon project.

## Verification

- `go test ./internal/ratelimit/ -v` — all tests pass.
- `go test ./internal/ratelimit/ -race -v` — all tests pass with race detector.
- `go test ./...` — full project tests still pass.
- No changes to files outside `internal/ratelimit/` (except possibly exporting `NowFunc` if needed for test access).
