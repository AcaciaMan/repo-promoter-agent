# Prompt: Add Repo Owner Detection and GitHub Token Support

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. The app generates promotional content for GitHub repos using a Gradient AI agent, stores it in SQLite with FTS, and serves it via a Go HTTP server with a simple HTML frontend.

I'm now starting a new feature: **GitHub traffic metrics (views & clones)** for repositories owned by `AcaciaMan`. This is **Phase 1, Step 1** of the plan described in `docs/intent-for-views-clones.md`.

The full intent document is at `docs/intent-for-views-clones.md` — read it for high-level context.

## Current project state

```
cmd/server/main.go              # entry point, env vars, routing
internal/agent/client.go        # Gradient AI agent client
internal/github/client.go       # GitHub API client (unauthenticated)
internal/handler/generate.go    # POST /api/generate handler
internal/handler/search.go      # GET /api/search handler
internal/store/store.go         # SQLite storage with FTS
static/index.html               # frontend
```

## Current GitHub client (`internal/github/client.go`)

The client currently has **no authentication support**. All requests are unauthenticated:

```go
type Client struct {
    httpClient *http.Client
}

func NewClient() *Client {
    return &Client{
        httpClient: &http.Client{Timeout: 10 * time.Second},
    }
}
```

Requests set only `User-Agent` and `Accept` headers:

```go
req.Header.Set("User-Agent", "repo-promoter-agent")
req.Header.Set("Accept", "application/vnd.github+json")
```

A `parseGitHubURL(rawURL string) (owner, repo string, err error)` function already exists — it extracts `owner` and `repo` from GitHub URLs.

## Current `cmd/server/main.go` env handling

```go
endpoint := mustEnv("AGENT_ENDPOINT")
accessKey := mustEnv("AGENT_ACCESS_KEY")
port := os.Getenv("PORT")       // optional, default "8080"
dbPath := os.Getenv("DB_PATH")  // optional, default "promotions.db"
githubClient := github.NewClient()
```

## Your task

Make two changes:

### 1. Add GitHub token support to the GitHub client

Modify `internal/github/client.go`:

- Add a `token` field to the `Client` struct (string, may be empty).
- Change `NewClient()` to accept an optional token: `NewClient(token string) *Client`.
- When `token` is non-empty, add the `Authorization: Bearer <token>` header to **all** GitHub API requests (not just traffic endpoints). This improves rate limits from 60/hr to 5000/hr for all calls, which is beneficial regardless of the traffic feature.
- When `token` is empty, continue to work as today (unauthenticated).
- To avoid duplicating the header-setting logic, extract a small helper method:

```go
func (c *Client) newGitHubRequest(ctx context.Context, method, url string) (*http.Request, error)
```

This method creates the request, sets `User-Agent`, `Accept`, and conditionally `Authorization`. Then use it in `fetchRepoMeta` and `fetchReadme` instead of manual header setting.

### 2. Add an exported helper to check owner

Add an exported function (it can be a standalone function, not a method):

```go
// RepoOwner extracts the owner from a GitHub URL. Returns empty string on error.
func RepoOwner(repoURL string) string
```

This wraps the existing `parseGitHubURL` and returns just the owner (or empty string on parse error). This will be used by the generate handler to gate traffic metric fetching.

### 3. Wire the token in `cmd/server/main.go`

- Read `GITHUB_TOKEN` from environment (optional — use `os.Getenv`, not `mustEnv`).
- If set, log: `"GitHub token configured — authenticated API access enabled"`.
- If not set, log: `"No GITHUB_TOKEN set — using unauthenticated GitHub API (60 req/hr limit)"`.
- Pass the token to `github.NewClient(token)`.

## What NOT to do

- Do NOT implement traffic API calls yet (that's the next prompt).
- Do NOT modify the store, handler, agent, or frontend.
- Do NOT change `FetchRepo` return type or behavior — only the HTTP request construction changes.
- Do NOT break any existing behavior when token is empty.

## Verification

After implementation, the following should work:

1. `go build ./...` compiles without errors.
2. With no `GITHUB_TOKEN` set, the app starts and works exactly as before.
3. With `GITHUB_TOKEN=ghp_test123` set, the log shows the "authenticated" message and all GitHub API requests include the `Authorization` header.
4. `github.RepoOwner("https://github.com/AcaciaMan/some-repo")` returns `"AcaciaMan"`.
5. `github.RepoOwner("https://github.com/other/repo")` returns `"other"`.
6. `github.RepoOwner("not-a-url")` returns `""`.
