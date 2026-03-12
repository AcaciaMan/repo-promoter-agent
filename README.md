# repo-promoter-agent

AI-powered GitHub repository promoter: paste a repo URL and get searchable, AI-generated summaries and promotional content (tweets, LinkedIn posts, tags) stored with full‑text search for easy discovery.

## Features

- **AI-generated promotional content** — Generates headlines, summaries, key benefits, tags, Twitter posts, LinkedIn posts, and calls-to-action for any public GitHub repo
- **GitHub repo metadata extraction** — Automatically fetches repo name, description, language, topics, stars, forks, and README from the GitHub API
- **Full-text search** — SQLite FTS5-powered search across all generated promotions
- **Single-page web UI** — Generate promotions and browse/search past results from a polished HTML frontend
- **Persistent storage** — All generated content is saved to a local SQLite database
- **Single binary** — Static assets embedded into the Go binary via `go:embed`

## Architecture

```
┌─────────────┐     ┌──────────────┐     ┌───────────────┐
│  Browser UI  │────▶│  HTTP Server │────▶│  AI Agent API │
│ (index.html) │◀────│   (Go)       │◀────│  (Gradient)   │
└─────────────┘     └──────┬───────┘     └───────────────┘
                           │
                    ┌──────┴───────┐
                    │  GitHub API  │
                    │ (repo data)  │
                    └──────┬───────┘
                           │
                    ┌──────┴───────┐
                    │   SQLite DB  │
                    │ (FTS5 search)│
                    └──────────────┘
```

## API Endpoints

| Method | Path             | Description                                      |
|--------|------------------|--------------------------------------------------|
| POST   | `/api/generate`  | Generate promotional content for a GitHub repo    |
| GET    | `/api/search`    | Full-text search or list recent promotions        |
| GET    | `/`              | Serves the single-page frontend                   |

### POST /api/generate

**Request body:**
```json
{
  "repo_url": "https://github.com/owner/repo",
  "target_channel": "twitter|linkedin|general",
  "target_audience": "optional audience description"
}
```

**Response:** A `Promotion` object containing `id`, `headline`, `summary`, `key_benefits`, `tags`, `twitter_posts`, `linkedin_post`, `call_to_action`, and `created_at`.

### GET /api/search

**Query parameters:**
- `q` — Search query (optional; if empty, returns recent promotions)
- `limit` — Max results (default: 20, max: 100)

## Generated Content

The AI agent produces structured JSON with:

| Field            | Description                                     |
|------------------|-------------------------------------------------|
| `headline`       | Attention-grabbing title                        |
| `summary`        | Concise repo description                        |
| `key_benefits`   | 3–5 bullet points                               |
| `tags`           | 5–8 discoverable tags                           |
| `twitter_posts`  | 3 tweets (≤280 chars each)                      |
| `linkedin_post`  | Professional post (150–300 words)               |
| `call_to_action` | Closing CTA string                              |

## Project Structure

```
cmd/server/main.go          — Entry point, HTTP server setup
internal/agent/client.go    — AI agent API client (prompt template, response parsing)
internal/github/client.go   — GitHub API client (repo metadata + README fetching)
internal/handler/generate.go — POST /api/generate handler
internal/handler/search.go  — GET /api/search handler
internal/store/store.go     — SQLite storage with FTS5 full-text search
static/index.html           — Single-page frontend UI
static/embed.go             — go:embed for static assets
```

## Getting Started

### Prerequisites

- Go 1.25+
- An AI agent endpoint (e.g. Gradient) with an API key

### Environment Variables

| Variable           | Required | Default          | Description                        |
|--------------------|----------|------------------|------------------------------------|
| `AGENT_ENDPOINT`   | Yes      | —                | URL of the AI agent service        |
| `AGENT_ACCESS_KEY` | Yes      | —                | API key for agent authentication   |
| `PORT`             | No       | `8080`           | HTTP server port                   |
| `DB_PATH`          | No       | `promotions.db`  | SQLite database file path          |

### Run

```bash
# Set required env vars (or create a .env file)
export AGENT_ENDPOINT=https://your-agent-endpoint
export AGENT_ACCESS_KEY=your-api-key

# Start the server
go run cmd/server/main.go
```

Then open http://localhost:8080 in your browser.

## Tech Stack

- **Go** — HTTP server, agent client, GitHub client
- **SQLite (modernc.org/sqlite)** — Pure-Go SQLite driver, no CGO required
- **FTS5** — Full-text search virtual table with auto-sync triggers
- **go:embed** — Static asset embedding
- **godotenv** — `.env` file loading
