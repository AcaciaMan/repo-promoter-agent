package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"repo-promoter-agent/internal/agent"
	"repo-promoter-agent/internal/github"
	"repo-promoter-agent/internal/store"
)

// GenerateHandler handles POST /api/generate requests.
type GenerateHandler struct {
	agentClient  *agent.Client
	githubClient *github.Client
	store        *store.Store
}

// NewGenerateHandler creates a GenerateHandler with all dependencies.
func NewGenerateHandler(agentClient *agent.Client, githubClient *github.Client, st *store.Store) *GenerateHandler {
	return &GenerateHandler{
		agentClient:  agentClient,
		githubClient: githubClient,
		store:        st,
	}
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

	// Parse request body.
	var req generateRequest
	if r.Body != nil {
		json.NewDecoder(r.Body).Decode(&req)
	}

	// Normalise target_channel.
	req.TargetChannel = normalizeChannel(req.TargetChannel)

	// Build agent input: real GitHub data or hardcoded fallback.
	var input agent.RepoInput
	repoURL := strings.TrimSpace(req.RepoURL)

	if repoURL != "" {
		fetched, err := h.githubClient.FetchRepo(r.Context(), repoURL)
		if err != nil {
			log.Printf("GitHub fetch failed for %q: %v", repoURL, err)
			writeError(w, http.StatusUnprocessableEntity, "failed to fetch repo: "+err.Error())
			return
		}
		input = fetched
	} else {
		input = defaultRepoInput()
	}

	input.TargetChannel = req.TargetChannel
	input.TargetAudience = req.TargetAudience

	// Fetch traffic metrics for AcaciaMan repos (best-effort) — needed
	// both for agent input (tone adjustment) and for storage/UI.
	var trafficMetrics github.TrafficMetrics
	if repoURL != "" {
		owner := github.RepoOwner(repoURL)
		if owner == "AcaciaMan" && h.githubClient.HasToken() {
			_, repoName, parseErr := github.ParseRepoURL(repoURL)
			if parseErr == nil {
				tm, mErr := h.githubClient.FetchTrafficMetrics(r.Context(), owner, repoName)
				if mErr != nil {
					log.Printf("WARNING: traffic metrics fetch failed for %s/%s: %v", owner, repoName, mErr)
				} else {
					trafficMetrics = tm
					// Include in agent input so they influence generated tone.
					input.Metrics.Views14dTotal = tm.Views14dTotal
					input.Metrics.Views14dUnique = tm.Views14dUnique
					input.Metrics.Clones14dTotal = tm.Clones14dTotal
					input.Metrics.Clones14dUnique = tm.Clones14dUnique
				}
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

	// Parse agent output into a Promotion.
	var promo store.Promotion
	if err := json.Unmarshal(result, &promo); err != nil {
		log.Printf("Failed to parse agent output: %v", err)
		// Return raw result as a fallback so content isn't lost.
		w.Header().Set("Content-Type", "application/json")
		w.Write(result)
		return
	}
	promo.TargetChannel = req.TargetChannel
	promo.TargetAudience = req.TargetAudience

	// Copy traffic metrics to promotion for DB storage and UI display.
	promo.Views14dTotal = trafficMetrics.Views14dTotal
	promo.Views14dUnique = trafficMetrics.Views14dUnique
	promo.Clones14dTotal = trafficMetrics.Clones14dTotal
	promo.Clones14dUnique = trafficMetrics.Clones14dUnique

	// Store (best-effort — never fail the request because of a DB error).
	if err := h.store.Save(r.Context(), &promo); err != nil {
		log.Printf("WARNING: failed to save promotion: %v", err)
	}

	// Return the stored promotion (with id + created_at).
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(promo)
}

func normalizeChannel(ch string) string {
	switch strings.ToLower(strings.TrimSpace(ch)) {
	case "twitter":
		return "twitter"
	case "linkedin":
		return "linkedin"
	default:
		return "general"
	}
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
