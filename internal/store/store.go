package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS promotions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    repo_url TEXT NOT NULL,
    repo_name TEXT NOT NULL,
    headline TEXT NOT NULL,
    summary TEXT NOT NULL,
    key_benefits TEXT NOT NULL,
    tags TEXT NOT NULL,
    twitter_posts TEXT NOT NULL,
    linkedin_post TEXT NOT NULL,
    call_to_action TEXT NOT NULL,
    target_channel TEXT NOT NULL DEFAULT '',
    target_audience TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE VIRTUAL TABLE IF NOT EXISTS promotions_fts USING fts5(
    repo_name,
    headline,
    summary,
    tags,
    linkedin_post,
    call_to_action,
    content=promotions,
    content_rowid=id
);

-- Triggers to keep FTS index in sync.
CREATE TRIGGER IF NOT EXISTS promotions_ai AFTER INSERT ON promotions BEGIN
    INSERT INTO promotions_fts(rowid, repo_name, headline, summary, tags, linkedin_post, call_to_action)
    VALUES (new.id, new.repo_name, new.headline, new.summary, new.tags, new.linkedin_post, new.call_to_action);
END;

CREATE TRIGGER IF NOT EXISTS promotions_ad AFTER DELETE ON promotions BEGIN
    INSERT INTO promotions_fts(promotions_fts, rowid, repo_name, headline, summary, tags, linkedin_post, call_to_action)
    VALUES ('delete', old.id, old.repo_name, old.headline, old.summary, old.tags, old.linkedin_post, old.call_to_action);
END;

CREATE TRIGGER IF NOT EXISTS promotions_au AFTER UPDATE ON promotions BEGIN
    INSERT INTO promotions_fts(promotions_fts, rowid, repo_name, headline, summary, tags, linkedin_post, call_to_action)
    VALUES ('delete', old.id, old.repo_name, old.headline, old.summary, old.tags, old.linkedin_post, old.call_to_action);
    INSERT INTO promotions_fts(rowid, repo_name, headline, summary, tags, linkedin_post, call_to_action)
    VALUES (new.id, new.repo_name, new.headline, new.summary, new.tags, new.linkedin_post, new.call_to_action);
