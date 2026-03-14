package store

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Promotion represents a stored promotional content record.
type Promotion struct {
	ID              int64           `json:"id"`
	RepoURL         string          `json:"repo_url"`
	RepoName        string          `json:"repo_name"`
	Headline        string          `json:"headline"`
	Summary         string          `json:"summary"`
	KeyBenefits     []string        `json:"key_benefits"`
	Tags            []string        `json:"tags"`
	TwitterPosts    []string        `json:"twitter_posts"`
	LinkedInPost    string          `json:"linkedin_post"`
	CallToAction    string          `json:"call_to_action"`
	TargetChannel   string          `json:"target_channel"`
	TargetAudience  string          `json:"target_audience"`
	CreatedAt       time.Time       `json:"created_at"`
	Stars           int             `json:"stars"`
	Forks           int             `json:"forks"`
	Watchers        int             `json:"watchers"`
	Views14dTotal   int             `json:"views_14d_total"`
	Views14dUnique  int             `json:"views_14d_unique"`
	Clones14dTotal  int             `json:"clones_14d_total"`
	Clones14dUnique int             `json:"clones_14d_unique"`
	Readme          string          `json:"readme"`
	AnalysisJSON    json.RawMessage `json:"analysis"`
}

// Store is a Solr-backed store for promotional content.
type Store struct {
	baseURL string
	core    string
	client  *http.Client
}

// New creates a Solr-backed store and pings the core to verify connectivity.
func New(solrURL, core string) (*Store, error) {
	s := &Store{
		baseURL: strings.TrimRight(solrURL, "/"),
		core:    core,
		client:  &http.Client{Timeout: 30 * time.Second},
	}

	pingURL := fmt.Sprintf("%s/solr/%s/admin/ping", s.baseURL, s.core)
	resp, err := s.client.Get(pingURL)
	if err != nil {
		return nil, fmt.Errorf("ping solr: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("solr ping returned status %d", resp.StatusCode)
	}

	return s, nil
}

// Close is a no-op — the HTTP client is stateless.
func (s *Store) Close() error {
	return nil
}

// Save upserts a Promotion document into Solr. The document's unique key is
// the repo_url value. On success it sets p.CreatedAt.
func (s *Store) Save(ctx context.Context, p *Promotion) error {
	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now()
	}

	doc := map[string]interface{}{
		"id":                p.RepoURL,
		"repo_url":          p.RepoURL,
		"repo_name":         p.RepoName,
		"headline":          p.Headline,
		"summary":           p.Summary,
		"key_benefits":      orEmptySlice(p.KeyBenefits),
		"tags":              orEmptySlice(p.Tags),
		"twitter_posts":     orEmptySlice(p.TwitterPosts),
		"linkedin_post":     p.LinkedInPost,
		"call_to_action":    p.CallToAction,
		"target_channel":    p.TargetChannel,
		"target_audience":   p.TargetAudience,
		"created_at":        p.CreatedAt.UTC().Format(time.RFC3339),
		"stars":             p.Stars,
		"forks":             p.Forks,
		"watchers":          p.Watchers,
		"views_14d_total":   p.Views14dTotal,
		"views_14d_unique":  p.Views14dUnique,
		"clones_14d_total":  p.Clones14dTotal,
		"clones_14d_unique": p.Clones14dUnique,
		"readme":            p.Readme,
	}

	if p.AnalysisJSON != nil {
		doc["analysis_json"] = string(p.AnalysisJSON)
	}

	body, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshal document: %w", err)
	}

	updateURL := fmt.Sprintf("%s/solr/%s/update/json/docs?commit=true", s.baseURL, s.core)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, updateURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("post to solr: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read solr response: %w", err)
	}

	var solrResp struct {
		ResponseHeader struct {
			Status int `json:"status"`
		} `json:"responseHeader"`
	}
	if err := json.Unmarshal(respBody, &solrResp); err != nil {
		return fmt.Errorf("parse solr response: %w", err)
	}
	if solrResp.ResponseHeader.Status != 0 {
		return fmt.Errorf("solr update error: status %d, body: %s", solrResp.ResponseHeader.Status, string(respBody))
	}

	p.ID = 0
	return nil
}

// Search performs a full-text search across promotions using edismax and
// returns matching results ordered by relevance.
func (s *Store) Search(ctx context.Context, query string, limit int) ([]Promotion, error) {
	if limit <= 0 {
		limit = 20
	}

	q := strings.TrimSpace(query)
	if q == "" {
		return []Promotion{}, nil
	}
	q = sanitizeSolrQuery(q)

	params := url.Values{
		"q":       {q},
		"defType": {"edismax"},
		"qf":      {"repo_name headline summary key_benefits tags twitter_posts linkedin_post call_to_action target_audience readme"},
		"rows":    {fmt.Sprintf("%d", limit)},
		"sort":    {"score desc"},
		"wt":      {"json"},
		"fl":      {"*"},
	}

	selectURL := fmt.Sprintf("%s/solr/%s/select?%s", s.baseURL, s.core, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, selectURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create search request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search solr: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read search response: %w", err)
	}

	return parseSolrDocs(body)
}

