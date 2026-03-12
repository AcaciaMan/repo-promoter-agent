package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"repo-promoter-agent/internal/agent"
)

// GenerateHandler handles POST /api/generate requests.
type GenerateHandler struct {
	agentClient *agent.Client
}

// NewGenerateHandler creates a GenerateHandler with the given agent client.
func NewGenerateHandler(agentClient *agent.Client) *GenerateHandler {
	return &GenerateHandler{agentClient: agentClient}
}

type generateRequest struct {
	RepoURL        string `json:"repo_url"`
	TargetChannel  string `json:"target_channel"`
	TargetAudience string `json:"target_audience"`
}

func (h *GenerateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	log.Printf("POST /api/generate from %s", r.RemoteAddr)

	// Parse request body. Use hardcoded defaults if body is missing or invalid.
	input := defaultRepoInput()
	var req generateRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			if req.RepoURL != "" {
				input.RepoURL = req.RepoURL
			}
			if req.TargetChannel != "" {
				input.TargetChannel = req.TargetChannel
			}
			if req.TargetAudience != "" {
				input.TargetAudience = req.TargetAudience
			}
		}
	}

	// Call the agent.
	start := time.Now()
	result, err := h.agentClient.Generate(r.Context(), input)
	elapsed := time.Since(start)

	if err != nil {
		log.Printf("Agent call failed after %s: %v", elapsed, err)
		writeError(w, http.StatusBadGateway, "agent request failed: "+err.Error())
		return
	}

	log.Printf("Agent call succeeded in %s (%d bytes)", elapsed, len(result))

	w.Header().Set("Content-Type", "application/json")
	w.Write(result)
}

func defaultRepoInput() agent.RepoInput {
	return agent.RepoInput{
		RepoURL:          "https://github.com/AcaciaMan/village-square",
		RepoName:         "Village Square",
		ShortDescription: "Digital village square for local announcements, garage sales, and connections between neighbors and suppliers.",
		ReadmeSummary:    "A community web app for a rural village — connecting villagers, local producers (fishermen, farmers, crafters), and the yearly Village Day celebration. Built with Go and designed for simplicity.",
		PrimaryLanguage:  "Go",
		Topics:           []string{"go", "community", "local"},
		Metrics: agent.RepoMetrics{
			Stars:      12,
			Forks:      3,
			Watchers:   5,
			OpenIssues: 2,
		},
		TargetChannel:  "twitter",
		TargetAudience: "Villagers and small-community organizers",
	}
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
