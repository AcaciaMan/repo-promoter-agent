# Prompt: Implement the GitHub API Client

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. Phase 1 is complete — I have a working Go service that calls a Gradient AI agent with **hardcoded** repo data and returns promotional content.

Now I'm starting **Phase 2**: replacing hardcoded data with real GitHub API calls.

## Current project state

```
cmd/server/main.go              # entry point
internal/agent/client.go        # Gradient agent HTTP client (done)
internal/handler/generate.go    # POST /api/generate handler (uses hardcoded data)
static/index.html               # test page
```

### Existing `RepoInput` type (in `internal/agent/client.go`)

```go
type RepoInput struct {
    RepoURL          string      `json:"repo_url"`
    RepoName         string      `json:"repo_name"`
    ShortDescription string      `json:"short_description"`
    ReadmeSummary    string      `json:"readme_summary"`
    PrimaryLanguage  string      `json:"primary_language,omitempty"`
    Topics           []string    `json:"topics,omitempty"`
    Metrics          RepoMetrics `json:"metrics"`
    TargetChannel    string      `json:"target_channel,omitempty"`
    TargetAudience   string      `json:"target_audience,omitempty"`
}

type RepoMetrics struct {
    Stars      int `json:"stars"`
    Forks      int `json:"forks"`
    Watchers   int `json:"watchers"`
    OpenIssues int `json:"open_issues"`
}
```

## Your task

Create `internal/github/client.go` — a client that fetches public repo metadata from the GitHub REST API and returns a populated `agent.RepoInput`.

## Requirements

### 1. Client struct

```go
type Client struct {
    httpClient *http.Client
}
func NewClient() *Client
```

- Use a reasonable timeout (10 seconds).
- No authentication for now — public repos only, subject to rate limits (60 req/hr unauthenticated). This is fine for a hackathon.

### 2. Core method: `FetchRepo`

```go
func (c *Client) FetchRepo(ctx context.Context, repoURL string) (agent.RepoInput, error)
```

This method should:

1. **Parse the repo URL** — extract `owner` and `repo` from URLs like:
   - `https://github.com/owner/repo`
   - `https://github.com/owner/repo/`
   - `github.com/owner/repo` (no scheme)
   - Return a clear error if the URL doesn't match.

2. **Fetch repo metadata** from `GET https://api.github.com/repos/{owner}/{repo}`:
   - Extract: `name`, `description`, `stargazers_count`, `forks_count`, `subscribers_count` (watchers), `open_issues_count`, `language`, `topics`.
   - Set `User-Agent` header (required by GitHub API).
   - Handle 404 (repo not found) and other errors with clear messages.

3. **Fetch README content** from `GET https://api.github.com/repos/{owner}/{repo}/readme`:
   - The response has a `content` field (base64-encoded) and `encoding` field.
   - Decode the base64 content.
   - Truncate to a reasonable length (e.g., first 2000 characters) to avoid huge prompts.
   - If README doesn't exist (404), use a fallback like "No README available."

4. **Build and return `agent.RepoInput`**:
   - `RepoURL` = the canonical GitHub URL
   - `RepoName` = from API `name` or `full_name`
   - `ShortDescription` = from API `description` (use empty string if null)
   - `ReadmeSummary` = truncated README text
   - `PrimaryLanguage` = from API `language`
   - `Topics` = from API `topics`
   - `Metrics` = mapped from API fields
   - `TargetChannel` and `TargetAudience` are NOT set here — the handler will fill those from the request.

### 3. URL parsing

Create a helper:

```go
func parseGitHubURL(rawURL string) (owner, repo string, err error)
```

Be lenient with input formats. Strip trailing slashes, handle missing scheme.

### 4. Error handling

- Return clear, user-facing error messages (these will end up in HTTP error responses).
- Distinguish between "repo not found" (404) and "GitHub API error" (5xx, rate limit, etc.).
- Do not panic.

### 5. GitHub API response types

Define only the fields you need from the GitHub API responses. Don't model the entire GitHub API.

## Deliverables

1. **`internal/github/client.go`** — full, working Go code.
2. **Import note** — confirm that `internal/agent` is imported for `agent.RepoInput` and `agent.RepoMetrics`.
3. **No changes to other files** — this prompt only adds a new file. Wiring into the handler happens in prompt 11.

## Constraints

- Standard library only (no `google/go-github` or other wrappers — too heavy for this).
- No GitHub authentication token for now (phase 2 can add it later if rate limits are hit).
- The `FetchRepo` method should make exactly **2 HTTP requests** (repo metadata + README). Don't fetch additional data.
- Handle the README being absent gracefully — it's not required.
- Truncate README to prevent excessively long prompts. Use byte length, not line count.
