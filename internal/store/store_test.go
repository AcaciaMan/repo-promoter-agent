//go:build integration

package store

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	solrURL := os.Getenv("SOLR_URL")
	if solrURL == "" {
		solrURL = "http://localhost:8983"
	}
	solrCore := os.Getenv("SOLR_CORE")
	if solrCore == "" {
		solrCore = "promotions"
	}
	st, err := New(solrURL, solrCore)
	if err != nil {
		t.Fatalf("failed to create Solr store: %v", err)
	}
	return st
}

func cleanupSolr(t *testing.T, st *Store) {
	t.Helper()
	payload := []byte(`{"delete":{"query":"*:*"},"commit":{}}`)
	url := fmt.Sprintf("%s/solr/%s/update", st.baseURL, st.core)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("cleanup: create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := st.client.Do(req)
	if err != nil {
		t.Fatalf("cleanup: post delete: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("cleanup: solr returned status %d", resp.StatusCode)
	}
}

func samplePromotion() *Promotion {
	return &Promotion{
		RepoURL:        "https://github.com/testowner/testrepo",
		RepoName:       "testrepo",
		Headline:       "Test Headline",
		Summary:        "Test Summary",
		KeyBenefits:    []string{"benefit1", "benefit2"},
		Tags:           []string{"go", "testing"},
		TwitterPosts:   []string{"tweet1", "tweet2", "tweet3"},
		LinkedInPost:   "LinkedIn post content",
		CallToAction:   "Star the repo!",
		TargetChannel:  "general",
		TargetAudience: "Go developers",
	}
}

func sampleAnalysisJSON() json.RawMessage {
	return json.RawMessage(`{
		"primary_value_proposition": "Helps developers test efficiently.",
		"ideal_audience": ["Go developers", "TDD practitioners"],
		"key_features": ["Fast execution", "Simple API"]
	}`)
}

func TestSave_Basic(t *testing.T) {
	st := newTestStore(t)
	cleanupSolr(t, st)
	ctx := context.Background()

	p := samplePromotion()
	if err := st.Save(ctx, p); err != nil {
		t.Fatalf("Save() error: %v", err)
	}
	if p.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be non-zero after save")
	}
}

func TestSave_WithAnalysis(t *testing.T) {
	st := newTestStore(t)
	cleanupSolr(t, st)
	ctx := context.Background()

	p := samplePromotion()
	p.AnalysisJSON = sampleAnalysisJSON()

	if err := st.Save(ctx, p); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	results, err := st.List(ctx, 1)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].AnalysisJSON == nil {
		t.Fatal("expected AnalysisJSON to be non-nil")
	}
	if !json.Valid(results[0].AnalysisJSON) {
		t.Fatal("AnalysisJSON is not valid JSON")
	}
	var m map[string]interface{}
	if err := json.Unmarshal(results[0].AnalysisJSON, &m); err != nil {
		t.Fatalf("failed to unmarshal AnalysisJSON: %v", err)
	}
	if _, ok := m["primary_value_proposition"]; !ok {
		t.Error("expected AnalysisJSON to contain 'primary_value_proposition'")
	}
}

func TestSave_WithoutAnalysis(t *testing.T) {
	st := newTestStore(t)
	cleanupSolr(t, st)
	ctx := context.Background()

	p := samplePromotion()
	// AnalysisJSON left as nil

	if err := st.Save(ctx, p); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	results, err := st.List(ctx, 1)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].AnalysisJSON != nil {
		t.Errorf("expected AnalysisJSON to be nil, got %s", results[0].AnalysisJSON)
	}
}

func TestSave_Upsert(t *testing.T) {
	st := newTestStore(t)
	cleanupSolr(t, st)
	ctx := context.Background()

	p1 := samplePromotion()
	p1.Headline = "Original"
	if err := st.Save(ctx, p1); err != nil {
		t.Fatalf("Save() first error: %v", err)
	}

	p2 := samplePromotion()
	p2.Headline = "Updated"
	if err := st.Save(ctx, p2); err != nil {
		t.Fatalf("Save() second error: %v", err)
	}

	results, err := st.List(ctx, 10)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result (upsert), got %d", len(results))
	}
	if results[0].Headline != "Updated" {
		t.Errorf("expected headline %q, got %q", "Updated", results[0].Headline)
	}
}

func TestSearch_FullText(t *testing.T) {
	st := newTestStore(t)
	cleanupSolr(t, st)
	ctx := context.Background()

	p := samplePromotion()
	p.Summary = "A tool for kubernetes deployment automation"
	if err := st.Save(ctx, p); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	results, err := st.Search(ctx, "kubernetes", 10)
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].RepoName != "testrepo" {
		t.Errorf("expected repo_name %q, got %q", "testrepo", results[0].RepoName)
	}
}

