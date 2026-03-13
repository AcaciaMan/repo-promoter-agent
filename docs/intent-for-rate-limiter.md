# Intent – In‑Memory Go Server Rate Limiter

This document describes the intent, scope, and phased implementation plan for adding an in‑memory rate limiter to the Go backend used in the DigitalOcean Gradient AI Hackathon project. The backend exposes `POST /api/generate` and `GET /api/search` endpoints and integrates with GitHub, a Gradient AI agent, and SQLite FTS storage. 

---

## Objectives

- Protect the `POST /api/generate` endpoint from abuse and accidental overuse while still allowing reasonable experimentation during the hackathon. 
- Protect the `GET /api/search` endpoint so the SQLite FTS and overall service remain responsive under repeated searches. 
- Keep the implementation **in‑memory**, simple, and dependency‑light, so it fits well into a single Go service deployed on DigitalOcean App Platform.
- Make the limiter behavior explicit and testable to avoid debugging surprises during live demos.

---

## High‑Level Requirements

- Rate limits:
  - `POST /api/generate`: max 5 requests per client in any rolling 5‑minute window.
  - `GET /api/search`: max 100 requests per client in any rolling 5‑minute window.
- Limiting scope:
  - Per‑client key, from one of:
    - IP address (for MVP), or
    - API key / session identifier (easy to swap later).
- Behavior when limit exceeded:
  - Return `429 Too Many Requests`.
  - Include a JSON error body with a human‑readable message.
  - Optionally include a `Retry-After` header (seconds until the oldest relevant request leaves the window).
- No external infrastructure:
  - No Redis or external cache; pure in‑memory data structures in the Go process.
  - Safe for single‑instance deployment on App Platform; multi‑instance consistency is explicitly out of scope for MVP.

---

## Design Overview

### Rate Limiting Strategy

- Use a rolling window algorithm per client and per endpoint category:
  - For each `(clientKey, bucket)` pair, keep timestamps of recent requests.
  - Buckets: `generate`, `search`.
- On each request:
  - Compute `clientKey` and `bucket`.
  - Drop timestamps older than `now - 5 minutes`.
  - If the remaining count is below the allowed limit, accept:
    - Append current timestamp and continue handler.
  - If the count is at or above the limit, reject with HTTP 429.

### Data Structures

- Central in‑memory store:
  - `map[string]*ClientBucketState`, keyed by `clientKey`.
- Each `ClientBucketState` contains:
  - `generate` slice of timestamps.
  - `search` slice of timestamps.
- Concurrency:
  - Protect the map with `sync.RWMutex` or use `sync.Map`.
  - Inside each client state, protect slices with per‑client mutex if needed, or hold the global lock for simplicity.

### Integration with Existing Handlers

- Implement a reusable middleware:
  - `RateLimitMiddleware(bucket string, max int, window time.Duration)`.
- Wrap handlers:
  - `POST /api/generate` → `RateLimitMiddleware("generate", 5, 5*time.Minute)`.
  - `GET /api/search` → `RateLimitMiddleware("search", 100, 5*time.Minute)`.
- The middleware:
  - Extracts `clientKey` (initially from `RemoteAddr` or `X-Forwarded-For` if present).
  - Checks and updates rate limit state.
  - Calls the underlying handler only if allowed.

---

## Phase 1 – Minimal Working Limiter

### Goal

Get a simple, correct, single‑process in‑memory limiter working for both endpoints, with tests and clear behavior, without over‑optimizing.

### Tasks for Claude Opus

1. **Define core types**
   - Create a `rateLimiter` struct:
     - Holds the `clients` map and a mutex.
     - Stores configuration: limits and window duration per bucket.
   - Define a `clientState` struct with timestamp slices for `generate` and `search`.

2. **Implement time‑window logic**
   - Implement a helper `allow(clientKey, bucket string) (allowed bool, retryAfter time.Duration)` that:
     - Cleans out timestamps older than the 5‑minute window.
     - Compares the length of the remaining slice with the corresponding limit.
     - If allowed, appends `time.Now()` and returns `(true, 0)`.
     - If not allowed, returns `(false, retryAfter)` where `retryAfter` is the time until the oldest entry falls out of the window.

