package github

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"repo-promoter-agent/internal/agent"
)

// Client talks to the GitHub REST API.
type Client struct {
	httpClient *http.Client
	token      string
}

// NewClient returns a GitHub client with a sensible timeout.
// If token is non-empty, all requests will use authenticated access.
func NewClient(token string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		token:      token,
	}
}

// newGitHubRequest creates an HTTP request with standard GitHub API headers.
func (c *Client) newGitHubRequest(ctx context.Context, method, url string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "repo-promoter-agent")
	req.Header.Set("Accept", "application/vnd.github+json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	return req, nil
}

// FetchRepo fetches public repo metadata + README from GitHub and returns a
// populated agent.RepoInput. TargetChannel and TargetAudience are left empty —
// the caller fills those in.
func (c *Client) FetchRepo(ctx context.Context, repoURL string) (agent.RepoInput, error) {
	owner, repo, err := parseGitHubURL(repoURL)
	if err != nil {
		return agent.RepoInput{}, err
	}

	// 1. Repo metadata
	meta, err := c.fetchRepoMeta(ctx, owner, repo)
	if err != nil {
		return agent.RepoInput{}, err
	}

	// 2. README (best-effort)
	readme := c.fetchReadme(ctx, owner, repo)

	description := ""
	if meta.Description != nil {
		description = *meta.Description
	}
	language := ""
	if meta.Language != nil {
		language = *meta.Language
	}

	return agent.RepoInput{
		RepoURL:          fmt.Sprintf("https://github.com/%s/%s", owner, repo),
		RepoName:         meta.FullName,
		ShortDescription: description,
		ReadmeSummary:    readme,
		PrimaryLanguage:  language,
		Topics:           meta.Topics,
		Metrics: agent.RepoMetrics{
			Stars:      meta.StargazersCount,
			Forks:      meta.ForksCount,
			Watchers:   meta.SubscribersCount,
			OpenIssues: meta.OpenIssuesCount,
		},
	}, nil
}

// --- GitHub API response types (only fields we need) ---

type repoMeta struct {
	Name             string   `json:"name"`
	FullName         string   `json:"full_name"`
	Description      *string  `json:"description"`
	Language         *string  `json:"language"`
	Topics           []string `json:"topics"`
	StargazersCount  int      `json:"stargazers_count"`
	ForksCount       int      `json:"forks_count"`
	SubscribersCount int      `json:"subscribers_count"`
	OpenIssuesCount  int      `json:"open_issues_count"`
}

type readmeResponse struct {
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
}

// TrafficMetrics holds aggregated 14-day views and clones data.
type TrafficMetrics struct {
	Views14dTotal   int `json:"views_14d_total"`
	Views14dUnique  int `json:"views_14d_unique"`
	Clones14dTotal  int `json:"clones_14d_total"`
	Clones14dUnique int `json:"clones_14d_unique"`
}

type trafficViewsResponse struct {
	Count   int `json:"count"`
	Uniques int `json:"uniques"`
}

type trafficClonesResponse struct {
	Count   int `json:"count"`
	Uniques int `json:"uniques"`
}

// --- Internal helpers ---

