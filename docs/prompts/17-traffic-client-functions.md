# Prompt: Implement GitHub Traffic API Client (Views & Clones)

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. I'm adding GitHub traffic metrics (views & clones) for AcaciaMan repos.

This is **Phase 1, Step 2**. The previous prompt (16) added:

- GitHub token support to the client (`Client` struct now has a `token` field, `NewClient(token string)`).
- A `newGitHubRequest` helper that sets `User-Agent`, `Accept`, and conditionally `Authorization: Bearer <token>`.
- An exported `RepoOwner(repoURL string) string` function.

The full intent document is at `docs/intent-for-views-clones.md`.

## Current GitHub client (`internal/github/client.go`) — after prompt 16

```go
type Client struct {
    httpClient *http.Client
    token      string
}

func NewClient(token string) *Client { ... }

func (c *Client) newGitHubRequest(ctx context.Context, method, url string) (*http.Request, error) {
    // Creates request, sets User-Agent, Accept, and optionally Authorization
}

func (c *Client) FetchRepo(ctx context.Context, repoURL string) (agent.RepoInput, error) { ... }
func (c *Client) fetchRepoMeta(ctx context.Context, owner, repo string) (repoMeta, error) { ... }
func (c *Client) fetchReadme(ctx context.Context, owner, repo string) string { ... }
func parseGitHubURL(rawURL string) (owner, repo string, err error) { ... }
func RepoOwner(repoURL string) string { ... }
```

## Your task

Add two new methods and supporting types for fetching GitHub traffic data.

### 1. Define traffic metric types

Add these types in `internal/github/client.go`:

```go
// TrafficMetrics holds aggregated 14-day views and clones data.
type TrafficMetrics struct {
    Views14dTotal   int `json:"views_14d_total"`
    Views14dUnique  int `json:"views_14d_unique"`
    Clones14dTotal  int `json:"clones_14d_total"`
    Clones14dUnique int `json:"clones_14d_unique"`
}
```

And internal response types matching the GitHub API responses:

```go
type trafficViewsResponse struct {
    Count   int `json:"count"`
    Uniques int `json:"uniques"`
}

type trafficClonesResponse struct {
    Count   int `json:"count"`
    Uniques int `json:"uniques"`
}
```

### 2. Implement `FetchTrafficMetrics`

```go
// FetchTrafficMetrics fetches 14-day views and clones for a repo.
// Requires an authenticated client (token must be set).
// Returns zero-value TrafficMetrics and an error if the token is empty or API calls fail.
func (c *Client) FetchTrafficMetrics(ctx context.Context, owner, repo string) (TrafficMetrics, error)
```

This method should:

1. **Check that the token is set.** If `c.token == ""`, return an error: `"GitHub token required for traffic API"`.

2. **Fetch views** from `GET https://api.github.com/repos/{owner}/{repo}/traffic/views`.
   - Use `c.newGitHubRequest()` to construct the request (which includes the auth header).
   - Parse the JSON response into `trafficViewsResponse`.
   - Extract `count` → `Views14dTotal` and `uniques` → `Views14dUnique`.

3. **Fetch clones** from `GET https://api.github.com/repos/{owner}/{repo}/traffic/clones`.
   - Same approach as views.
   - Extract `count` → `Clones14dTotal` and `uniques` → `Clones14dUnique`.

4. **Return** a populated `TrafficMetrics`.

5. **Error handling:**
   - If the views call fails, return the error (don't attempt clones).
   - If the clones call fails, return the error.
   - HTTP 403 (forbidden) or 404 (not found) should produce clear error messages like: `"traffic API returned HTTP 403 for {owner}/{repo} — check token permissions"`.

### 3. Add a `HasToken` method

```go
// HasToken reports whether the client has an authentication token configured.
func (c *Client) HasToken() bool {
    return c.token != ""
}
```

This will be used by the generate handler to decide whether to attempt traffic calls.

## What NOT to do

- Do NOT modify the store, handler, agent, or frontend.
- Do NOT call `FetchTrafficMetrics` from anywhere yet — the handler will be wired in a later prompt.
- Do NOT change `FetchRepo` or any existing method signatures.
- Do NOT parallelize the views/clones calls yet — keep it sequential and simple for now.

## Verification

After implementation:

1. `go build ./...` compiles without errors.
2. The existing `FetchRepo` behavior is unchanged.
3. The new `TrafficMetrics` type is exported and usable from other packages.
4. `HasToken()` returns `true` when token is set, `false` otherwise.
5. Manual test (optional): run a quick `go test` or temporary `main` that calls `FetchTrafficMetrics("AcaciaMan", "some-repo")` with a real token to verify the API response parsing.
