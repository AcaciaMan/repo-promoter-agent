# Prompt: Rate Limiter Polish — README Docs + Frontend 429 UX

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. Phase 1 of the rate limiter is complete (prompts 44–50). Phase 2 prompts 51–52 enhanced logging and added env-var config overrides. This is **Phase 2, prompt 3 of 3** — the final polish step.

The full intent is in `docs/intent-for-rate-limiter.md`.

## Current state

### Rate limiter behavior

- `POST /api/generate`: max requests per client per 5-minute rolling window (default 5, configurable via `RATE_LIMIT_GENERATE_MAX`).
- `GET /api/search`: max requests per client per 5-minute rolling window (default 100, configurable via `RATE_LIMIT_SEARCH_MAX`).
- On limit exceeded: HTTP 429 with `Retry-After` header and JSON body:
  ```json
  {"error": "rate limit exceeded", "retry_after_seconds": 42}
  ```

### README.md — no rate limiter docs yet

The README documents endpoints, env vars, traffic metrics, and repo analysis. The rate limiter is not mentioned. The env vars table at line ~110 lists `AGENT_ENDPOINT`, `AGENT_ACCESS_KEY`, `PORT`, `DB_PATH`, `GITHUB_TOKEN`, `ANALYSIS_AGENT_ENDPOINT`, `ANALYSIS_AGENT_ACCESS_KEY`.

### Frontend — no 429-specific handling

In `static/index.html`, the generate handler does:

```javascript
if(!res.ok){const e=await res.json().catch(()=>({}));throw new Error(e.error||res.statusText)}
```

This shows the generic error message "rate limit exceeded" but doesn't tell the user **how long to wait**. The search handler has the same pattern.

## Your task

Two changes: (A) document the rate limiter in the README, (B) improve frontend 429 handling.

## Requirements

### Part A — README documentation

#### 1. Add rate limiter section

Add a new section after the "Repo Analysis" section, titled `## Rate Limiting`. Include:

- One-sentence explanation that the server rate-limits API requests per client IP.
- Table showing both endpoints, their limits, and the window:

| Endpoint | Default Limit | Window | Override Env Var |
|----------|--------------|--------|-----------------|
| `POST /api/generate` | 5 requests | 5 minutes | `RATE_LIMIT_GENERATE_MAX` |
| `GET /api/search` | 100 requests | 5 minutes | `RATE_LIMIT_SEARCH_MAX` |

- Brief note on the 429 response format:
  - `Retry-After` header (seconds).
  - JSON body with `error` and `retry_after_seconds`.
- One sentence noting that behind a reverse proxy (like DigitalOcean App Platform), the limiter uses `X-Forwarded-For` for client identification.

#### 2. Update the env vars table

Add two rows to the existing Environment Variables table:

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `RATE_LIMIT_GENERATE_MAX` | No | `5` | Max generate requests per client per 5-min window |
| `RATE_LIMIT_SEARCH_MAX` | No | `100` | Max search requests per client per 5-min window |

#### 3. Update the Project Structure listing

Add the `internal/ratelimit/` files to the project structure block:

```
internal/ratelimit/ratelimit.go  — In-memory per-client rate limiter
internal/ratelimit/clientkey.go  — Client IP extraction (X-Forwarded-For aware)
```

### Part B — Frontend 429 UX

#### 1. Detect 429 in the generate handler

In the JavaScript `generate` fetch handler, before the generic `!res.ok` check, add a specific check for status 429:

```javascript
if(res.status===429){
    const e=await res.json().catch(()=>({}));
    const secs=e.retry_after_seconds||60;
    throw new Error('Rate limit reached. Please wait '+secs+' seconds before trying again.');
}
```

This gives the user an actionable message with the wait time.

#### 2. Detect 429 in the search handler

Same pattern for the search fetch:

```javascript
if(res.status===429){
    const e=await res.json().catch(()=>({}));
    const secs=e.retry_after_seconds||60;
    throw new Error('Search rate limit reached. Please wait '+secs+' seconds.');
}
```

#### 3. No visual redesign

Don't change the error display CSS or add countdown timers. Just improve the error message text. Keep it simple — the existing `.error` class styling is fine.

## Verification

- `go build ./...` compiles cleanly.
- `go test ./...` — all tests pass (no logic changes).
- README includes the Rate Limiting section with correct env var names and limits.
- README env vars table includes `RATE_LIMIT_GENERATE_MAX` and `RATE_LIMIT_SEARCH_MAX`.
- README project structure includes `internal/ratelimit/` files.
- In the browser, triggering a 429 on generate shows: "Rate limit reached. Please wait N seconds before trying again."
- In the browser, triggering a 429 on search shows: "Search rate limit reached. Please wait N seconds."