func (c *Client) fetchRepoMeta(ctx context.Context, owner, repo string) (repoMeta, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)
	req, err := c.newGitHubRequest(ctx, http.MethodGet, url)
	if err != nil {
		return repoMeta{}, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return repoMeta{}, fmt.Errorf("GitHub API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return repoMeta{}, fmt.Errorf("failed to read GitHub response: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return repoMeta{}, fmt.Errorf("repository %s/%s not found", owner, repo)
	}
	if resp.StatusCode != http.StatusOK {
		return repoMeta{}, fmt.Errorf("GitHub API error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var meta repoMeta
	if err := json.Unmarshal(body, &meta); err != nil {
		return repoMeta{}, fmt.Errorf("failed to parse GitHub response: %w", err)
	}
	return meta, nil
}

const maxReadmeLength = 2000

func (c *Client) fetchReadme(ctx context.Context, owner, repo string) string {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/readme", owner, repo)
	req, err := c.newGitHubRequest(ctx, http.MethodGet, url)
	if err != nil {
		return "No README available."
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "No README available."
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "No README available."
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "No README available."
	}

	var rr readmeResponse
	if err := json.Unmarshal(body, &rr); err != nil {
		return "No README available."
	}

	if rr.Encoding != "base64" {
		return "No README available."
	}

	decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(rr.Content, "\n", ""))
	if err != nil {
		return "No README available."
	}

	text := string(decoded)
	if len(text) > maxReadmeLength {
		text = text[:maxReadmeLength]
	}
	return text
}

// HasToken reports whether the client has an authentication token configured.
func (c *Client) HasToken() bool {
	return c.token != ""
}

// FetchTrafficMetrics fetches 14-day views and clones for a repo.
// Requires an authenticated client (token must be set).
func (c *Client) FetchTrafficMetrics(ctx context.Context, owner, repo string) (TrafficMetrics, error) {
	if c.token == "" {
		return TrafficMetrics{}, fmt.Errorf("GitHub token required for traffic API")
	}

	// Fetch views.
	viewsURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/traffic/views", owner, repo)
	viewsReq, err := c.newGitHubRequest(ctx, http.MethodGet, viewsURL)
	if err != nil {
		return TrafficMetrics{}, fmt.Errorf("failed to create views request: %w", err)
	}
	viewsResp, err := c.httpClient.Do(viewsReq)
	if err != nil {
		return TrafficMetrics{}, fmt.Errorf("views API request failed: %w", err)
	}
	defer viewsResp.Body.Close()
	viewsBody, err := io.ReadAll(viewsResp.Body)
	if err != nil {
		return TrafficMetrics{}, fmt.Errorf("failed to read views response: %w", err)
	}
	if viewsResp.StatusCode == http.StatusForbidden || viewsResp.StatusCode == http.StatusNotFound {
		return TrafficMetrics{}, fmt.Errorf("traffic API returned HTTP %d for %s/%s — check token permissions", viewsResp.StatusCode, owner, repo)
	}
	if viewsResp.StatusCode != http.StatusOK {
		return TrafficMetrics{}, fmt.Errorf("views API error (HTTP %d): %s", viewsResp.StatusCode, string(viewsBody))
	}
	var views trafficViewsResponse
	if err := json.Unmarshal(viewsBody, &views); err != nil {
		return TrafficMetrics{}, fmt.Errorf("failed to parse views response: %w", err)
	}

	// Fetch clones.
	clonesURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/traffic/clones", owner, repo)
	clonesReq, err := c.newGitHubRequest(ctx, http.MethodGet, clonesURL)
	if err != nil {
		return TrafficMetrics{}, fmt.Errorf("failed to create clones request: %w", err)
	}
	clonesResp, err := c.httpClient.Do(clonesReq)
	if err != nil {
		return TrafficMetrics{}, fmt.Errorf("clones API request failed: %w", err)
	}
	defer clonesResp.Body.Close()
	clonesBody, err := io.ReadAll(clonesResp.Body)
	if err != nil {
		return TrafficMetrics{}, fmt.Errorf("failed to read clones response: %w", err)
	}
	if clonesResp.StatusCode == http.StatusForbidden || clonesResp.StatusCode == http.StatusNotFound {
		return TrafficMetrics{}, fmt.Errorf("traffic API returned HTTP %d for %s/%s — check token permissions", clonesResp.StatusCode, owner, repo)
	}
	if clonesResp.StatusCode != http.StatusOK {
		return TrafficMetrics{}, fmt.Errorf("clones API error (HTTP %d): %s", clonesResp.StatusCode, string(clonesBody))
	}
	var clones trafficClonesResponse
	if err := json.Unmarshal(clonesBody, &clones); err != nil {
		return TrafficMetrics{}, fmt.Errorf("failed to parse clones response: %w", err)
	}

	return TrafficMetrics{
		Views14dTotal:   views.Count,
		Views14dUnique:  views.Uniques,
		Clones14dTotal:  clones.Count,
		Clones14dUnique: clones.Uniques,
	}, nil
}

// RepoOwner extracts the owner from a GitHub URL. Returns empty string on error.
func RepoOwner(repoURL string) string {
	owner, _, err := parseGitHubURL(repoURL)
	if err != nil {
		return ""
	}
	return owner
}

// ParseRepoURL extracts owner and repo from a GitHub URL.
func ParseRepoURL(rawURL string) (owner, repo string, err error) {
	return parseGitHubURL(rawURL)
}

// parseGitHubURL extracts owner and repo from various GitHub URL formats:
//   - https://github.com/owner/repo
//   - https://github.com/owner/repo/
//   - github.com/owner/repo
func parseGitHubURL(rawURL string) (owner, repo string, err error) {
	s := strings.TrimSpace(rawURL)
	// Strip scheme if present
	s = strings.TrimPrefix(s, "https://")
	s = strings.TrimPrefix(s, "http://")
	// Strip trailing slash
	s = strings.TrimRight(s, "/")

	// Expect: github.com/owner/repo
	if !strings.HasPrefix(s, "github.com/") {
		return "", "", fmt.Errorf("invalid GitHub URL: %q (must start with github.com)", rawURL)
	}

	parts := strings.Split(strings.TrimPrefix(s, "github.com/"), "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid GitHub URL: %q (expected github.com/owner/repo)", rawURL)
	}

	repo = strings.TrimSuffix(parts[1], ".git")
	return parts[0], repo, nil
}
