package ratelimit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func newTestLimiter(configs map[string]BucketConfig) (*Limiter, *time.Time) {
	l := NewLimiter(configs)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	l.nowFunc = func() time.Time { return now }
	return l, &now
}

func TestAllow_GenerateBucket(t *testing.T) {
	l, _ := newTestLimiter(map[string]BucketConfig{
		"generate": {Max: 5, Window: 5 * time.Minute},
	})

	for i := 0; i < 5; i++ {
		result := l.Allow("client-a", "generate")
		if !result.Allowed {
			t.Fatalf("request %d should be allowed", i+1)
		}
		if result.RetryAfter != 0 {
			t.Fatalf("request %d retryAfter should be 0, got %v", i+1, result.RetryAfter)
		}
		if result.Count != i+1 {
			t.Fatalf("request %d count should be %d, got %d", i+1, i+1, result.Count)
		}
		if result.Max != 5 {
			t.Fatalf("request %d max should be 5, got %d", i+1, result.Max)
		}
	}

	// 6th call should be denied.
	result := l.Allow("client-a", "generate")
	if result.Allowed {
		t.Fatal("6th request should be denied")
	}
	if result.RetryAfter <= 0 {
		t.Fatalf("retryAfter should be > 0, got %v", result.RetryAfter)
	}
	if result.Count != 5 {
		t.Fatalf("count should be 5, got %d", result.Count)
	}
	if result.Max != 5 {
		t.Fatalf("max should be 5, got %d", result.Max)
	}
	// All requests at same time, so retryAfter ≈ window.
	if result.RetryAfter < 4*time.Minute || result.RetryAfter > 5*time.Minute {
		t.Fatalf("retryAfter should be ~5m, got %v", result.RetryAfter)
	}
}

func TestAllow_SearchBucket(t *testing.T) {
	l, _ := newTestLimiter(map[string]BucketConfig{
		"search": {Max: 100, Window: 5 * time.Minute},
	})

	for i := 0; i < 100; i++ {
		result := l.Allow("client-a", "search")
		if !result.Allowed {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}

	result := l.Allow("client-a", "search")
	if result.Allowed {
		t.Fatal("101st request should be denied")
	}
	if result.RetryAfter <= 0 {
		t.Fatalf("retryAfter should be > 0, got %v", result.RetryAfter)
	}
	if result.Count != 100 {
		t.Fatalf("count should be 100, got %d", result.Count)
	}
	if result.Max != 100 {
		t.Fatalf("max should be 100, got %d", result.Max)
	}
}

func TestAllow_WindowExpiry(t *testing.T) {
	l, now := newTestLimiter(map[string]BucketConfig{
		"generate": {Max: 2, Window: 1 * time.Minute},
	})

	// 2 allowed.
	for i := 0; i < 2; i++ {
		result := l.Allow("client-a", "generate")
		if !result.Allowed {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}

	// 3rd denied.
	result := l.Allow("client-a", "generate")
	if result.Allowed {
		t.Fatal("3rd request should be denied")
	}

	// Advance past the window.
	*now = now.Add(61 * time.Second)

	// 4th allowed (old timestamps expired).
	result = l.Allow("client-a", "generate")
	if !result.Allowed {
		t.Fatal("4th request should be allowed after window expiry")
	}
	if result.Count != 1 {
		t.Fatalf("count after expiry should be 1, got %d", result.Count)
	}
}

func TestAllow_ClientIsolation(t *testing.T) {
	l, _ := newTestLimiter(map[string]BucketConfig{
		"generate": {Max: 2, Window: 5 * time.Minute},
	})

	// Exhaust client-a.
	for i := 0; i < 2; i++ {
		l.Allow("client-a", "generate")
	}
	result := l.Allow("client-a", "generate")
	if result.Allowed {
		t.Fatal("client-a 3rd request should be denied")
	}

	// client-b should still be allowed.
	result = l.Allow("client-b", "generate")
	if !result.Allowed {
		t.Fatal("client-b 1st request should be allowed")
	}
}

func TestAllow_UnknownBucket(t *testing.T) {
	l, _ := newTestLimiter(map[string]BucketConfig{
		"generate": {Max: 5, Window: 5 * time.Minute},
	})

	result := l.Allow("client-a", "nonexistent")
	if !result.Allowed {
		t.Fatal("unknown bucket should fail open")
	}
	if result.RetryAfter != 0 {
		t.Fatalf("retryAfter should be 0 for unknown bucket, got %v", result.RetryAfter)
	}
}

func TestAllow_RetryAfterAccuracy(t *testing.T) {
	l, now := newTestLimiter(map[string]BucketConfig{
		"generate": {Max: 2, Window: 1 * time.Minute},
	})

	// 1st call at T+0.
	l.Allow("client-a", "generate")

	// Advance 20s, 2nd call.
	*now = now.Add(20 * time.Second)
	l.Allow("client-a", "generate")

	// 3rd call denied. retryAfter should be ~40s (first request exits at T+60s, now is T+20s).
	result := l.Allow("client-a", "generate")
	if result.Allowed {
		t.Fatal("3rd request should be denied")
	}
	if result.Count != 2 {
		t.Fatalf("count should be 2, got %d", result.Count)
	}

	expected := 40 * time.Second
	tolerance := 1 * time.Second
	if result.RetryAfter < expected-tolerance || result.RetryAfter > expected+tolerance {
		t.Fatalf("retryAfter should be ~%v, got %v", expected, result.RetryAfter)
	}
}

func TestClientKeyFromRequest(t *testing.T) {
	tests := []struct {
		name       string
		xff        string
		xri        string
		remoteAddr string
		expected   string
	}{
		{
			name:       "X-Forwarded-For single",
			xff:        "203.0.113.5",
			remoteAddr: "10.0.0.1:1234",
			expected:   "203.0.113.5",
		},
		{
			name:       "X-Forwarded-For multiple",
			xff:        "203.0.113.5, 70.41.3.18",
			remoteAddr: "10.0.0.1:1234",
			expected:   "203.0.113.5",
		},
		{
			name:       "X-Real-IP only",
			xri:        "198.51.100.7",
			remoteAddr: "10.0.0.1:1234",
			expected:   "198.51.100.7",
		},
		{
			name:       "RemoteAddr fallback",
			remoteAddr: "192.168.1.1:54321",
			expected:   "192.168.1.1",
		},
		{
			name:       "IPv6 RemoteAddr",
			remoteAddr: "[::1]:8080",
			expected:   "::1",
		},
		{
			name:       "X-Forwarded-For with spaces",
			xff:        "  203.0.113.5 , 70.41.3.18 ",
			remoteAddr: "10.0.0.1:1234",
			expected:   "203.0.113.5",
		},
		{
			name:       "Empty X-Forwarded-For",
			xff:        " ",
			remoteAddr: "10.0.0.1:1234",
			expected:   "10.0.0.1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.RemoteAddr = tc.remoteAddr
			if tc.xff != "" {
				r.Header.Set("X-Forwarded-For", tc.xff)
			}
			if tc.xri != "" {
				r.Header.Set("X-Real-IP", tc.xri)
			}

			got := ClientKeyFromRequest(r)
			if got != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}

func TestMiddleware_Returns429(t *testing.T) {
	l, _ := newTestLimiter(map[string]BucketConfig{
		"generate": {Max: 1, Window: 5 * time.Minute},
	})

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	wrapped := l.Middleware("generate")(inner)

	// 1st request — allowed.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/generate", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("1st request: expected 200, got %d", rec.Code)
	}

	// 2nd request — rate limited.
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/generate", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("2nd request: expected 429, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", ct)
	}
	ra := rec.Header().Get("Retry-After")
	if ra == "" {
		t.Fatal("expected Retry-After header to be set")
	}

	var body rateLimitError
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode 429 body: %v", err)
	}
	if body.Error != "rate limit exceeded" {
		t.Fatalf("expected error 'rate limit exceeded', got %q", body.Error)
	}
	if body.RetryAfterSeconds <= 0 {
		t.Fatalf("expected retry_after_seconds > 0, got %d", body.RetryAfterSeconds)
	}
}

func TestMiddleware_SkipsOptions(t *testing.T) {
	l, _ := newTestLimiter(map[string]BucketConfig{
		"generate": {Max: 1, Window: 5 * time.Minute},
	})

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	wrapped := l.Middleware("generate")(inner)

	// OPTIONS request — should pass through, not consume budget.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/api/generate", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("OPTIONS request: expected 200, got %d", rec.Code)
	}

	// POST request — should be allowed (first real request).
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/generate", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("POST after OPTIONS: expected 200, got %d", rec.Code)
	}
}

