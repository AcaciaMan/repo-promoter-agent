# Prompt: Create the Go Project Structure and Main Entry Point

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. This is **Phase 1** — a local Go HTTP service that calls a Gradient AI agent with hardcoded repo data and returns promotional content to a browser.

The project already has:
- `go.mod` with module name `repo-promoter-agent` (Go 1.25.3)
- `.env` and `.env.example` with `AGENT_ENDPOINT`, `AGENT_ACCESS_KEY`, `PORT`
- docs in `docs/`

No Go source code exists yet.

## Finalized contracts (from previous sessions)

> **IMPORTANT**: Before running this prompt, paste the actual finalized artifacts below. Use the placeholder structure for now.

### Environment variables

```
AGENT_ENDPOINT=https://xxxxx.agents.do-ai.run   # base URL of the Gradient agent
AGENT_ACCESS_KEY=xxxxxxxxxxxxx                    # Bearer token for the agent
PORT=8080                                         # HTTP server listen port (default: 8080)
```

### `/api/generate` endpoint

- **Request**: `POST /api/generate` — body TBD from prompt 04 decisions (at minimum `{"repo_url": "..."}` or no body)
- **Response**: `200 OK` with promotional JSON, or error JSON

## Your task

Create the **Go project structure** and the **`main.go` entry point** that wires everything together. Do NOT implement the agent client or handler logic — those come in prompts 06 and 07. This prompt focuses on the skeleton: file layout, dependency loading, server startup.

## Requirements

### 1. Project layout

Propose a minimal but clean file/package layout. The project is small — don't over-engineer. Suggested starting point:

```
cmd/server/main.go          # entry point: load env, wire dependencies, start server
internal/agent/client.go    # agent HTTP client (prompt 06)
internal/handler/generate.go # /api/generate handler (prompt 07)
static/index.html           # test HTML page (prompt 08)
```

If this is too much structure for a hackathon, simplify it — but explain your reasoning.

### 2. `main.go` responsibilities

The entry point should:

1. Load `.env` using `godotenv` (fail gracefully if `.env` doesn't exist — env vars might be set directly).
2. Read required env vars (`AGENT_ENDPOINT`, `AGENT_ACCESS_KEY`) and **fail fast with a clear error** if either is missing.
3. Read optional env vars (`PORT`, default `8080`).
4. Create the agent client (passing endpoint + key) — for now, define the constructor signature but the implementation is a stub/TODO.
5. Set up HTTP routes:
   - `POST /api/generate` → handler (stub for now)
   - `GET /` → serve static files from `static/` directory
6. Start the HTTP server and log the address.
7. Use only the standard library for HTTP (no Gin, Echo, etc.).

### 3. Dependencies

- `github.com/joho/godotenv` for `.env` loading
- No other external dependencies

## Deliverables

1. **File layout** — list of files with a one-line description of each file's purpose.
2. **`cmd/server/main.go`** — full, working Go code. Handler and agent client can be stubs/TODOs, but the server must compile and start.
3. **Stub files** — minimal `internal/agent/client.go` and `internal/handler/generate.go` with just enough types and function signatures that `main.go` compiles. Mark unimplemented bodies with `// TODO: implement in prompt 06/07`.
4. **`go get` command** — the exact command to run to install `godotenv`.

## Constraints

- Standard library only for HTTP serving (no frameworks).
- The server must compile and run after this prompt — I should be able to `go run cmd/server/main.go`, see it listening, and hit `GET /` (even if `/api/generate` returns a TODO response).
- Keep the code idiomatic Go: proper error handling, clear naming, minimal comments.
- No tests in this prompt (they come later).
