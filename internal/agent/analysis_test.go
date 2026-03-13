package agent

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func fakeAnalysisResponse(content string) string {
	resp := map[string]interface{}{
		"choices": []map[string]interface{}{
			{
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": content,
				},
			},
		},
	}
	b, _ := json.Marshal(resp)
	return string(b)
}

func sampleAnalysisInput() AnalysisInput {
	return AnalysisInput{
		RepoURL:          "https://github.com/testowner/testrepo",
		RepoName:         "testrepo",
		ShortDescription: "A test repository for unit testing",
		ReadmeText:       "# Test Repo\n\nThis is a test repo with some features.",
		Topics:           []string{"go", "testing"},
		Metrics: AnalysisMetrics{
			Stars:    42,
			Forks:    5,
			Watchers: 3,
		},
		TargetAudience: "Go developers",
	}
}

const validAnalysisJSON = `{
  "repo_url": "https://github.com/testowner/testrepo",
  "repo_name": "testrepo",
  "primary_value_proposition": "Helps Go developers test their code more efficiently.",
  "ideal_audience": ["Go developers writing unit tests", "Teams adopting TDD practices"],
  "key_features": ["Fast test execution", "Simple assertion API"],
  "differentiators": ["Minimal dependencies compared to testify"],
  "risk_or_limitations": ["Early-stage project with limited documentation"],
  "social_proof_signals": ["Modest traction with 42 stars"],
  "recommended_positioning_angle": ["Lightweight alternative to heavy test frameworks"]
}`

func TestAnalyze_HappyPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(fakeAnalysisResponse(validAnalysisJSON)))
	}))
	defer server.Close()

	client := NewAnalysisClient(server.URL, "test-key")
	output, err := client.Analyze(context.Background(), sampleAnalysisInput())
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if output.RepoURL != "https://github.com/testowner/testrepo" {
		t.Errorf("RepoURL = %q, want %q", output.RepoURL, "https://github.com/testowner/testrepo")
	}
	if output.RepoName != "testrepo" {
		t.Errorf("RepoName = %q, want %q", output.RepoName, "testrepo")
	}
	if output.PrimaryValueProposition == "" {
		t.Errorf("PrimaryValueProposition is empty")
	}
	if len(output.IdealAudience) < 1 {
		t.Errorf("IdealAudience has %d items, want at least 1", len(output.IdealAudience))
	}
	if len(output.KeyFeatures) < 1 {
		t.Errorf("KeyFeatures has %d items, want at least 1", len(output.KeyFeatures))
	}
	if len(output.Differentiators) < 1 {
		t.Errorf("Differentiators has %d items, want at least 1", len(output.Differentiators))
	}
}

func TestAnalyze_MarkdownFences(t *testing.T) {
	fenced := "```json\n" + validAnalysisJSON + "\n```"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(fakeAnalysisResponse(fenced)))
	}))
	defer server.Close()

	client := NewAnalysisClient(server.URL, "test-key")
	output, err := client.Analyze(context.Background(), sampleAnalysisInput())
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if output.RepoName != "testrepo" {
		t.Errorf("RepoName = %q, want %q", output.RepoName, "testrepo")
	}
}

func TestAnalyze_EmptyReadme(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(fakeAnalysisResponse(validAnalysisJSON)))
	}))
	defer server.Close()

	input := sampleAnalysisInput()
	input.ReadmeText = ""

	client := NewAnalysisClient(server.URL, "test-key")
	output, err := client.Analyze(context.Background(), input)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if output.RepoName != "testrepo" {
		t.Errorf("RepoName = %q, want %q", output.RepoName, "testrepo")
	}
}

func TestAnalyze_EmptyTopics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(fakeAnalysisResponse(validAnalysisJSON)))
	}))
	defer server.Close()

	input := sampleAnalysisInput()
	input.Topics = []string{}

	client := NewAnalysisClient(server.URL, "test-key")
	output, err := client.Analyze(context.Background(), input)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if output.RepoName != "testrepo" {
		t.Errorf("RepoName = %q, want %q", output.RepoName, "testrepo")
	}
}

func TestAnalyze_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(fakeAnalysisResponse("This is not JSON at all")))
	}))
	defer server.Close()

	client := NewAnalysisClient(server.URL, "test-key")
	_, err := client.Analyze(context.Background(), sampleAnalysisInput())
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
	if !strings.Contains(err.Error(), "not valid JSON") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "not valid JSON")
	}
}

func TestAnalyze_PartialJSON(t *testing.T) {
	partial := "Here is the analysis: " + validAnalysisJSON + " Hope that helps!"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(fakeAnalysisResponse(partial)))
	}))
	defer server.Close()

	client := NewAnalysisClient(server.URL, "test-key")
	output, err := client.Analyze(context.Background(), sampleAnalysisInput())
	if err != nil {
		t.Fatalf("expected no error (fallback extraction), got: %v", err)
	}
	if output.RepoName != "testrepo" {
		t.Errorf("RepoName = %q, want %q", output.RepoName, "testrepo")
	}
}

