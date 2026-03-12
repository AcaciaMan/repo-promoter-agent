package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	"repo-promoter-agent/internal/agent"
	"repo-promoter-agent/internal/handler"
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

	// Create agent client.
	agentClient := agent.NewClient(endpoint, accessKey)

	// Set up routes.
	mux := http.NewServeMux()
	mux.Handle("/api/generate", handler.NewGenerateHandler(agentClient))
	mux.Handle("/", http.FileServer(http.Dir("static")))

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
