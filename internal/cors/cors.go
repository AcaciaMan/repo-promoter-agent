package cors

import (
	"net/http"
	"strconv"
	"strings"
)

// Config holds the CORS middleware configuration.
type Config struct {
	AllowedOrigins []string // explicit list of allowed origins (e.g., "https://example.com")
	AllowedMethods []string // HTTP methods to allow (e.g., "GET", "POST", "OPTIONS")
	AllowedHeaders []string // request headers to allow (e.g., "Content-Type")
	MaxAge         int      // preflight cache duration in seconds (Access-Control-Max-Age)
}

// Middleware returns a standard middleware that adds CORS headers based on cfg.
// If cfg.AllowedOrigins is empty, the middleware is a no-op passthrough.
func Middleware(cfg Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		// No-op passthrough when no origins are configured.
		if len(cfg.AllowedOrigins) == 0 {
			return next
		}

		methods := strings.Join(cfg.AllowedMethods, ", ")
		headers := strings.Join(cfg.AllowedHeaders, ", ")

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// No Origin header — same-origin or non-browser client; pass through.
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Always set Vary: Origin when the middleware is active.
			w.Header().Set("Vary", "Origin")

			if !originAllowed(origin, cfg.AllowedOrigins) {
				// Origin present but not allowed — no CORS headers; pass through.
				next.ServeHTTP(w, r)
				return
			}

			// Origin matches — set CORS allow header.
			w.Header().Set("Access-Control-Allow-Origin", origin)

			// Preflight handling.
			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Methods", methods)
				w.Header().Set("Access-Control-Allow-Headers", headers)
				if cfg.MaxAge > 0 {
					w.Header().Set("Access-Control-Max-Age", strconv.Itoa(cfg.MaxAge))
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// originAllowed reports whether origin is in the allowed list (exact match).
func originAllowed(origin string, allowed []string) bool {
	for _, a := range allowed {
		if a == origin {
			return true
		}
	}
	return false
}