func TestAnalyze_AgentHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	client := NewAnalysisClient(server.URL, "test-key")
	_, err := client.Analyze(context.Background(), sampleAnalysisInput())
	if err == nil {
		t.Fatal("expected error for HTTP 500, got nil")
	}
	if !strings.Contains(err.Error(), "status 500") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "status 500")
	}
}

func TestAnalyze_EmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices": []}`))
	}))
	defer server.Close()

	client := NewAnalysisClient(server.URL, "test-key")
	_, err := client.Analyze(context.Background(), sampleAnalysisInput())
	if err == nil {
		t.Fatal("expected error for empty choices, got nil")
	}
	if !strings.Contains(err.Error(), "no choices") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "no choices")
	}
}

func TestAnalyze_ContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(fakeAnalysisResponse(validAnalysisJSON)))
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	client := NewAnalysisClient(server.URL, "test-key")
	_, err := client.Analyze(ctx, sampleAnalysisInput())
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
}

func TestAnalyze_ExtraFieldsIgnored(t *testing.T) {
	extraJSON := `{
  "repo_url": "https://github.com/testowner/testrepo",
  "repo_name": "testrepo",
  "primary_value_proposition": "Helps Go developers test their code.",
  "ideal_audience": ["Go developers"],
  "key_features": ["Fast tests"],
  "differentiators": ["Minimal deps"],
  "risk_or_limitations": ["Early-stage"],
  "social_proof_signals": ["42 stars"],
  "recommended_positioning_angle": ["Lightweight"],
  "extra_field": "should be ignored"
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(fakeAnalysisResponse(extraJSON)))
	}))
	defer server.Close()

	client := NewAnalysisClient(server.URL, "test-key")
	output, err := client.Analyze(context.Background(), sampleAnalysisInput())
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if output.RepoName != "testrepo" {
		t.Errorf("RepoName = %q, want %q", output.RepoName, "testrepo")
	}
}

func TestAnalyze_MissingOptionalFields(t *testing.T) {
	minimalJSON := `{
  "repo_url": "https://github.com/testowner/testrepo",
  "repo_name": "testrepo",
  "primary_value_proposition": "A test tool."
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(fakeAnalysisResponse(minimalJSON)))
	}))
	defer server.Close()

	client := NewAnalysisClient(server.URL, "test-key")
	output, err := client.Analyze(context.Background(), sampleAnalysisInput())
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if output.RepoURL != "https://github.com/testowner/testrepo" {
		t.Errorf("RepoURL = %q, want %q", output.RepoURL, "https://github.com/testowner/testrepo")
	}
	if output.RepoName != "testrepo" {
		t.Errorf("RepoName = %q, want %q", output.RepoName, "testrepo")
	}
	if output.PrimaryValueProposition != "A test tool." {
		t.Errorf("PrimaryValueProposition = %q, want %q", output.PrimaryValueProposition, "A test tool.")
	}
	if len(output.IdealAudience) != 0 {
		t.Errorf("IdealAudience has %d items, want 0", len(output.IdealAudience))
	}
	if len(output.KeyFeatures) != 0 {
		t.Errorf("KeyFeatures has %d items, want 0", len(output.KeyFeatures))
	}
}

func TestAnalyze_RequestFormat(t *testing.T) {
	var capturedMethod string
	var capturedPath string
	var capturedContentType string
	var capturedAuth string
	var capturedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		capturedContentType = r.Header.Get("Content-Type")
		capturedAuth = r.Header.Get("Authorization")
		capturedBody, _ = io.ReadAll(r.Body)

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(fakeAnalysisResponse(validAnalysisJSON)))
	}))
	defer server.Close()

	client := NewAnalysisClient(server.URL, "test-key")
	_, err := client.Analyze(context.Background(), sampleAnalysisInput())
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if capturedMethod != http.MethodPost {
		t.Errorf("Method = %q, want %q", capturedMethod, http.MethodPost)
	}
	if capturedPath != "/api/v1/chat/completions" {
		t.Errorf("Path = %q, want %q", capturedPath, "/api/v1/chat/completions")
	}
	if capturedContentType != "application/json" {
		t.Errorf("Content-Type = %q, want %q", capturedContentType, "application/json")
	}
	if capturedAuth != "Bearer test-key" {
		t.Errorf("Authorization = %q, want %q", capturedAuth, "Bearer test-key")
	}

	// Validate request body structure.
	var reqBody struct {
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(capturedBody, &reqBody); err != nil {
		t.Fatalf("request body is not valid JSON: %v", err)
	}
	if len(reqBody.Messages) != 1 {
		t.Fatalf("messages count = %d, want 1", len(reqBody.Messages))
	}
	if reqBody.Messages[0].Role != "user" {
		t.Errorf("message role = %q, want %q", reqBody.Messages[0].Role, "user")
	}
	if !strings.Contains(reqBody.Messages[0].Content, "testrepo") {
		t.Errorf("message content does not contain %q", "testrepo")
	}
}
