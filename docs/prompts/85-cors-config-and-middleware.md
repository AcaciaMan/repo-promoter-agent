# Prompt: CORS — Config Type and Middleware Implementation

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. All previous phases (prompts 01–84) are complete, including an in-memory rate limiter wired into `cmd/server/main.go`.

The project is heading toward **DigitalOcean App Platform deployment**. Currently the frontend is served same-origin (embedded static files via `embed.FS`), so no CORS headers are needed. However, once deployed, the frontend may be served from a separate origin (CDN, preview deployments, etc.). I'm adding a **configurable CORS middleware** so only explicitly allowed origins can call the `/api/*` endpoints.

This prompt (85) creates the **CORS config type and middleware implementation**. Subsequent prompts will add:
- 86: comprehensive unit tests
- 87: env var configuration and wiring into `cmd/server/main.go`
- 88: smoke test and verification

## Existing project state

```
cmd/server/main.go              # entry point — routes, env vars, middleware wiring
internal/ratelimit/ratelimit.go  # rate limiter with Middleware() pattern to follow
internal/handler/                # HTTP handlers (generate, search, suggest, mlt, popular)
static/embed.go                  # embedded frontend (same-origin, no CORS needed currently)
```

### Middleware pattern to follow

The rate limiter in `internal/ratelimit/ratelimit.go` uses a higher-order function pattern:

```go
func (l *Limiter) Middleware(bucket string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // ... logic ...
            next.ServeHTTP(w, r)
        })
    }
}
```

The CORS middleware does **not** need to be method-receiver-based since it wraps the entire mux. Use a simpler pattern: a function that takes `Config` and returns `func(http.Handler) http.Handler`.

### Current route setup in `cmd/server/main.go`

```go
mux := http.NewServeMux()
mux.Handle("/api/generate", limiter.Middleware("generate")(handler.NewGenerateHandler(...)))
mux.Handle("/api/search", limiter.Middleware("search")(handler.NewSearchHandler(...)))
mux.Handle("/api/suggest", limiter.Middleware("search")(handler.NewSuggestHandler(...)))
mux.Handle("/api/mlt", limiter.Middleware("search")(handler.NewMLTHandler(...)))
mux.Handle("/api/analytics/popular", limiter.Middleware("search")(handler.NewPopularHandler(...)))
mux.Handle("/", noCacheHandler(http.FileServerFS(static.Files)))

addr := ":" + port
log.Printf("Server listening on http://localhost%s", addr)
if err := http.ListenAndServe(addr, mux); err != nil {
    log.Fatalf("Server failed: %v", err)
}
```

The CORS middleware will wrap the entire `mux`, so in a later prompt it will be wired as:
```go
http.ListenAndServe(addr, corsMiddleware(mux))
```

## Your task

Create a new package `internal/cors/` with a single file `internal/cors/cors.go` containing the CORS config type and middleware function.

## Requirements

### 1. `Config` struct

```go
type Config struct {
    AllowedOrigins []string // explicit list of allowed origins (e.g., "https://example.com")
    AllowedMethods []string // HTTP methods to allow (e.g., "GET", "POST", "OPTIONS")
    AllowedHeaders []string // request headers to allow (e.g., "Content-Type")
    MaxAge         int      // preflight cache duration in seconds (Access-Control-Max-Age)
}
```

### 2. `Middleware` function

```go
func Middleware(cfg Config) func(http.Handler) http.Handler
```

Returns a standard middleware that wraps an `http.Handler`.

### 3. Origin matching logic

- Read the `Origin` header from every request.
- If `Origin` is empty (same-origin browser request or non-browser client), **pass through unchanged** — do not set any CORS headers.
- If `Origin` matches one of the `AllowedOrigins` entries (exact string match, case-sensitive), set `Access-Control-Allow-Origin` to that **specific origin** (never use `*`).
- If `Origin` is present but does **not** match any allowed origin, do **not** set any CORS headers — the browser will block the response. Do **not** return an error status code (the request itself may still execute; it's the browser that enforces the block).
- Always set `Vary: Origin` when the middleware is active (allowed origins list is non-empty) to ensure caches don't serve a response with the wrong origin header.

### 4. Preflight handling (`OPTIONS` requests)

When the request is `OPTIONS` **and** has a non-empty `Origin` that matches an allowed origin:

1. Set `Access-Control-Allow-Origin: <matched-origin>`
2. Set `Access-Control-Allow-Methods: <cfg.AllowedMethods joined by ", ">`
3. Set `Access-Control-Allow-Headers: <cfg.AllowedHeaders joined by ", ">`
4. Set `Access-Control-Max-Age: <cfg.MaxAge>` (only if `MaxAge > 0`)
5. Set `Vary: Origin`
6. Write `204 No Content` — do **not** call `next.ServeHTTP`. The preflight is fully handled.

When `OPTIONS` but origin doesn't match or is empty: pass through to `next`.

### 5. Non-preflight requests

When not `OPTIONS` and origin matches:

1. Set `Access-Control-Allow-Origin: <matched-origin>`
2. Set `Vary: Origin`
3. Call `next.ServeHTTP(w, r)`.

### 6. Empty config passthrough

If `cfg.AllowedOrigins` is empty (length 0), the middleware should be a **no-op passthrough**: just call `next.ServeHTTP(w, r)` without touching headers. This preserves backward compatibility when no `CORS_ALLOWED_ORIGINS` env var is set.

### 7. Helper: `originAllowed`

Create an unexported helper to check origin membership:

```go
func originAllowed(origin string, allowed []string) bool
```

Iterate through `allowed` and return `true` on exact match. Simple and clear.

## Design decisions

- **No wildcard `*` support**: Always reflect the specific allowed origin. This is more secure and required when credentials are involved (future-proofing).
- **No `Access-Control-Allow-Credentials`**: The app doesn't use cookies or auth headers from the browser yet. Can be added later.
- **No external dependencies**: Use only standard library (`net/http`, `strings`, `strconv`).
- **No regex or suffix matching**: Only exact origins. Wildcard subdomain matching can be added later if preview deployments need it.

## Verification

- `go build ./internal/cors/` compiles with no errors.
- `go vet ./internal/cors/` reports no issues.
- The package imports only standard library packages.
- No changes to any file outside `internal/cors/`.
