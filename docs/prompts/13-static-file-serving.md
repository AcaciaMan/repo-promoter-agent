# Prompt: Review and Harden Static File Serving

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. The project is a Go backend + HTML frontend that generates promotional content for GitHub repos using a DigitalOcean Gradient AI agent.

The project is functionally complete with:
- `POST /api/generate` — generates promo content (with real GitHub data or hardcoded fallback)
- `GET /api/search` — full-text search over stored promotions
- `GET /` — serves `static/index.html` via `http.FileServer`
- SQLite + FTS5 for persistence

## Current static file serving code

In `cmd/server/main.go`:

```go
mux.Handle("/", http.FileServer(http.Dir("static")))
```

The `static/` directory contains only `index.html`.

## Your task

Review and improve the static file serving setup. This is going to be deployed to DigitalOcean App Platform eventually, so it needs to be robust.

## Issues to evaluate and fix

### 1. Working directory dependency

`http.Dir("static")` is relative — it only works if the binary is run from the project root. If run from `cmd/server/` or from a deployment container, it breaks.

**Options to evaluate**:
- (a) Use `go:embed` to embed the `static/` directory into the binary. No filesystem dependency at all. This is the cleanest for deployment.
- (b) Make the static dir configurable via env var (e.g., `STATIC_DIR`).
- (c) Keep relative path but document that the binary must be run from project root.

Recommend one approach. If `go:embed` is chosen, implement it.

### 2. API route priority

With the current `http.DefaultServeMux` behavior, `mux.Handle("/", ...)` is a catch-all. Verify that:
- `/api/generate` and `/api/search` are matched **before** the `"/"` catch-all.
- Requests to `/api/nonexistent` don't fall through to static file serving (returning `index.html`).

If there's a problem, fix it.

### 3. Cache headers

For a hackathon this isn't critical, but if simple: should the HTML response include `Cache-Control: no-cache` so the browser always gets the latest version during development?

### 4. MIME types

Verify that `http.FileServer` sets `Content-Type: text/html` correctly for `index.html`. If the project later adds CSS/JS files, will it work?

### 5. 404 for missing static files

Currently, requesting `/nonexistent.html` returns a 404 from `http.FileServer`. Is this the desired behavior, or should it serve `index.html` as an SPA fallback? For this project, plain 404 is fine — just confirm.

## Deliverables

1. **Recommendation** on which approach (embed vs. env var vs. relative) with reasoning.
2. **Updated `cmd/server/main.go`** — full file with the chosen approach implemented.
3. **Updated `static/` embedding** — if using `go:embed`, show the embed directive and any file changes needed.
4. **Brief notes** on each of the 5 issues above with the resolution.

## Constraints

- Keep it simple — don't over-engineer for a hackathon.
- The solution must work for both `go run cmd/server/main.go` (local dev) and a compiled binary in a container (deployment).
- Don't restructure other files. Only `main.go` and possibly a new small file for embedding.
- `go build ./...` must succeed after changes.