func TestCleanup_RemovesStaleClients(t *testing.T) {
	l, now := newTestLimiter(map[string]BucketConfig{
		"generate": {Max: 5, Window: 1 * time.Minute},
	})

	// Create an entry for client-a.
	l.Allow("client-a", "generate")

	// Verify client-a exists.
	l.mu.Lock()
	if _, exists := l.clients["client-a"]; !exists {
		l.mu.Unlock()
		t.Fatal("client-a should exist after Allow call")
	}
	l.mu.Unlock()

	// Advance past the window.
	*now = now.Add(2 * time.Minute)

	// Trigger cleanup directly.
	l.cleanup()

	// Verify client-a has been removed.
	l.mu.Lock()
	_, exists := l.clients["client-a"]
	l.mu.Unlock()
	if exists {
		t.Fatal("client-a should have been evicted by cleanup")
	}

	// Verify Allow still works fresh for client-a.
	result := l.Allow("client-a", "generate")
	if !result.Allowed {
		t.Fatal("client-a should be allowed after cleanup (fresh state)")
	}
}

func TestAllow_Concurrent(t *testing.T) {
	l := NewLimiter(map[string]BucketConfig{
		"generate": {Max: 1000, Window: 5 * time.Minute},
	})

	var wg sync.WaitGroup
	goroutines := 100
	callsPerGoroutine := 10
	allowedCount := int64(0)
	var mu sync.Mutex

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			localAllowed := 0
			for j := 0; j < callsPerGoroutine; j++ {
				result := l.Allow("client-X", "generate")
				if result.Allowed {
					localAllowed++
				}
			}
			mu.Lock()
			allowedCount += int64(localAllowed)
			mu.Unlock()
		}()
	}

	wg.Wait()

	totalCalls := goroutines * callsPerGoroutine
	if allowedCount != int64(totalCalls) {
		t.Fatalf("expected %d allowed calls, got %d", totalCalls, allowedCount)
	}
}