func TestSearch_EmptyQuery(t *testing.T) {
	st := newTestStore(t)
	cleanupSolr(t, st)
	ctx := context.Background()

	results, err := st.Search(ctx, "", 10)
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty query, got %d", len(results))
	}
}

func TestSearch_NoMatch(t *testing.T) {
	st := newTestStore(t)
	cleanupSolr(t, st)
	ctx := context.Background()

	results, err := st.Search(ctx, "nonexistent", 10)
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestList_OrderByDate(t *testing.T) {
	st := newTestStore(t)
	cleanupSolr(t, st)
	ctx := context.Background()

	pA := samplePromotion()
	pA.RepoURL = "https://github.com/testowner/alpha"
	pA.RepoName = "alpha"
	pA.CreatedAt = time.Date(2026, 3, 14, 10, 0, 0, 0, time.UTC)
	if err := st.Save(ctx, pA); err != nil {
		t.Fatalf("Save(alpha) error: %v", err)
	}

	pB := samplePromotion()
	pB.RepoURL = "https://github.com/testowner/beta"
	pB.RepoName = "beta"
	pB.CreatedAt = time.Date(2026, 3, 14, 12, 0, 0, 0, time.UTC)
	if err := st.Save(ctx, pB); err != nil {
		t.Fatalf("Save(beta) error: %v", err)
	}

	results, err := st.List(ctx, 10)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].RepoName != "beta" {
		t.Errorf("expected first result to be beta (more recent), got %q", results[0].RepoName)
	}
	if results[1].RepoName != "alpha" {
		t.Errorf("expected second result to be alpha (older), got %q", results[1].RepoName)
	}
}

func TestList_RespectsLimit(t *testing.T) {
	st := newTestStore(t)
	cleanupSolr(t, st)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		p := samplePromotion()
		p.RepoURL = fmt.Sprintf("https://github.com/testowner/repo%d", i)
		p.RepoName = fmt.Sprintf("repo%d", i)
		p.CreatedAt = time.Date(2026, 3, 14, 10+i, 0, 0, 0, time.UTC)
		if err := st.Save(ctx, p); err != nil {
			t.Fatalf("Save(repo%d) error: %v", i, err)
		}
	}

	results, err := st.List(ctx, 2)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results with limit 2, got %d", len(results))
	}
}

func TestList_Empty(t *testing.T) {
	st := newTestStore(t)
	cleanupSolr(t, st)
	ctx := context.Background()

	results, err := st.List(ctx, 10)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if results == nil {
		t.Fatal("expected non-nil empty slice, got nil")
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestSearch_SpecialCharacters(t *testing.T) {
	st := newTestStore(t)
	cleanupSolr(t, st)
	ctx := context.Background()

	p := samplePromotion()
	if err := st.Save(ctx, p); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Should not crash — query is sanitized
	_, err := st.Search(ctx, "C++ (advanced)", 10)
	if err != nil {
		t.Fatalf("Search() with special chars error: %v", err)
	}
}

func TestSave_WithTrafficMetrics(t *testing.T) {
	st := newTestStore(t)
	cleanupSolr(t, st)
	ctx := context.Background()

	p := samplePromotion()
	p.Stars = 42
	p.Forks = 7
	p.Watchers = 15
	p.Views14dTotal = 100
	p.Views14dUnique = 50
	p.Clones14dTotal = 20
	p.Clones14dUnique = 10

	if err := st.Save(ctx, p); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	results, err := st.List(ctx, 1)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	r := results[0]
	if r.Stars != 42 {
		t.Errorf("Stars = %d, want 42", r.Stars)
	}
	if r.Forks != 7 {
		t.Errorf("Forks = %d, want 7", r.Forks)
	}
	if r.Watchers != 15 {
		t.Errorf("Watchers = %d, want 15", r.Watchers)
	}
	if r.Views14dTotal != 100 {
		t.Errorf("Views14dTotal = %d, want 100", r.Views14dTotal)
	}
	if r.Views14dUnique != 50 {
		t.Errorf("Views14dUnique = %d, want 50", r.Views14dUnique)
	}
	if r.Clones14dTotal != 20 {
		t.Errorf("Clones14dTotal = %d, want 20", r.Clones14dTotal)
	}
	if r.Clones14dUnique != 10 {
		t.Errorf("Clones14dUnique = %d, want 10", r.Clones14dUnique)
	}
}
