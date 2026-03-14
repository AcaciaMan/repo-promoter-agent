# Prompt: CORS — Unit Tests

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. I'm adding a configurable CORS middleware (Phase 1). 

Prompt 85 created the CORS middleware in `internal/cors/cors.go`. This prompt (86) adds comprehensive **unit tests** in `internal/cors/cors_test.go`.

## Existing state

### `internal/cors/cors.go` (created in prompt 85)

Contains:

```go
// Config holds CORS settings.
type Config struct {
    AllowedOrigins []string
    AllowedMethods []string
    AllowedHeaders []string
    MaxAge         int
}

// Middleware returns HTTP middleware that enforces CORS based on Config.
// If AllowedOrigins is empty, it's a no-op passthrough.
func Middleware(cfg Config) func(http.Handler) http.Handler

// originAllowed checks if origin is in the allowed list (exact match).
func originAllowed(origin string, allowed []string) bool
```

### Middleware behavior summary (from prompt 85)

1. **Empty config** → no-op passthrough, no headers set.
2. **No `Origin` header** → passthrough, no CORS headers.
3. **Origin matches allowed** → set `Access-Control-Allow-Origin: <origin>` + `Vary: Origin`, call next handler.
4. **Origin doesn't match** → no CORS headers set, still call next handler (browser blocks the response).
5. **`OPTIONS` + matching origin** → set all preflight headers (`Allow-Origin`, `Allow-Methods`, `Allow-Headers`, `Max-Age`), return `204 No Content`, do NOT call next handler.
6. **`OPTIONS` + non-matching origin** → pass through to next handler unchanged.

### Test style reference

Follow the pattern from `internal/ratelimit/ratelimit_test.go` — table-driven tests where appropriate, `httptest.NewRecorder()` for capturing responses.

## Your task

Create `internal/cors/cors_test.go` with thorough unit tests.

## Required test cases

### 1. `TestMiddleware_EmptyConfig` — no-op passthrough

- Create middleware with `Config{}` (empty `AllowedOrigins`).
- Send a request with `Origin: http://example.com`.
- Verify:
  - Response status is `200` (from dummy next handler).
  - `Access-Control-Allow-Origin` header is **not present**.
  - Next handler **was called** (use a flag/counter).

### 2. `TestMiddleware_NoOriginHeader` — same-origin request

- Create middleware with `Config{AllowedOrigins: []string{"http://localhost:3000"}}`.
- Send a request with **no `Origin` header**.
- Verify:
  - Response status is `200`.
  - `Access-Control-Allow-Origin` header is **not present**.
  - Next handler **was called**.

### 3. `TestMiddleware_AllowedOrigin` — matching origin

- Config: `AllowedOrigins: []string{"http://localhost:3000"}`.
- Send a `GET` request with `Origin: http://localhost:3000`.
- Verify:
  - `Access-Control-Allow-Origin` is `http://localhost:3000`.
  - `Vary` header contains `Origin`.
  - Next handler **was called**.

### 4. `TestMiddleware_DisallowedOrigin` — non-matching origin

- Config: `AllowedOrigins: []string{"http://localhost:3000"}`.
- Send a `GET` request with `Origin: http://evil.example.com`.
- Verify:
  - `Access-Control-Allow-Origin` header is **not present**.
  - `Vary` header contains `Origin` (always set when origins are configured).
  - Next handler **was called** (CORS doesn't block server-side).

### 5. `TestMiddleware_PreflightAllowed` — OPTIONS with matching origin

- Config:
  ```go
  Config{
      AllowedOrigins: []string{"http://localhost:3000"},
      AllowedMethods: []string{"GET", "POST", "OPTIONS"},
      AllowedHeaders: []string{"Content-Type"},
      MaxAge:         3600,
  }
  ```
- Send `OPTIONS` request with:
  - `Origin: http://localhost:3000`
  - `Access-Control-Request-Method: POST`
- Verify:
  - Response status is `204 No Content`.
  - `Access-Control-Allow-Origin` is `http://localhost:3000`.
  - `Access-Control-Allow-Methods` is `GET, POST, OPTIONS`.
  - `Access-Control-Allow-Headers` is `Content-Type`.
  - `Access-Control-Max-Age` is `3600`.
  - `Vary` header contains `Origin`.
  - Next handler was **NOT called**.

### 6. `TestMiddleware_PreflightDisallowed` — OPTIONS with non-matching origin

- Config: `AllowedOrigins: []string{"http://localhost:3000"}`.
- Send `OPTIONS` request with `Origin: http://evil.example.com`.
- Verify:
  - No `Access-Control-Allow-Origin` header.
  - Next handler **was called** (passes through).

### 7. `TestMiddleware_MultipleOrigins` — multiple allowed origins

- Config: `AllowedOrigins: []string{"http://localhost:3000", "https://app.example.com"}`.
- Send request with `Origin: https://app.example.com`.
- Verify `Access-Control-Allow-Origin` is `https://app.example.com` (reflects the matched one, not the other).
- Send another request with `Origin: http://localhost:3000`.
- Verify `Access-Control-Allow-Origin` is `http://localhost:3000`.
- Send request with `Origin: http://other.example.com`.
- Verify no `Access-Control-Allow-Origin` header.

### 8. `TestMiddleware_MaxAgeZero` — no Max-Age when 0

- Config with `MaxAge: 0`.
- Send preflight `OPTIONS` with matching origin.
- Verify `Access-Control-Max-Age` header is **not present**.

### 9. `TestOriginAllowed` — unit test for helper

- Test the unexported `originAllowed` function directly:
  - `originAllowed("http://localhost:3000", []string{"http://localhost:3000"})` → `true`
  - `originAllowed("http://evil.com", []string{"http://localhost:3000"})` → `true` → wait, should be `false`
  - `originAllowed("http://evil.com", []string{"http://localhost:3000"})` → `false`
  - `originAllowed("", []string{"http://localhost:3000"})` → `false`
  - `originAllowed("http://localhost:3000", nil)` → `false`
  - `originAllowed("http://localhost:3000", []string{})` → `false`
  - Case-sensitive: `originAllowed("HTTP://LOCALHOST:3000", []string{"http://localhost:3000"})` → `false`

## Test helpers

Create a `dummyHandler` that sets a flag and writes `200 OK`:

```go
func dummyHandler() (http.Handler, *bool) {
    called := false
    h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        called = true
        w.WriteHeader(http.StatusOK)
    })
    return h, &called
}
```

Use `httptest.NewRecorder()` and `httptest.NewRequest()` throughout.

## Verification

- `go test ./internal/cors/` — all tests pass.
- `go vet ./internal/cors/` — no issues.
- No changes to any file outside `internal/cors/`.
