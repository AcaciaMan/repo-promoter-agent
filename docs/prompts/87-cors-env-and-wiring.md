# Prompt: CORS — Environment Config and Wiring into main.go

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. I'm adding a configurable CORS middleware (Phase 1).

- Prompt 85 created `internal/cors/cors.go` with `Config` type and `Middleware` function.
- Prompt 86 added comprehensive unit tests in `internal/cors/cors_test.go`.
- This prompt (87) wires the CORS middleware into `cmd/server/main.go` via a new `CORS_ALLOWED_ORIGINS` environment variable.

## Existing state

### `internal/cors/cors.go` (created in prompt 85)

```go
package cors

type Config struct {
    AllowedOrigins []string
    AllowedMethods []string
    AllowedHeaders []string
    MaxAge         int
}

// Middleware returns HTTP middleware that enforces CORS based on Config.
// If AllowedOrigins is empty, it's a no-op passthrough.
func Middleware(cfg Config) func(http.Handler) http.Handler
```

### `cmd/server/main.go` — current env var pattern

```go
// Required env vars — fail fast if missing.
endpoint := mustEnv("AGENT_ENDPOINT")
accessKey := mustEnv("AGENT_ACCESS_KEY")

// Optional env vars.
port := os.Getenv("PORT")
if port == "" {
    port = "8080"
}
```

Helper functions available: `mustEnv(key)`, `envIntOr(key, default)`.

### `cmd/server/main.go` — current server startup

```go
// Set up routes.
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

### Rate limiter interaction

The rate limiter middleware (in `internal/ratelimit/ratelimit.go`) already skips rate limiting for `OPTIONS` requests:

```go
if r.Method == http.MethodOptions {
    next.ServeHTTP(w, r)
    return
}
```

With CORS wrapping the mux as the **outermost middleware**, `OPTIONS` preflight requests will be handled by the CORS middleware (returning `204`) before reaching the rate limiter. This is the desired behavior. No changes are needed to the rate limiter.

## Your task

Modify `cmd/server/main.go` to:
1. Parse the new `CORS_ALLOWED_ORIGINS` environment variable.
2. Construct a `cors.Config`.
3. Wrap the mux with the CORS middleware.

## Requirements

### 1. Environment variable: `CORS_ALLOWED_ORIGINS`

- **Format**: comma-separated list of origins (e.g., `https://repo-promoter.ondigitalocean.app,http://localhost:8080`).
- **Optional**: if empty or unset, CORS middleware is a no-op passthrough (backward compatible).
- **Parsing**: split by `,` and trim whitespace from each entry. Skip empty entries (handles trailing commas gracefully).

### 2. Config construction

After parsing `CORS_ALLOWED_ORIGINS`, build a `cors.Config`:

```go
corsCfg := cors.Config{
    AllowedOrigins: allowedOrigins, // parsed from env var
    AllowedMethods: []string{"GET", "POST", "OPTIONS"},
    AllowedHeaders: []string{"Content-Type"},
    MaxAge:         86400, // 24 hours preflight cache
}
```

Use sensible defaults for methods, headers, and max age. These cover the app's current needs:
- `GET` — search, suggest, mlt, analytics endpoints.
- `POST` — generate endpoint.
- `OPTIONS` — preflight requests.
- `Content-Type` — the only custom header the frontend sends (`application/json`).

### 3. Middleware wiring

Apply the CORS middleware as the **outermost wrapper** around the mux, so it runs before rate limiting:

```go
var rootHandler http.Handler = mux
rootHandler = cors.Middleware(corsCfg)(rootHandler)

if err := http.ListenAndServe(addr, rootHandler); err != nil {
```

### 4. Startup logging

Log the configured origins at startup, following the existing logging style:

- If origins are configured: `log.Printf("CORS enabled for origins: %v", allowedOrigins)`
- If no origins configured: `log.Println("CORS not configured — same-origin only (set CORS_ALLOWED_ORIGINS to enable)")`

### 5. Import

Add `"repo-promoter-agent/internal/cors"` to the import block. Also add `"strings"` if not already imported (needed for `strings.Split` and `strings.TrimSpace`).

## What NOT to change

- Do **not** modify any file in `internal/cors/` — that's already done.
- Do **not** modify `internal/ratelimit/` — the OPTIONS passthrough is already compatible.
- Do **not** change the route registrations — the CORS middleware wraps the entire mux.
- Do **not** add `CORS_ALLOWED_ORIGINS` to `.env` — it should remain empty/unset for local development.

## Verification

- `go build ./...` compiles with no errors.
- Start server **without** `CORS_ALLOWED_ORIGINS` → log shows "CORS not configured — same-origin only".
- Start server **with** `CORS_ALLOWED_ORIGINS=http://localhost:3000` → log shows "CORS enabled for origins: [http://localhost:3000]".
- No changes to files outside `cmd/server/main.go`.