END;
`

// Promotion represents a stored promotional content record.
type Promotion struct {
	ID              int64     `json:"id"`
	RepoURL         string    `json:"repo_url"`
	RepoName        string    `json:"repo_name"`
	Headline        string    `json:"headline"`
	Summary         string    `json:"summary"`
	KeyBenefits     []string  `json:"key_benefits"`
	Tags            []string  `json:"tags"`
	TwitterPosts    []string  `json:"twitter_posts"`
	LinkedInPost    string    `json:"linkedin_post"`
	CallToAction    string    `json:"call_to_action"`
	TargetChannel   string    `json:"target_channel"`
	TargetAudience  string    `json:"target_audience"`
	CreatedAt       time.Time `json:"created_at"`
	Views14dTotal   int       `json:"views_14d_total"`
	Views14dUnique  int       `json:"views_14d_unique"`
	Clones14dTotal  int       `json:"clones_14d_total"`
	Clones14dUnique int       `json:"clones_14d_unique"`
}

// Store is a SQLite-backed store for promotional content.
type Store struct {
	db *sql.DB
}

// New opens (or creates) the SQLite database at dbPath and runs schema
// migration. The database is ready to use immediately after this returns.
func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("initialize schema: %w", err)
	}

	s := &Store{db: db}
	if err := s.applyMigrations(); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply migrations: %w", err)
	}

	return s, nil
}

func (s *Store) applyMigrations() error {
	columns := []string{
		"views_14d_total INTEGER NOT NULL DEFAULT 0",
		"views_14d_unique INTEGER NOT NULL DEFAULT 0",
		"clones_14d_total INTEGER NOT NULL DEFAULT 0",
		"clones_14d_unique INTEGER NOT NULL DEFAULT 0",
	}
	for _, col := range columns {
		_, err := s.db.Exec("ALTER TABLE promotions ADD COLUMN " + col)
		if err != nil && !strings.Contains(err.Error(), "duplicate column") {
			return fmt.Errorf("migration failed: %w", err)
		}
	}
	return nil
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// Save inserts a new promotion into the database. If a promotion for the same
// repo_url already exists, the old record is deleted first. On success it sets
// p.ID and p.CreatedAt from the inserted row.
func (s *Store) Save(ctx context.Context, p *Promotion) error {
	benefits, err := marshalJSON(p.KeyBenefits)
	if err != nil {
		return fmt.Errorf("marshal key_benefits: %w", err)
	}
	tags, err := marshalJSON(p.Tags)
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}
	tweets, err := marshalJSON(p.TwitterPosts)
	if err != nil {
		return fmt.Errorf("marshal twitter_posts: %w", err)
	}

	// Delete any existing promotion for the same repo URL.
	const delQ = `DELETE FROM promotions WHERE repo_url = ?`
	if _, err := s.db.ExecContext(ctx, delQ, p.RepoURL); err != nil {
		return fmt.Errorf("delete old promotion: %w", err)
	}

	const q = `INSERT INTO promotions
		(repo_url, repo_name, headline, summary, key_benefits, tags, twitter_posts,
		 linkedin_post, call_to_action, target_channel, target_audience,
		 views_14d_total, views_14d_unique, clones_14d_total, clones_14d_unique)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id, created_at`

	var createdAt string
	err = s.db.QueryRowContext(ctx, q,
		p.RepoURL, p.RepoName, p.Headline, p.Summary,
		benefits, tags, tweets,
		p.LinkedInPost, p.CallToAction,
		p.TargetChannel, p.TargetAudience,
		p.Views14dTotal, p.Views14dUnique, p.Clones14dTotal, p.Clones14dUnique,
	).Scan(&p.ID, &createdAt)
	if err != nil {
		return fmt.Errorf("insert promotion: %w", err)
	}

	p.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	return nil
}

// Search performs a full-text search across promotions and returns matching
// results ordered by relevance. If limit is 0 it defaults to 20.
func (s *Store) Search(ctx context.Context, query string, limit int) ([]Promotion, error) {
	if limit <= 0 {
		limit = 20
	}

	ftsQuery := sanitizeFTSQuery(query)
	if ftsQuery == "" {
		return []Promotion{}, nil
	}

	const q = `SELECT p.id, p.repo_url, p.repo_name, p.headline, p.summary,
		p.key_benefits, p.tags, p.twitter_posts, p.linkedin_post,
		p.call_to_action, p.target_channel, p.target_audience, p.created_at,
		p.views_14d_total, p.views_14d_unique, p.clones_14d_total, p.clones_14d_unique
		FROM promotions_fts fts
		JOIN promotions p ON p.id = fts.rowid
		WHERE promotions_fts MATCH ?
		ORDER BY rank
		LIMIT ?`

	rows, err := s.db.QueryContext(ctx, q, ftsQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("search promotions: %w", err)
	}
	defer rows.Close()

	return scanPromotions(rows)
}

// List returns the most recent promotions ordered by created_at descending.
// If limit is 0 it defaults to 20.
func (s *Store) List(ctx context.Context, limit int) ([]Promotion, error) {
	if limit <= 0 {
		limit = 20
	}

	const q = `SELECT id, repo_url, repo_name, headline, summary,
		key_benefits, tags, twitter_posts, linkedin_post,
		call_to_action, target_channel, target_audience, created_at,
		views_14d_total, views_14d_unique, clones_14d_total, clones_14d_unique
		FROM promotions
		ORDER BY created_at DESC
		LIMIT ?`

	rows, err := s.db.QueryContext(ctx, q, limit)
	if err != nil {
		return nil, fmt.Errorf("list promotions: %w", err)
	}
	defer rows.Close()

	return scanPromotions(rows)
}

// --- helpers ---

func scanPromotions(rows *sql.Rows) ([]Promotion, error) {
	var result []Promotion
	for rows.Next() {
		var p Promotion
		var benefits, tags, tweets, createdAt string
		if err := rows.Scan(
			&p.ID, &p.RepoURL, &p.RepoName, &p.Headline, &p.Summary,
			&benefits, &tags, &tweets, &p.LinkedInPost,
			&p.CallToAction, &p.TargetChannel, &p.TargetAudience, &createdAt,
			&p.Views14dTotal, &p.Views14dUnique, &p.Clones14dTotal, &p.Clones14dUnique,
		); err != nil {
			return nil, fmt.Errorf("scan promotion: %w", err)
		}
		p.KeyBenefits = unmarshalJSONOrEmpty(benefits)
		p.Tags = unmarshalJSONOrEmpty(tags)
		p.TwitterPosts = unmarshalJSONOrEmpty(tweets)
		p.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		result = append(result, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate promotions: %w", err)
	}
	if result == nil {
		result = []Promotion{}
	}
	return result, nil
}

func marshalJSON(v []string) (string, error) {
	if v == nil {
		v = []string{}
	}
	b, err := json.Marshal(v)
	return string(b), err
}

func unmarshalJSONOrEmpty(s string) []string {
	var v []string
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return []string{}
	}
	return v
}

// sanitizeFTSQuery wraps each whitespace-separated token in double quotes to
// prevent FTS5 syntax errors from user input.
func sanitizeFTSQuery(query string) string {
	// Strip characters that are special in FTS5 even inside quotes.
	replacer := strings.NewReplacer(
		"\"", "",
		"*", "",
		"(", "",
		")", "",
		":", "",
		"^", "",
	)
	cleaned := replacer.Replace(query)

	var tokens []string
	for _, t := range strings.Fields(cleaned) {
		if t != "" {
			tokens = append(tokens, "\""+t+"\"")
		}
	}
	return strings.Join(tokens, " ")
}
