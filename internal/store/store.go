package store

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
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

// SearchResult holds search results together with facet counts.
type SearchResult struct {
	Results    []Promotion                  `json:"results"`
	Facets     map[string][]Facet           `json:"facets,omitempty"`
	Highlights map[string]map[string]string `json:"highlights,omitempty"`
	Collation  string                       `json:"collation,omitempty"`
}

// Facet represents a single facet value and its document count.
type Facet struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

// SearchOptions holds optional filter parameters for search and list queries.
type SearchOptions struct {
	Tags     []string // Filter by exact tag values (AND logic: all must match)
	Channel  string   // Filter by target_channel exact value
	MinStars int      // Filter to docs with stars >= this value (0 = no filter)
	Sort     string   // Sort order: "relevance", "newest", "stars", "views" (empty = default)
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

	// Warmup: fire a lightweight query so Solr opens searchers / fills caches
	// before real user traffic arrives. Errors are non-fatal.
	warmStart := time.Now()
	warmURL := fmt.Sprintf("%s/solr/%s/select?q=*:*&rows=1&wt=json", s.baseURL, s.core)
	if wr, err := s.client.Get(warmURL); err == nil {
		io.Copy(io.Discard, wr.Body)
		wr.Body.Close()
		log.Printf("Solr warmup query completed in %s", time.Since(warmStart))
	} else {
		log.Printf("Solr warmup query failed (non-fatal): %v", err)
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

		// Deserialize analysis JSON to populate indexed analysis fields.
		var analysis struct {
			PrimaryValueProposition     string   `json:"primary_value_proposition"`
			IdealAudience               []string `json:"ideal_audience"`
			KeyFeatures                 []string `json:"key_features"`
			Differentiators             []string `json:"differentiators"`
			RecommendedPositioningAngle []string `json:"recommended_positioning_angle"`
		}
		if err := json.Unmarshal(p.AnalysisJSON, &analysis); err == nil {
			if analysis.PrimaryValueProposition != "" {
				doc["analysis_value_proposition"] = analysis.PrimaryValueProposition
			}
			if len(analysis.IdealAudience) > 0 {
				doc["analysis_ideal_audience"] = analysis.IdealAudience
			}
			if len(analysis.KeyFeatures) > 0 {
				doc["analysis_key_features"] = analysis.KeyFeatures
			}
			if len(analysis.Differentiators) > 0 {
				doc["analysis_differentiators"] = analysis.Differentiators
			}
			if len(analysis.RecommendedPositioningAngle) > 0 {
				doc["analysis_positioning"] = analysis.RecommendedPositioningAngle
			}
		}
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
func (s *Store) Search(ctx context.Context, query string, limit int, opts SearchOptions) (SearchResult, error) {
	if limit <= 0 {
		limit = 20
	}

	q := strings.TrimSpace(query)
	if q == "" {
		return SearchResult{}, nil
	}
	q = sanitizeSolrQuery(q)

	// edismax with field boosting: headline/tags/name weighted highest,
	// summary mid-tier, social posts baseline, readme lowest.
	// pf boosts exact phrase matches in key fields; ps allows 2-word slop.
	// mm requires most query terms to match; tie lets other fields contribute.
	params := url.Values{
		"q":                        {q},
		"defType":                  {"edismax"},
		"qf":                       {"repo_name^3 headline^4 summary^2 key_benefits^1.5 tags^3 twitter_posts^1 linkedin_post^1 call_to_action^1 target_audience^1.5 readme^0.5 analysis_value_proposition^2 analysis_key_features^1.5 analysis_differentiators^1.5 analysis_ideal_audience^1 analysis_positioning^1"},
		"pf":                       {"headline^6 summary^3 repo_name^4"},
		"ps":                       {"2"},
		"mm":                       {"2<-1 5<80%"},
		"tie":                      {"0.1"},
		"rows":                     {fmt.Sprintf("%d", limit)},
		"sort":                     {solrSort(opts.Sort, true)},
		"wt":                       {"json"},
		"fl":                       {"*"},
		"facet":                    {"true"},
		"facet.field":              {"tags", "target_channel"},
		"facet.mincount":           {"1"},
		"hl":                       {"true"},
		"hl.fl":                    {"headline,summary,key_benefits,linkedin_post,call_to_action,target_audience,analysis_value_proposition,analysis_key_features,analysis_differentiators"},
		"hl.simple.pre":            {"<mark>"},
		"hl.simple.post":           {"</mark>"},
		"hl.snippets":              {"2"},
		"hl.fragsize":              {"200"},
		"hl.method":                {"unified"},
		"spellcheck":               {"true"},
		"spellcheck.collate":       {"true"},
		"spellcheck.count":         {"5"},
		"spellcheck.maxCollations": {"1"},
	}
	applyFilters(params, opts)

	selectURL := fmt.Sprintf("%s/solr/%s/select?%s", s.baseURL, s.core, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, selectURL, nil)
	if err != nil {
		return SearchResult{}, fmt.Errorf("create search request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return SearchResult{}, fmt.Errorf("search solr: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return SearchResult{}, fmt.Errorf("read search response: %w", err)
	}

	docs, err := parseSolrDocs(body)
	if err != nil {
		return SearchResult{}, err
	}
	facets := parseFacets(body)
	highlights := parseHighlights(body)
	collation := parseCollation(body)
	return SearchResult{Results: docs, Facets: facets, Highlights: highlights, Collation: collation}, nil
}

// List returns the most recent promotions ordered by created_at descending.
func (s *Store) List(ctx context.Context, limit int, opts SearchOptions) (SearchResult, error) {
	if limit <= 0 {
		limit = 20
	}

	params := url.Values{
		"q":              {"*:*"},
		"rows":           {fmt.Sprintf("%d", limit)},
		"sort":           {solrSort(opts.Sort, false)},
		"wt":             {"json"},
		"fl":             {"*"},
		"facet":          {"true"},
		"facet.field":    {"tags", "target_channel"},
		"facet.mincount": {"1"},
	}
	applyFilters(params, opts)

	selectURL := fmt.Sprintf("%s/solr/%s/select?%s", s.baseURL, s.core, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, selectURL, nil)
	if err != nil {
		return SearchResult{}, fmt.Errorf("create list request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return SearchResult{}, fmt.Errorf("list from solr: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return SearchResult{}, fmt.Errorf("read list response: %w", err)
	}

	docs, err := parseSolrDocs(body)
	if err != nil {
		return SearchResult{}, err
	}
	facets := parseFacets(body)
	return SearchResult{Results: docs, Facets: facets}, nil
}

// Suggestion represents a single autocomplete suggestion.
type Suggestion struct {
	Term   string `json:"term"`
	Weight int    `json:"weight"`
}

// Suggest returns autocomplete suggestions for the given prefix.
func (s *Store) Suggest(ctx context.Context, prefix string, limit int) ([]Suggestion, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 20 {
		limit = 20
	}

	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return nil, nil
	}

	params := url.Values{
		"suggest.q":     {prefix},
		"suggest.count": {fmt.Sprintf("%d", limit)},
	}
	suggestURL := fmt.Sprintf("%s/solr/%s/suggest?%s", s.baseURL, s.core, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, suggestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create suggest request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("suggest from solr: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read suggest response: %w", err)
	}

	return parseSuggestions(body)
}

// parseSuggestions extracts suggestions from a Solr suggest response.
func parseSuggestions(body []byte) ([]Suggestion, error) {
	var envelope struct {
		Suggest map[string]map[string]struct {
			Suggestions []struct {
				Term   string `json:"term"`
				Weight int    `json:"weight"`
			} `json:"suggestions"`
		} `json:"suggest"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("parse suggest response: %w", err)
	}

	var result []Suggestion
	for _, dict := range envelope.Suggest {
		for _, entry := range dict {
			for _, s := range entry.Suggestions {
				result = append(result, Suggestion{
					Term:   s.Term,
					Weight: s.Weight,
				})
			}
		}
	}
	return result, nil
}

// MoreLikeThis returns promotions similar to the document identified by docID.
// It uses Solr's MLT (More Like This) query parser on content fields.
func (s *Store) MoreLikeThis(ctx context.Context, docID string, limit int) ([]Promotion, error) {
	if limit <= 0 {
		limit = 5
	}
	if limit > 10 {
		limit = 10
	}

	docID = strings.TrimSpace(docID)
	if docID == "" {
		return nil, nil
	}

	params := url.Values{
		"q":                    {fmt.Sprintf("id:%q", docID)},
		"fl":                   {"*"},
		"wt":                   {"json"},
		"mlt":                  {"true"},
		"mlt.fl":               {"summary,tags,key_benefits,headline,analysis_value_proposition"},
		"mlt.mintf":            {"1"},
		"mlt.mindf":            {"1"},
		"mlt.count":            {fmt.Sprintf("%d", limit)},
		"mlt.interestingTerms": {"details"},
	}

	selectURL := fmt.Sprintf("%s/solr/%s/select?%s", s.baseURL, s.core, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, selectURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create mlt request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mlt from solr: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read mlt response: %w", err)
	}

	return parseMLTDocs(body)
}

// parseMLTDocs extracts the More Like This results from a Solr MLT response.
func parseMLTDocs(body []byte) ([]Promotion, error) {
	// Solr MLT component returns moreLikeThis → {docID} → {docs}
	var envelope struct {
		MoreLikeThis map[string]struct {
			Docs []map[string]interface{} `json:"docs"`
		} `json:"moreLikeThis"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("parse mlt response: %w", err)
	}

	var allDocs []map[string]interface{}
	for _, group := range envelope.MoreLikeThis {
		allDocs = append(allDocs, group.Docs...)
	}
	if len(allDocs) == 0 {
		return nil, nil
	}

	// Reuse parseSolrDocs logic by re-wrapping into the standard response format
	wrapped := struct {
		Response struct {
			Docs []map[string]interface{} `json:"docs"`
		} `json:"response"`
	}{}
	wrapped.Response.Docs = allDocs
	wrappedBytes, err := json.Marshal(wrapped)
	if err != nil {
		return nil, fmt.Errorf("rewrap mlt docs: %w", err)
	}
	return parseSolrDocs(wrappedBytes)
}

// --- helpers ---

// applyFilters adds Solr filter query (fq) parameters based on SearchOptions.
func applyFilters(params url.Values, opts SearchOptions) {
	for _, tag := range opts.Tags {
		params.Add("fq", fmt.Sprintf("tags:%q", tag))
	}
	if opts.Channel != "" {
		params.Add("fq", fmt.Sprintf("target_channel:%q", opts.Channel))
	}
	if opts.MinStars > 0 {
		params.Add("fq", fmt.Sprintf("stars:[%d TO *]", opts.MinStars))
	}
}

// solrSort returns the Solr sort clause for a SearchOptions.Sort value.
// For Search queries (hasScore=true), default is "score desc".
// For List queries (hasScore=false), default is "created_at desc".
func solrSort(sort string, hasScore bool) string {
	switch sort {
	case "newest":
		return "created_at desc"
	case "stars":
		return "stars desc"
	case "views":
		return "views_14d_total desc"
	default:
		if hasScore {
			return "score desc"
		}
		return "created_at desc"
	}
}

// parseFacets extracts facet counts from a Solr response body.
func parseFacets(body []byte) map[string][]Facet {
	var envelope struct {
		FacetCounts struct {
			FacetFields map[string][]interface{} `json:"facet_fields"`
		} `json:"facet_counts"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil
	}
	if len(envelope.FacetCounts.FacetFields) == 0 {
		return nil
	}

	facets := make(map[string][]Facet)
	for field, pairs := range envelope.FacetCounts.FacetFields {
		var items []Facet
		for i := 0; i+1 < len(pairs); i += 2 {
			name, _ := pairs[i].(string)
			count, _ := pairs[i+1].(float64)
			if name != "" && int(count) > 0 {
				items = append(items, Facet{Value: name, Count: int(count)})
			}
		}
		if len(items) > 0 {
			facets[field] = items
		}
	}
	return facets
}

// parseHighlights extracts highlighted snippets from a Solr response.
// Returns a map of document ID → field name → joined highlight HTML.
func parseHighlights(body []byte) map[string]map[string]string {
	var envelope struct {
		Highlighting map[string]map[string][]string `json:"highlighting"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil
	}
	if len(envelope.Highlighting) == 0 {
		return nil
	}

	result := make(map[string]map[string]string)
	for docID, fields := range envelope.Highlighting {
		fieldMap := make(map[string]string)
		for field, snippets := range fields {
			if len(snippets) > 0 {
				fieldMap[field] = strings.Join(snippets, " … ")
			}
		}
		if len(fieldMap) > 0 {
			result[docID] = fieldMap
		}
	}
	return result
}

// parseCollation extracts the best spell-check collation from a Solr response.
func parseCollation(body []byte) string {
	var envelope struct {
		Spellcheck struct {
			Collations []interface{} `json:"collations"`
		} `json:"spellcheck"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return ""
	}
	// Solr returns collations as alternating ["collation", "suggested query", ...]
	for i := 0; i+1 < len(envelope.Spellcheck.Collations); i += 2 {
		key, _ := envelope.Spellcheck.Collations[i].(string)
		if key == "collation" {
			if val, ok := envelope.Spellcheck.Collations[i+1].(string); ok {
				return val
			}
			// Could also be an object with "collationQuery" field
			if obj, ok := envelope.Spellcheck.Collations[i+1].(map[string]interface{}); ok {
				if cq, ok := obj["collationQuery"].(string); ok {
					return cq
				}
			}
		}
	}
	return ""
}

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
