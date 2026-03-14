package cors

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func dummyHandler() (http.Handler, *bool) {
	called := false
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	return h, &called
}

func TestMiddleware_EmptyConfig(t *testing.T) {
	next, called := dummyHandler()
	handler := Middleware(Config{})(next)

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d; want %d", rec.Code, http.StatusOK)
	}
	if v := rec.Header().Get("Access-Control-Allow-Origin"); v != "" {
		t.Errorf("Access-Control-Allow-Origin = %q; want empty", v)
	}
	if !*called {
		t.Error("next handler was not called")
	}
}

func TestMiddleware_NoOriginHeader(t *testing.T) {
	next, called := dummyHandler()
	handler := Middleware(Config{
		AllowedOrigins: []string{"http://localhost:3000"},
	})(next)

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d; want %d", rec.Code, http.StatusOK)
	}
	if v := rec.Header().Get("Access-Control-Allow-Origin"); v != "" {
		t.Errorf("Access-Control-Allow-Origin = %q; want empty", v)
	}
	if !*called {
		t.Error("next handler was not called")
	}
}

func TestMiddleware_AllowedOrigin(t *testing.T) {
	next, called := dummyHandler()
	handler := Middleware(Config{
		AllowedOrigins: []string{"http://localhost:3000"},
	})(next)

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if v := rec.Header().Get("Access-Control-Allow-Origin"); v != "http://localhost:3000" {
		t.Errorf("Access-Control-Allow-Origin = %q; want %q", v, "http://localhost:3000")
	}
	if v := rec.Header().Get("Vary"); v != "Origin" {
		t.Errorf("Vary = %q; want %q", v, "Origin")
	}
	if !*called {
		t.Error("next handler was not called")
	}
}

func TestMiddleware_DisallowedOrigin(t *testing.T) {
	next, called := dummyHandler()
	handler := Middleware(Config{
		AllowedOrigins: []string{"http://localhost:3000"},
	})(next)

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Origin", "http://evil.example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if v := rec.Header().Get("Access-Control-Allow-Origin"); v != "" {
		t.Errorf("Access-Control-Allow-Origin = %q; want empty", v)
	}
	if v := rec.Header().Get("Vary"); v != "Origin" {
		t.Errorf("Vary = %q; want %q", v, "Origin")
	}
	if !*called {
		t.Error("next handler was not called")
	}
}

func TestMiddleware_PreflightAllowed(t *testing.T) {
	next, called := dummyHandler()
	handler := Middleware(Config{
		AllowedOrigins: []string{"http://localhost:3000"},
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type"},
		MaxAge:         3600,
	})(next)

	req := httptest.NewRequest(http.MethodOptions, "/api/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d; want %d", rec.Code, http.StatusNoContent)
	}
	if v := rec.Header().Get("Access-Control-Allow-Origin"); v != "http://localhost:3000" {
		t.Errorf("Access-Control-Allow-Origin = %q; want %q", v, "http://localhost:3000")
	}
	if v := rec.Header().Get("Access-Control-Allow-Methods"); v != "GET, POST, OPTIONS" {
		t.Errorf("Access-Control-Allow-Methods = %q; want %q", v, "GET, POST, OPTIONS")
	}
	if v := rec.Header().Get("Access-Control-Allow-Headers"); v != "Content-Type" {
		t.Errorf("Access-Control-Allow-Headers = %q; want %q", v, "Content-Type")
	}
	if v := rec.Header().Get("Access-Control-Max-Age"); v != "3600" {
		t.Errorf("Access-Control-Max-Age = %q; want %q", v, "3600")
	}
	if v := rec.Header().Get("Vary"); v != "Origin" {
		t.Errorf("Vary = %q; want %q", v, "Origin")
	}
	if *called {
		t.Error("next handler should not have been called for preflight")
	}
}

func TestMiddleware_PreflightDisallowed(t *testing.T) {
	next, called := dummyHandler()
	handler := Middleware(Config{
		AllowedOrigins: []string{"http://localhost:3000"},
	})(next)

	req := httptest.NewRequest(http.MethodOptions, "/api/test", nil)
	req.Header.Set("Origin", "http://evil.example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if v := rec.Header().Get("Access-Control-Allow-Origin"); v != "" {
		t.Errorf("Access-Control-Allow-Origin = %q; want empty", v)
	}
	if !*called {
		t.Error("next handler was not called")
	}
}

func TestMiddleware_MultipleOrigins(t *testing.T) {
	cfg := Config{
		AllowedOrigins: []string{"http://localhost:3000", "https://app.example.com"},
	}

	tests := []struct {
		name       string
		origin     string
		wantAllow  string
		wantCalled bool
	}{
		{"second origin matches", "https://app.example.com", "https://app.example.com", true},
		{"first origin matches", "http://localhost:3000", "http://localhost:3000", true},
		{"no origin matches", "http://other.example.com", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next, called := dummyHandler()
			handler := Middleware(cfg)(next)

			req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
			req.Header.Set("Origin", tt.origin)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if v := rec.Header().Get("Access-Control-Allow-Origin"); v != tt.wantAllow {
				t.Errorf("Access-Control-Allow-Origin = %q; want %q", v, tt.wantAllow)
			}
			if *called != tt.wantCalled {
				t.Errorf("called = %v; want %v", *called, tt.wantCalled)
			}
		})
	}
}

func TestMiddleware_MaxAgeZero(t *testing.T) {
	next, _ := dummyHandler()
	handler := Middleware(Config{
		AllowedOrigins: []string{"http://localhost:3000"},
		AllowedMethods: []string{"GET"},
		AllowedHeaders: []string{"Content-Type"},
		MaxAge:         0,
	})(next)

	req := httptest.NewRequest(http.MethodOptions, "/api/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d; want %d", rec.Code, http.StatusNoContent)
	}
	if v := rec.Header().Get("Access-Control-Max-Age"); v != "" {
		t.Errorf("Access-Control-Max-Age = %q; want empty", v)
	}
}

func TestOriginAllowed(t *testing.T) {
	tests := []struct {
		name    string
		origin  string
		allowed []string
		want    bool
	}{
		{"exact match", "http://localhost:3000", []string{"http://localhost:3000"}, true},
		{"no match", "http://evil.com", []string{"http://localhost:3000"}, false},
		{"empty origin", "", []string{"http://localhost:3000"}, false},
		{"nil allowed", "http://localhost:3000", nil, false},
		{"empty allowed", "http://localhost:3000", []string{}, false},
		{"case sensitive", "HTTP://LOCALHOST:3000", []string{"http://localhost:3000"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := originAllowed(tt.origin, tt.allowed)
			if got != tt.want {
				t.Errorf("originAllowed(%q, %v) = %v; want %v", tt.origin, tt.allowed, got, tt.want)
			}
		})
	}
}