3. **Build HTTP middleware**
   - Implement `func (rl *rateLimiter) Middleware(bucket string, max int, window time.Duration) func(http.Handler) http.Handler`.
   - In the middleware:
     - Compute `clientKey` from `http.Request`.
     - Call `allow(clientKey, bucket)`.
     - If blocked, write:
       - `StatusTooManyRequests` (429).
       - `Content-Type: application/json`.
       - Optional `Retry-After` header in seconds.
       - JSON body: `{"error":"rate_limit_exceeded","message":"...","retry_after_seconds":N}`.
     - If allowed, call `next.ServeHTTP`.

4. **Wire into routes**
   - Wrap existing handlers:
     - `generateHandler` with `Middleware("generate", 5, 5*time.Minute)`.
     - `searchHandler` with `Middleware("search", 100, 5*time.Minute)`.

5. **Add unit tests**
   - Test per‑bucket behavior:
     - 5 allowed `generate` calls in quick succession, 6th is blocked.
     - 100 allowed `search` calls, 101st is blocked.
   - Test that after artificially advancing time (by injecting a `timeNow` function or using a test‑only clock), requests are allowed again after 5 minutes.
   - Test that `clientA` hitting the limit does not affect `clientB`.

---

## Phase 2 – Robustness and Observability

### Goal

Improve reliability, debuggability, and performance without changing external behavior.

### Tasks for Claude Opus

1. **Refine data structure behavior**
   - Ensure slices are pruned efficiently (avoid unbounded growth).
   - Consider switching from raw slices to a ring buffer if performance profiling shows need; otherwise keep slices for simplicity.

2. **Add logging hooks**
   - Log a concise message when a request is blocked:
     - Endpoint, client key, current count, limit, bucket.
   - Ensure logs are suitable for debugging in the hackathon environment but not too noisy.

3. **Add metrics hooks (optional)**
   - Expose counters (even if only via logs for now) for:
     - Total requests per bucket.
     - Blocked requests per bucket.

4. **Configuration consolidation**
   - Centralize rate limit configuration in one struct or function so changing thresholds is a single‑point edit.
   - Optionally allow overrides via environment variables (e.g., `RATE_LIMIT_GENERATE`, `RATE_LIMIT_SEARCH`) for tuning in different environments.

---

## Phase 3 – Edge Cases and Future‑Proofing

### Goal

Handle tricky scenarios and prepare for potential extensions beyond the hackathon MVP.

### Tasks for Claude Opus

1. **IP and header handling**
   - Improve `clientKey` extraction:
     - First check `X-Forwarded-For` and `X-Real-IP`.
     - Fallback to `RemoteAddr`.
   - Document assumptions about running behind a proxy or load balancer.

2. **Graceful degradation**
   - Define behavior if the rate limiter encounters an internal error (e.g., map corruption should not be possible, but define a fallback policy).
   - Decide whether to default to “fail open” (allow) or “fail closed” (block) and document it; for the hackathon MVP, likely “fail open” to avoid demo disruption.

3. **Future multi‑instance scenario (non‑MVP)**
   - Add comments and TODOs describing how the current design would be adapted if the backend scales to multiple instances:
     - Shared Redis‑backed limiter.
     - or per‑instance limits with higher thresholds.
   - Clearly mark this as out of scope for the current hackathon implementation.

---

## Acceptance Criteria

- `POST /api/generate`:
  - More than 5 requests from the same client within 5 minutes receive HTTP 429 with a JSON error and optional `Retry-After` header.
- `GET /api/search`:
  - More than 100 requests from the same client within 5 minutes receive HTTP 429 with a JSON error and optional `Retry-After` header.
- Rate limits are enforced independently per client and per endpoint category.
- The limiter is unit‑tested, integrated into the existing Go backend, and does not introduce deadlocks or panics under concurrent load.
- The implementation remains in‑memory and does not require additional infrastructure beyond the current Go, SQLite, and Gradient AI setup. 
