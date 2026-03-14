package main

import (
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"

	"repo-promoter-agent/internal/agent"
	"repo-promoter-agent/internal/github"
	"repo-promoter-agent/internal/handler"
	"repo-promoter-agent/internal/ratelimit"
	"repo-promoter-agent/internal/store"
	"repo-promoter-agent/static"
)

func main() {
	// Load .env file (fail gracefully if it doesn't exist).
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables directly")
	}

	// Required env vars — fail fast if missing.
	endpoint := mustEnv("AGENT_ENDPOINT")
	accessKey := mustEnv("AGENT_ACCESS_KEY")

	// Optional env vars.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	solrURL := os.Getenv("SOLR_URL")
	if solrURL == "" {
		solrURL = "http://localhost:8983"
	}
	solrCore := os.Getenv("SOLR_CORE")
	if solrCore == "" {
		solrCore = "promotions"
	}

	// Create store.
	st, err := store.New(solrURL, solrCore)
	if err != nil {
		log.Fatalf("Failed to connect to Solr: %v", err)
	}
	defer st.Close()
	log.Printf("Connected to Solr at %s (core: %s)", solrURL, solrCore)

	// Create clients.
	agentClient := agent.NewClient(endpoint, accessKey)

	ghToken := os.Getenv("GITHUB_TOKEN")
	if ghToken != "" {
		log.Println("GitHub token configured — authenticated API access enabled")
	} else {
		log.Println("No GITHUB_TOKEN set — using unauthenticated GitHub API (60 req/hr limit)")
	}
	githubClient := github.NewClient(ghToken)

	// Analysis Agent (optional — enables repo analysis feature).
	analysisEndpoint := os.Getenv("ANALYSIS_AGENT_ENDPOINT")
	analysisKey := os.Getenv("ANALYSIS_AGENT_ACCESS_KEY")
	var analysisClient *agent.AnalysisClient
	if analysisEndpoint != "" && analysisKey != "" {
		analysisClient = agent.NewAnalysisClient(analysisEndpoint, analysisKey)
		log.Println("Analysis Agent configured — repo analysis feature enabled")
	} else {
		log.Println("Analysis Agent not configured — repo analysis feature disabled (set ANALYSIS_AGENT_ENDPOINT and ANALYSIS_AGENT_ACCESS_KEY to enable)")
	}

	// Create rate limiter.
	generateMax := envIntOr("RATE_LIMIT_GENERATE_MAX", 5)
	searchMax := envIntOr("RATE_LIMIT_SEARCH_MAX", 100)

	limiter := ratelimit.NewLimiter(map[string]ratelimit.BucketConfig{
		"generate": {Max: generateMax, Window: 5 * time.Minute},
		"search":   {Max: searchMax, Window: 5 * time.Minute},
	})
	stopCleanup := limiter.StartCleanup(10 * time.Minute)
	defer stopCleanup()
	log.Printf("Rate limiter enabled: generate=%d/5m0s, search=%d/5m0s", generateMax, searchMax)

	// Set up routes.
	mux := http.NewServeMux()
	mux.Handle("/api/generate", limiter.Middleware("generate")(handler.NewGenerateHandler(agentClient, githubClient, st, analysisClient)))
	mux.Handle("/api/search", limiter.Middleware("search")(handler.NewSearchHandler(st)))
	mux.Handle("/", noCacheHandler(http.FileServerFS(static.Files)))

	addr := ":" + port
	log.Printf("Server listening on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required environment variable %s is not set", key)
	}
	return v
}

// envIntOr reads an integer from the named env var, or returns the default.
// Logs a warning if the value is set but not a valid integer.
// A value of 0 is treated as "disable" and returns math.MaxInt32.
func envIntOr(key string, defaultVal int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		log.Printf("WARNING: %s=%q is not a valid integer, using default %d", key, v, defaultVal)
		return defaultVal
	}
	if n == 0 {
		log.Printf("%s=0 — rate limiting disabled for this bucket", key)
		return math.MaxInt32
	}
	return n
}

// noCacheHandler wraps a handler to set Cache-Control: no-cache for development.
func noCacheHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		h.ServeHTTP(w, r)
	})
}
