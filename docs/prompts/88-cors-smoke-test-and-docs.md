# Prompt: CORS — Smoke Test and Documentation

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. I'm adding a configurable CORS middleware (Phase 1).

- Prompt 85 created `internal/cors/cors.go` — config type and middleware.
- Prompt 86 added unit tests in `internal/cors/cors_test.go`.
- Prompt 87 wired CORS into `cmd/server/main.go` via `CORS_ALLOWED_ORIGINS` env var.
- This prompt (88) verifies everything works end-to-end and documents the feature.

## Your task

Perform the following steps in order.

## Step 1: Run all tests

Run ALL tests to verify no regressions:

```bash
go test ./...
```

All tests must pass, including:
- `internal/cors/` — the new CORS tests from prompt 86.
- `internal/ratelimit/` — existing rate limiter tests (must not be broken).
- `internal/store/` — existing Solr store tests.
- `internal/agent/` — existing agent tests.

If any test fails, fix the underlying issue before proceeding.

## Step 2: Build verification

```bash
go build ./...
go vet ./...
```

Both must pass cleanly.

## Step 3: Manual curl smoke tests

Start the server with CORS enabled:

```bash
CORS_ALLOWED_ORIGINS=http://localhost:3000 go run ./cmd/server/main.go
```

(On Windows, set the env var using PowerShell `$env:CORS_ALLOWED_ORIGINS="http://localhost:3000"` before running.)

Verify the startup log line: `CORS enabled for origins: [http://localhost:3000]`

Then run these curl tests (in a separate terminal):

### Test A: Allowed origin on a GET endpoint

```bash
curl -i -H "Origin: http://localhost:3000" http://localhost:8080/api/search?q=test
```

Expected in response headers:
- `Access-Control-Allow-Origin: http://localhost:3000` ✅
- `Vary: Origin` ✅

### Test B: Disallowed origin

```bash
curl -i -H "Origin: http://evil.example.com" http://localhost:8080/api/search?q=test
```

Expected:
- **No** `Access-Control-Allow-Origin` header ✅
- Response body still returned (server doesn't block, browser does) ✅

### Test C: Preflight (OPTIONS) with allowed origin

```bash
curl -i -X OPTIONS \
  -H "Origin: http://localhost:3000" \
  -H "Access-Control-Request-Method: POST" \
  http://localhost:8080/api/generate
```

Expected:
- Status: `204 No Content` ✅
- `Access-Control-Allow-Origin: http://localhost:3000` ✅
- `Access-Control-Allow-Methods: GET, POST, OPTIONS` ✅
- `Access-Control-Allow-Headers: Content-Type` ✅
- `Access-Control-Max-Age: 86400` ✅

### Test D: Preflight with disallowed origin

```bash
curl -i -X OPTIONS \
  -H "Origin: http://evil.example.com" \
  -H "Access-Control-Request-Method: POST" \
  http://localhost:8080/api/generate
```

Expected:
- **No** CORS headers ✅
- Request passes through to handler (may return 405 or handler response) ✅

### Test E: No Origin header (same-origin / non-browser)

```bash
curl -i http://localhost:8080/api/search?q=test
```

Expected:
- **No** `Access-Control-Allow-Origin` header ✅
- Normal response ✅

### Test F: Default behavior (no env var)

Stop the server. Restart **without** `CORS_ALLOWED_ORIGINS`:

```bash
go run ./cmd/server/main.go
```

Verify startup log: `CORS not configured — same-origin only (set CORS_ALLOWED_ORIGINS to enable)`

```bash
curl -i -H "Origin: http://localhost:3000" http://localhost:8080/api/search?q=test
```

Expected:
- **No** CORS headers (middleware is no-op) ✅
- Normal response ✅

## Step 4: Update README.md

Add a new section for the `CORS_ALLOWED_ORIGINS` env var in the **Environment Variables** or **Configuration** section of `README.md`. Follow the existing format for other env vars. Include:

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `CORS_ALLOWED_ORIGINS` | No | _(empty — same-origin only)_ | Comma-separated list of allowed origins for cross-origin requests (e.g., `https://app.example.com,http://localhost:3000`). When empty, no CORS headers are sent (suitable for same-origin deployment). |

Also add a brief paragraph in the README (near rate limiter docs or in a new **CORS** subsection) explaining:
- Purpose: restrict cross-origin API access to specific frontends.
- Default: no CORS headers (same-origin behavior, backward compatible).
- Example: `CORS_ALLOWED_ORIGINS=https://repo-promoter.ondigitalocean.app,http://localhost:3000`
- Note: Preflight (`OPTIONS`) responses are cached for 24 hours via `Access-Control-Max-Age`.

## Step 5: Final verification

Run tests one more time after any README changes to confirm nothing is broken:

```bash
go test ./...
go vet ./...
```

## Summary of expected outcomes

After this prompt completes:
1. ✅ `go test ./...` passes — all tests green.
2. ✅ `go build ./...` and `go vet ./...` clean.
3. ✅ CORS works correctly with `CORS_ALLOWED_ORIGINS` set.
4. ✅ CORS is a no-op when `CORS_ALLOWED_ORIGINS` is unset (backward compatible).
5. ✅ README documents the new env var.
6. ✅ No unnecessary files created outside the scope of this feature.
