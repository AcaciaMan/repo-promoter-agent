package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"text/template"
	"time"
)

// --- Analysis Agent types ---

// AnalysisInput is the structured repo data sent to the Analysis Agent.
type AnalysisInput struct {
	RepoURL          string          `json:"repo_url"`
	RepoName         string          `json:"repo_name"`
	ShortDescription string          `json:"short_description"`
	ReadmeText       string          `json:"readme_text"`
	Topics           []string        `json:"topics"`
	Metrics          AnalysisMetrics `json:"metrics"`
	TargetAudience   string          `json:"target_audience,omitempty"`
}

// AnalysisMetrics holds repo metrics for the Analysis Agent input.
type AnalysisMetrics struct {
	Stars           int `json:"stars"`
	Forks           int `json:"forks"`
	Watchers        int `json:"watchers"`
	Views14dTotal   int `json:"views_14d_total,omitempty"`
	Views14dUnique  int `json:"views_14d_unique,omitempty"`
	Clones14dTotal  int `json:"clones_14d_total,omitempty"`
	Clones14dUnique int `json:"clones_14d_unique,omitempty"`
}

// AnalysisOutput is the structured JSON returned by the Analysis Agent.
type AnalysisOutput struct {
	RepoURL                     string   `json:"repo_url"`
	RepoName                    string   `json:"repo_name"`
	PrimaryValueProposition     string   `json:"primary_value_proposition"`
	IdealAudience               []string `json:"ideal_audience"`
	KeyFeatures                 []string `json:"key_features"`
	Differentiators             []string `json:"differentiators"`
	RiskOrLimitations           []string `json:"risk_or_limitations"`
	SocialProofSignals          []string `json:"social_proof_signals"`
	RecommendedPositioningAngle []string `json:"recommended_positioning_angle"`
}

// --- Analysis Agent prompt template ---

const analysisPromptTemplate = `Analyze this GitHub repository and produce a structured marketing analysis.

REPO DATA:
{{.RepoDataJSON}}

Return a JSON object with exactly this structure:
{{.OutputSchemaJSON}}

RULES:
- Output ONLY the JSON object. No markdown fences, no preamble, no explanation.
- Echo repo_url and repo_name exactly from the input.
- Generate 2–4 items for each array field (ideal_audience, key_features, differentiators, risk_or_limitations, social_proof_signals, recommended_positioning_angle).
- Base every statement strictly on the provided input. Do not invent features, integrations, or capabilities not described in the input data.
- If something is uncertain (e.g., docs quality, test coverage), omit it rather than speculating.
- Keep each string item to ONE sentence, maximum 20 words. Brevity is a hard requirement.
- FORBIDDEN words and phrases: revolutionary, cutting-edge, seamlessly, seamless, robust, powerful, state-of-the-art, game-changing, innovative, next-generation, world-class, heavyweight, elevate. Use plain, specific language instead.
- Tailor to target_audience if provided; otherwise infer from the repo's language and domain.
- For social_proof_signals, be precise and honest:
  - 0–5 stars: "very early stage, minimal public adoption"
  - 6–20 stars: "small user base, early traction"
  - 21–100 stars: "modest community interest"
  - 101–1000 stars: "solid community adoption"
  - 1000+ stars: describe proportionally (e.g., "widely adopted with N stars")
  - If views/clones are zero or absent, say "no recent traffic data available"
  - Do not inflate low numbers or speculate about trends.
- If readme_text is empty or very short, acknowledge limited information explicitly. Output fewer items (minimum 1 per array field) rather than generating generic filler. Say "insufficient data to determine key features" rather than guessing.
- For risk_or_limitations: only list risks clearly evidenced in the input. Examples of valid risks: absent README, zero stars, no description. Do NOT fabricate risks about documentation quality, test coverage, or maintenance status unless explicitly indicated. If risking contradicting yourself, choose only "none clearly indicated from available data". Never output BOTH "none clearly indicated" AND another risk in the same array.
- For recommended_positioning_angle: be specific and actionable. Reference concrete features from the input rather than generic platitudes.`

const analysisOutputSchema = `{
  "repo_url": "string",
  "repo_name": "string",
  "primary_value_proposition": "One sentence explaining what this repo helps users achieve.",
  "ideal_audience": ["Short description of audience segment"],
  "key_features": ["Feature written as a user-facing benefit"],
  "differentiators": ["What makes this repo special vs. alternatives"],
  "risk_or_limitations": ["Caveats or 'none clearly indicated'"],
  "social_proof_signals": ["Interpretation of stars/traffic"],
  "recommended_positioning_angle": ["Suggested marketing angle"]
}`

var analysisPromptTmpl = template.Must(template.New("analysis").Parse(analysisPromptTemplate))

// --- Analysis Agent client ---