// List returns the most recent promotions ordered by created_at descending.
func (s *Store) List(ctx context.Context, limit int) ([]Promotion, error) {
	if limit <= 0 {
		limit = 20
	}

	params := url.Values{
		"q":    {"*:*"},
		"rows": {fmt.Sprintf("%d", limit)},
		"sort": {"created_at desc"},
		"wt":   {"json"},
		"fl":   {"*"},
	}

	selectURL := fmt.Sprintf("%s/solr/%s/select?%s", s.baseURL, s.core, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, selectURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create list request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list from solr: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read list response: %w", err)
	}

	return parseSolrDocs(body)
}

// --- helpers ---

// parseSolrDocs extracts Promotion records from a Solr select JSON response.
func parseSolrDocs(body []byte) ([]Promotion, error) {
	var envelope struct {
		Response struct {
			Docs []map[string]interface{} `json:"docs"`
		} `json:"response"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("parse solr response: %w", err)
	}

	result := make([]Promotion, 0, len(envelope.Response.Docs))
	for _, doc := range envelope.Response.Docs {
		p := Promotion{
			ID:              0,
			RepoURL:         getString(doc, "repo_url"),
			RepoName:        getString(doc, "repo_name"),
			Headline:        getString(doc, "headline"),
			Summary:         getString(doc, "summary"),
			KeyBenefits:     getStringSlice(doc, "key_benefits"),
			Tags:            getStringSlice(doc, "tags"),
			TwitterPosts:    getStringSlice(doc, "twitter_posts"),
			LinkedInPost:    getString(doc, "linkedin_post"),
			CallToAction:    getString(doc, "call_to_action"),
			TargetChannel:   getString(doc, "target_channel"),
			TargetAudience:  getString(doc, "target_audience"),
			Stars:           getInt(doc, "stars"),
			Forks:           getInt(doc, "forks"),
			Watchers:        getInt(doc, "watchers"),
			Views14dTotal:   getInt(doc, "views_14d_total"),
			Views14dUnique:  getInt(doc, "views_14d_unique"),
			Clones14dTotal:  getInt(doc, "clones_14d_total"),
			Clones14dUnique: getInt(doc, "clones_14d_unique"),
			Readme:          getString(doc, "readme"),
		}

		if s := getString(doc, "created_at"); s != "" {
			p.CreatedAt = parseTime(s)
		}

		if s := getString(doc, "analysis_json"); s != "" {
			p.AnalysisJSON = json.RawMessage(s)
		}

		result = append(result, p)
	}
	return result, nil
}

func getString(doc map[string]interface{}, key string) string {
	v, ok := doc[key]
	if !ok || v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case []interface{}:
		if len(val) > 0 {
			if s, ok := val[0].(string); ok {
				return s
			}
		}
	}
	return fmt.Sprintf("%v", v)
}

func getStringSlice(doc map[string]interface{}, key string) []string {
	v, ok := doc[key]
	if !ok || v == nil {
		return []string{}
	}
	switch val := v.(type) {
	case []interface{}:
		out := make([]string, 0, len(val))
		for _, item := range val {
			if s, ok := item.(string); ok {
				out = append(out, s)
			} else {
				out = append(out, fmt.Sprintf("%v", item))
			}
		}
		return out
	case string:
		return []string{val}
	}
	return []string{}
}

func getInt(doc map[string]interface{}, key string) int {
	v, ok := doc[key]
	if !ok || v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return int(val)
	case int:
		return val
	case int64:
		return int(val)
	}
	return 0
}

func parseTime(s string) time.Time {
	for _, layout := range []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.000Z",
		"2006-01-02 15:04:05",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

func orEmptySlice(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// sanitizeSolrQuery escapes Solr special characters in a user query.
func sanitizeSolrQuery(query string) string {
	replacer := strings.NewReplacer(
		`\`, `\\`,
		`+`, `\+`,
		`-`, `\-`,
		`!`, `\!`,
		`(`, `\(`,
		`)`, `\)`,
		`{`, `\{`,
		`}`, `\}`,
		`[`, `\[`,
		`]`, `\]`,
		`^`, `\^`,
		`"`, `\"`,
		`~`, `\~`,
		`*`, `\*`,
		`?`, `\?`,
		`:`, `\:`,
		`/`, `\/`,
	)
	escaped := replacer.Replace(query)

	// Also escape && and || sequences
	escaped = strings.ReplaceAll(escaped, `\&\&`, `\&&`)
	escaped = strings.ReplaceAll(escaped, `\|\|`, `\||`)

	return escaped
}
