package ratelimit

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"sync"
	"time"
)

// BucketConfig defines the rate limit for a named bucket.
type BucketConfig struct {
	Max    int           // maximum requests allowed in the window
	Window time.Duration // rolling window duration
}

// clientState holds per-client, per-bucket timestamp slices.
type clientState struct {
	mu      sync.Mutex
	buckets map[string][]time.Time // key = bucket name ("generate", "search")
}

// Limiter enforces per-client rate limits across named buckets.
type Limiter struct {
	mu      sync.Mutex
	clients map[string]*clientState // key = client identifier (IP)
	configs map[string]BucketConfig // key = bucket name
	nowFunc func() time.Time        // injectable clock for testing; defaults to time.Now
}

// NewLimiter creates a Limiter with the given bucket configurations.
func NewLimiter(configs map[string]BucketConfig) *Limiter {
	return &Limiter{
		clients: make(map[string]*clientState),
		configs: configs,
		nowFunc: time.Now,
	}
}

// AllowResult contains the result of an Allow call.
type AllowResult struct {
	Allowed    bool
	RetryAfter time.Duration
	Count      int // current number of requests in the window (after pruning)
	Max        int // configured maximum for the bucket
}

// Allow checks whether a request from the given client for the given bucket
// should be allowed. Returns an AllowResult with the decision details.
func (l *Limiter) Allow(clientKey, bucket string) AllowResult {
	cfg, ok := l.configs[bucket]
	if !ok {
		log.Printf("ratelimit: WARNING: no config for bucket %q, failing open", bucket)
		return AllowResult{Allowed: true}
	}

	// Get or create client state (global lock).
	l.mu.Lock()
	cs, exists := l.clients[clientKey]
	if !exists {
		cs = &clientState{buckets: make(map[string][]time.Time)}
		l.clients[clientKey] = cs
		l.mu.Unlock()
		log.Printf("ratelimit: new client client=%s bucket=%s", clientKey, bucket)
	} else {
		l.mu.Unlock()
	}

	now := l.nowFunc()
	cutoff := now.Add(-cfg.Window)

	// Per-client lock for timestamp operations.
	cs.mu.Lock()
	defer cs.mu.Unlock()

	timestamps := cs.buckets[bucket]

	// Prune: find first index within the window (timestamps are in order).
	idx := 0
	for idx < len(timestamps) && timestamps[idx].Before(cutoff) {
		idx++
	}
	timestamps = timestamps[idx:]

	// Check: deny if at capacity.
	if len(timestamps) >= cfg.Max {
		retryAfter := timestamps[0].Add(cfg.Window).Sub(now)
		cs.buckets[bucket] = timestamps
		return AllowResult{Allowed: false, RetryAfter: retryAfter, Count: len(timestamps), Max: cfg.Max}
	}

	// Accept: record this request.
	timestamps = append(timestamps, now)
	cs.buckets[bucket] = timestamps
	return AllowResult{Allowed: true, Count: len(timestamps), Max: cfg.Max}
}

// rateLimitError is the JSON response body for 429 responses.
type rateLimitError struct {
	Error             string `json:"error"`
	RetryAfterSeconds int    `json:"retry_after_seconds"`
}

// Middleware returns an http.Handler middleware that enforces rate limits
// for the given bucket.
func (l *Limiter) Middleware(bucket string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip CORS preflight requests.
			if r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			clientKey := ClientKeyFromRequest(r)
			result := l.Allow(clientKey, bucket)

			if result.Allowed {
				next.ServeHTTP(w, r)
				return
			}

			// Rate limited — respond with 429.
			retrySeconds := int(math.Ceil(result.RetryAfter.Seconds()))
			if retrySeconds < 1 {
				retrySeconds = 1
			}

			log.Printf("rate limited: bucket=%s client=%s count=%d/%d retry_after=%s", bucket, clientKey, result.Count, result.Max, result.RetryAfter.Truncate(time.Second))

			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", fmt.Sprintf("%d", retrySeconds))
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(rateLimitError{
				Error:             "rate limit exceeded",
				RetryAfterSeconds: retrySeconds,
			})
		})
	}
}

// StartCleanup begins a background goroutine that periodically removes
// stale client entries. Returns a stop function.
func (l *Limiter) StartCleanup(interval time.Duration) (stop func()) {
	ticker := time.NewTicker(interval)
	done := make(chan struct{})
	var once sync.Once

	go func() {
		for {
			select {
			case <-done:
				ticker.Stop()
				return
			case <-ticker.C:
				l.cleanup()
			}
		}
	}()

	return func() {
		once.Do(func() { close(done) })
	}
}

// cleanup removes stale client entries from the clients map.
func (l *Limiter) cleanup() {
	now := l.nowFunc()

	l.mu.Lock()
	defer l.mu.Unlock()

	evicted := 0
	for key, cs := range l.clients {
		cs.mu.Lock()
		allEmpty := true
		for bucket, timestamps := range cs.buckets {
			cfg, ok := l.configs[bucket]
			if !ok {
				// No config for this bucket; drop the slice.
				delete(cs.buckets, bucket)
				continue
			}
			cutoff := now.Add(-cfg.Window)
			idx := 0
			for idx < len(timestamps) && timestamps[idx].Before(cutoff) {
				idx++
			}
			if idx > 0 {
				cs.buckets[bucket] = timestamps[idx:]
			}
			if len(cs.buckets[bucket]) > 0 {
				allEmpty = false
			}
		}
		cs.mu.Unlock()

		if allEmpty {
			delete(l.clients, key)
			evicted++
		}
	}

	if evicted > 0 {
		log.Printf("ratelimit: cleanup evicted %d stale client(s)", evicted)
	}
}
