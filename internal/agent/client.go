package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"text/template"
	"time"
)

// --- Prompt template ---

const promptTemplate = `Generate promotional content for this GitHub repository.

REPO DATA:
{{.RepoDataJSON}}

Return a JSON object with exactly this structure:
{{.OutputSchemaJSON}}

RULES:
- Output ONLY the JSON object. No markdown fences, no preamble, no explanation.
- Echo repo_url and repo_name exactly from the input.
- Generate exactly 3 twitter_posts, each ≤280 characters including hashtags and URL.
- Generate 3–5 key_benefits.
- Generate 5–8 tags, expanding on the input topics.
- linkedin_post should be 150–300 words with professional tone.
- If target_channel is "twitter", optimize tone for Twitter. If "linkedin", optimize for LinkedIn. Always populate all fields regardless.
- Stay faithful to the repo data. Do not invent features not described in the input.
- Tailor content to target_audience if provided.`

const outputSchema = `{
  "repo_url": "string",
  "repo_name": "string",
  "headline": "string",
  "summary": "string",
  "key_benefits": ["string"],
  "tags": ["string"],
  "twitter_posts": ["string"],
  "linkedin_post": "string",
  "call_to_action": "string"
}`

var promptTmpl = template.Must(template.New("prompt").Parse(promptTemplate))

// --- Input types ---

// RepoInput is the structured repo data sent to the agent.
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

// RepoMetrics holds basic repo popularity numbers.
type RepoMetrics struct {
	Stars      int `json:"stars"`
	Forks      int `json:"forks"`
	Watchers   int `json:"watchers"`
	OpenIssues int `json:"open_issues"`
}

// --- Chat completion request/response types ---

type chatRequest struct {
	Messages []chatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletion struct {
	Choices []chatChoice `json:"choices"`
}

type chatChoice struct {
	Message chatMessage `json:"message"`
}

// --- Client ---

// Client calls the Gradient AI agent's chat completion endpoint.
type Client struct {
	endpoint   string
	accessKey  string
	httpClient *http.Client
}

// NewClient creates an agent Client with the given endpoint and access key.
func NewClient(endpoint, accessKey string) *Client {
	return &Client{
		endpoint:  endpoint,
		accessKey: accessKey,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// Generate sends repo data to the agent and returns the promotional JSON.
func (c *Client) Generate(ctx context.Context, input RepoInput) (json.RawMessage, error) {
	// 1. Marshal input to JSON.
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("marshal repo input: %w", err)
	}

	// 2. Render prompt template.
	var promptBuf bytes.Buffer
	err = promptTmpl.Execute(&promptBuf, map[string]string{
		"RepoDataJSON":     string(inputJSON),
		"OutputSchemaJSON": outputSchema,
	})
	if err != nil {
		return nil, fmt.Errorf("render prompt template: %w", err)
	}

	// 3. Build chat completion request.
	reqBody := chatRequest{
		Messages: []chatMessage{
			{Role: "user", Content: promptBuf.String()},
		},
		Stream: false,
	}
	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal chat request: %w", err)
	}

	// 4. Send HTTP POST.
	url := strings.TrimRight(c.endpoint, "/") + "/api/v1/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqJSON))
	if err != nil {
		return nil, fmt.Errorf("create HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.accessKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("agent request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read agent response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("agent returned status %d: %s", resp.StatusCode, string(body))
	}

	// 5. Parse response envelope.
	var completion chatCompletion
	if err := json.Unmarshal(body, &completion); err != nil {
		return nil, fmt.Errorf("parse agent response envelope: %w", err)
	}

	if len(completion.Choices) == 0 {
		return nil, fmt.Errorf("agent returned no choices")
	}

	// 6. Extract content.
	content := strings.TrimSpace(completion.Choices[0].Message.Content)

	// 7. Clean: strip markdown fences if present.
	content = stripMarkdownFences(content)

	// 8. Validate JSON.
	if !json.Valid([]byte(content)) {
		// Fallback: find first '{' and last '}'.
		if start := strings.Index(content, "{"); start != -1 {
			if end := strings.LastIndex(content, "}"); end > start {
				extracted := content[start : end+1]
				if json.Valid([]byte(extracted)) {
					return json.RawMessage(extracted), nil
				}
			}
		}
		return nil, fmt.Errorf("agent response is not valid JSON: %.200s", content)
	}

	return json.RawMessage(content), nil
}

var markdownFenceRe = regexp.MustCompile("(?s)```(?:json)?\\s*(.*?)\\s*```")

func stripMarkdownFences(s string) string {
	matches := markdownFenceRe.FindStringSubmatch(s)
	if len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}
	return s
}
