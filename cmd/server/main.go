package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	"repo-promoter-agent/internal/agent"
	"repo-promoter-agent/internal/github"
	"repo-promoter-agent/internal/handler"
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
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "promotions.db"
	}

	// Create store.
	st, err := store.New(dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer st.Close()

	// Create clients.
	agentClient := agent.NewClient(endpoint, accessKey)
	githubClient := github.NewClient()

	// Set up routes.
	mux := http.NewServeMux()
	mux.Handle("/api/generate", handler.NewGenerateHandler(agentClient, githubClient, st))
	mux.Handle("/api/search", handler.NewSearchHandler(st))
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

// noCacheHandler wraps a handler to set Cache-Control: no-cache for development.
func noCacheHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		h.ServeHTTP(w, r)
	})
}