// AnalysisClient calls the Analysis Agent's chat completion endpoint.
type AnalysisClient struct {
	endpoint   string
	accessKey  string
	httpClient *http.Client
}

// NewAnalysisClient creates an AnalysisClient with a 5-minute timeout.
func NewAnalysisClient(endpoint, accessKey string) *AnalysisClient {
	return &AnalysisClient{
		endpoint:  endpoint,
		accessKey: accessKey,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

// Analyze sends repo data to the analysis agent and returns the parsed output.
func (c *AnalysisClient) Analyze(ctx context.Context, input AnalysisInput) (*AnalysisOutput, error) {
	start := time.Now()

	// 1. Marshal input to JSON.
	inputJSON, err := json.Marshal(input)
	if err != nil {
		log.Printf("[analysis] failed for %s (duration: %dms, error: %v)", input.RepoName, time.Since(start).Milliseconds(), err)
		return nil, fmt.Errorf("marshal analysis input: %w", err)
	}

	log.Printf("[analysis] calling agent for %s (input size: %d bytes)", input.RepoName, len(inputJSON))

	// 2. Render prompt template.
	var promptBuf bytes.Buffer
	err = analysisPromptTmpl.Execute(&promptBuf, map[string]string{
		"RepoDataJSON":     string(inputJSON),
		"OutputSchemaJSON": analysisOutputSchema,
	})
	if err != nil {
		log.Printf("[analysis] failed for %s (duration: %dms, error: %v)", input.RepoName, time.Since(start).Milliseconds(), err)
		return nil, fmt.Errorf("render analysis prompt template: %w", err)
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
		log.Printf("[analysis] failed for %s (duration: %dms, error: %v)", input.RepoName, time.Since(start).Milliseconds(), err)
		return nil, fmt.Errorf("marshal chat request: %w", err)
	}

	// 4. Send HTTP POST.
	url := strings.TrimRight(c.endpoint, "/") + "/api/v1/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqJSON))
	if err != nil {
		log.Printf("[analysis] failed for %s (duration: %dms, error: %v)", input.RepoName, time.Since(start).Milliseconds(), err)
		return nil, fmt.Errorf("create HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.accessKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		log.Printf("[analysis] failed for %s (duration: %dms, error: %v)", input.RepoName, time.Since(start).Milliseconds(), err)
		return nil, fmt.Errorf("analysis agent request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[analysis] failed for %s (duration: %dms, error: %v)", input.RepoName, time.Since(start).Milliseconds(), err)
		return nil, fmt.Errorf("read analysis agent response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err = fmt.Errorf("analysis agent returned status %d: %s", resp.StatusCode, string(body))
		log.Printf("[analysis] failed for %s (duration: %dms, error: %v)", input.RepoName, time.Since(start).Milliseconds(), err)
		return nil, err
	}

	// 5. Parse response envelope.
	var completion chatCompletion
	if err := json.Unmarshal(body, &completion); err != nil {
		log.Printf("[analysis] failed for %s (duration: %dms, error: %v)", input.RepoName, time.Since(start).Milliseconds(), err)
		return nil, fmt.Errorf("parse analysis agent response envelope: %w", err)
	}

	if len(completion.Choices) == 0 {
		err = fmt.Errorf("analysis agent returned no choices")
		log.Printf("[analysis] failed for %s (duration: %dms, error: %v)", input.RepoName, time.Since(start).Milliseconds(), err)
		return nil, err
	}

	// 6. Extract and clean content.
	content := strings.TrimSpace(completion.Choices[0].Message.Content)
	content = stripMarkdownFences(content)

	// 7. Validate JSON.
	if !json.Valid([]byte(content)) {
		if start := strings.Index(content, "{"); start != -1 {
			if end := strings.LastIndex(content, "}"); end > start {
				extracted := content[start : end+1]
				if json.Valid([]byte(extracted)) {
					content = extracted
				}
			}
		}
		if !json.Valid([]byte(content)) {
			err = fmt.Errorf("analysis agent response is not valid JSON: %.200s", content)
			log.Printf("[analysis] failed for %s (duration: %dms, error: %v)", input.RepoName, time.Since(start).Milliseconds(), err)
			return nil, err
		}
	}

	// 8. Unmarshal into AnalysisOutput.
	var output AnalysisOutput
	if err := json.Unmarshal([]byte(content), &output); err != nil {
		log.Printf("[analysis] failed for %s (duration: %dms, error: %v)", input.RepoName, time.Since(start).Milliseconds(), err)
		return nil, fmt.Errorf("unmarshal analysis output: %w", err)
	}

	log.Printf("[analysis] success for %s (duration: %dms, output size: %d bytes)", input.RepoName, time.Since(start).Milliseconds(), len(content))
	return &output, nil
}
