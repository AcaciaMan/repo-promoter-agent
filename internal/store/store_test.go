package store

import (
	"context"
	"encoding/json"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	st, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
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
		"repo_url": "https://github.com/testowner/testrepo",
		"repo_name": "testrepo",
		"primary_value_proposition": "Helps developers test efficiently.",
		"ideal_audience": ["Go developers", "TDD practitioners"],
		"key_features": ["Fast execution", "Simple API"],
		"differentiators": ["Minimal dependencies"],
		"risk_or_limitations": ["Early-stage project"],
		"social_proof_signals": ["Modest traction"],
		"recommended_positioning_angle": ["Lightweight testing"]
	}`)
}

func TestSave_WithAnalysis(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	p := samplePromotion()
	p.AnalysisJSON = sampleAnalysisJSON()

	if err := st.Save(ctx, p); err != nil {
		t.Fatalf("Save() error: %v", err)
	}
	if p.ID <= 0 {
		t.Errorf("expected ID > 0, got %d", p.ID)
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

	// Marshal to JSON and verify "analysis" field is null
	b, err := json.Marshal(results[0])
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}
	val, ok := m["analysis"]
	if !ok {
		t.Fatal("expected 'analysis' key in marshaled JSON")
	}
	if val != nil {
		t.Errorf("expected 'analysis' to be null, got %v", val)
	}
}

func TestSave_ReplacesOldPromotion_WithAnalysis(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	// Save first without analysis
	p1 := samplePromotion()
	if err := st.Save(ctx, p1); err != nil {
		t.Fatalf("Save() first error: %v", err)
	}

	// Save second with same repo_url, with analysis
	p2 := samplePromotion()
	p2.Headline = "Updated Headline"
	p2.AnalysisJSON = sampleAnalysisJSON()
	if err := st.Save(ctx, p2); err != nil {
		t.Fatalf("Save() second error: %v", err)
	}

	results, err := st.List(ctx, 10)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result (old deleted), got %d", len(results))
	}
	if results[0].AnalysisJSON == nil {
		t.Error("expected replacement promotion to have AnalysisJSON set")
	}
	if results[0].Headline != "Updated Headline" {
		t.Errorf("expected updated headline, got %q", results[0].Headline)
	}
}

func TestSearch_ReturnsAnalysis(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	p := samplePromotion()
	p.AnalysisJSON = sampleAnalysisJSON()
	if err := st.Save(ctx, p); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	results, err := st.Search(ctx, "testrepo", 10)
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].AnalysisJSON == nil {
		t.Fatal("expected AnalysisJSON to be non-nil in search result")
	}
	if !json.Valid(results[0].AnalysisJSON) {
		t.Fatal("AnalysisJSON from search is not valid JSON")
	}
}

func TestSearch_ReturnsNullAnalysis(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	p := samplePromotion()
	if err := st.Save(ctx, p); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	results, err := st.Search(ctx, "testrepo", 10)
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].AnalysisJSON != nil {
		t.Errorf("expected AnalysisJSON to be nil, got %s", results[0].AnalysisJSON)
	}
}

func TestList_MixedAnalysis(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	// Promotion A with analysis
	pA := samplePromotion()
	pA.RepoURL = "https://github.com/testowner/alpha"
	pA.RepoName = "alpha"
	pA.AnalysisJSON = sampleAnalysisJSON()
	if err := st.Save(ctx, pA); err != nil {
		t.Fatalf("Save(alpha) error: %v", err)
	}

	// Promotion B without analysis
	pB := samplePromotion()
	pB.RepoURL = "https://github.com/testowner/beta"
	pB.RepoName = "beta"
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

	var withAnalysis, withoutAnalysis int
	for _, r := range results {
		if r.AnalysisJSON != nil {
			withAnalysis++
		} else {
			withoutAnalysis++
		}
	}
	if withAnalysis != 1 {
		t.Errorf("expected 1 with analysis, got %d", withAnalysis)
	}
	if withoutAnalysis != 1 {
		t.Errorf("expected 1 without analysis, got %d", withoutAnalysis)
	}
}

func TestSave_AnalysisJSON_RoundTrip(t *testing.T) {
	st := newTestStore(t)
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

	var m map[string]interface{}
	if err := json.Unmarshal(results[0].AnalysisJSON, &m); err != nil {
		t.Fatalf("failed to unmarshal AnalysisJSON: %v", err)
	}

	// Check primary_value_proposition
	pvp, ok := m["primary_value_proposition"].(string)
	if !ok {
		t.Fatal("primary_value_proposition is not a string")
	}
	if pvp != "Helps developers test efficiently." {
		t.Errorf("primary_value_proposition = %q, want %q", pvp, "Helps developers test efficiently.")
	}

	// Check ideal_audience is an array with expected length
	audience, ok := m["ideal_audience"].([]interface{})
	if !ok {
		t.Fatal("ideal_audience is not an array")
	}
	if len(audience) != 2 {
		t.Errorf("ideal_audience length = %d, want 2", len(audience))
	}

	// Check key_features contains expected items
	features, ok := m["key_features"].([]interface{})
	if !ok {
		t.Fatal("key_features is not an array")
	}
	expectedFeatures := map[string]bool{"Fast execution": false, "Simple API": false}
	for _, f := range features {
		if s, ok := f.(string); ok {
			expectedFeatures[s] = true
		}
	}
	for k, found := range expectedFeatures {
		if !found {
			t.Errorf("key_features missing expected item %q", k)
		}
	}
}

func TestSave_AnalysisJSON_MarshalToJSON(t *testing.T) {
	st := newTestStore(t)
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

	b, err := json.Marshal(results[0])
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	analysis, ok := m["analysis"]
	if !ok {
		t.Fatal("expected 'analysis' key in marshaled JSON")
	}

	// analysis should be a nested object (map), not a string
	analysisMap, ok := analysis.(map[string]interface{})
	if !ok {
		t.Fatalf("expected 'analysis' to be a nested object, got %T", analysis)
	}
	if _, ok := analysisMap["primary_value_proposition"]; !ok {
		t.Error("nested analysis object missing 'primary_value_proposition'")
	}
}

func TestSave_AnalysisJSON_NullMarshalToJSON(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	p := samplePromotion()
	if err := st.Save(ctx, p); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	results, err := st.List(ctx, 1)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}

	b, err := json.Marshal(results[0])
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	val, ok := m["analysis"]
	if !ok {
		t.Fatal("expected 'analysis' key in marshaled JSON")
	}
	if val != nil {
		t.Errorf("expected 'analysis' to be null, got %v", val)
	}
}

func TestMigration_AnalysisColumn(t *testing.T) {
	st := newTestStore(t)

	// Run applyMigrations again — should be idempotent
	if err := st.applyMigrations(); err != nil {
		t.Fatalf("second applyMigrations() error: %v", err)
	}

	// Run a third time for good measure
	if err := st.applyMigrations(); err != nil {
		t.Fatalf("third applyMigrations() error: %v", err)
	}
}
