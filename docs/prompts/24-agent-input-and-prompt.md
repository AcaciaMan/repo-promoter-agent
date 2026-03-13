# Prompt: Extend Agent Input Schema and Prompt for Traffic Metrics

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. I'm working on **Phase 3** — making traffic metrics influence the AI-generated promotional tone.

Phases 1–2 are complete:
- The GitHub client fetches traffic metrics (views/clones) for AcaciaMan repos.
- Traffic metrics are stored in SQLite and shown in the UI.
- But the AI agent currently does **not** receive traffic metrics — they are only fetched and stored for display purposes.

The full intent document is at `docs/intent-for-views-clones.md`.

## Current agent input types (`internal/agent/client.go`)

```go
type RepoInput struct {
    RepoURL          string      `json:"repo_url"`
    RepoName         string      `json:"repo_name"`
    ShortDescription string      `json:"short_description"`
    ReadmeSummary    string      `json:"readme_summary"`
    PrimaryLanguage  string      `json:"primary_language,omitempty"`
    Topics           []string    `json:"topics,omitempty"`
    Metrics          RepoMetrics `json:"metrics"`
    TargetChannel    string      `json:"target_channel,omitempty"`
    TargetAudience   string      `json:"target_audience,omitempty"`
}

type RepoMetrics struct {
    Stars      int `json:"stars"`
    Forks      int `json:"forks"`
    Watchers   int `json:"watchers"`
    OpenIssues int `json:"open_issues"`
}
```

## Current prompt template (`internal/agent/client.go`)

```go
const promptTemplate = `Generate promotional content for this GitHub repository.

REPO DATA:
{{.RepoDataJSON}}

Return a JSON object with exactly this structure:
{{.OutputSchemaJSON}}

RULES:
- Output ONLY the JSON object. No markdown fences, no preamble, no explanation.
- Echo repo_url and repo_name exactly from the input.
- Generate exactly 3 twitter_posts, each ≤280 characters including hashtags and URL.
- Generate 3–5 key_benefits.
- Generate 5–8 tags, expanding on the input topics.
- linkedin_post should be 150–300 words with professional tone.
- If target_channel is "twitter", optimize tone for Twitter. If "linkedin", optimize for LinkedIn. Always populate all fields regardless.
- Stay faithful to the repo data. Do not invent features not described in the input.
- Tailor content to target_audience if provided.`
```

## Your task

Two changes in `internal/agent/client.go`:

### 1. Extend `RepoMetrics` with traffic fields

Add four new fields to the existing `RepoMetrics` struct:

```go
type RepoMetrics struct {
    Stars           int `json:"stars"`
    Forks           int `json:"forks"`
    Watchers        int `json:"watchers"`
    OpenIssues      int `json:"open_issues"`
    Views14dTotal   int `json:"views_14d_total,omitempty"`
    Views14dUnique  int `json:"views_14d_unique,omitempty"`
    Clones14dTotal  int `json:"clones_14d_total,omitempty"`
    Clones14dUnique int `json:"clones_14d_unique,omitempty"`
}
```

Use `omitempty` so these fields are excluded from the JSON when zero. This means:
- For AcaciaMan repos with traffic data: the agent sees `"views_14d_total": 42` etc.
- For other repos or when no token: the fields are simply absent from the JSON — the agent won't see them at all.

### 2. Update the prompt template with traffic tone guidance

Add a new section to the `RULES:` block in `promptTemplate`. Insert these rules after the existing rules:

```
- The metrics field may include views_14d_total, views_14d_unique, clones_14d_total, and clones_14d_unique (14-day traffic data). If these are present and non-zero:
  - You may describe the project as being actively discovered or attracting attention.
  - Reflect the relative level (low: <50 views, moderate: 50–200, high: >200) naturally in the tone — do not cite exact numbers.
  - Use phrases like "gaining traction", "actively explored by developers", "attracting growing interest" when appropriate.
- If traffic metrics are absent or zero, keep tone neutral regarding popularity. Do not make claims about the project being widely used or popular.
- Never fabricate or exaggerate traffic numbers. Only use the metrics to subtly adjust tone, not to make specific numerical claims.
```

### Important: agent output schema stays unchanged

The `outputSchema` constant must NOT change. Traffic metrics influence the *tone* of the generated text, not the *structure* of the output.

## What NOT to do

- Do NOT modify the generate handler, store, GitHub client, or frontend.
- Do NOT change the `outputSchema` constant.
- Do NOT change the `RepoInput` struct (only `RepoMetrics` changes).
- Do NOT change the `Generate` method or any other method in this file.

## Verification

After implementation:

1. `go build ./...` compiles without errors.
2. The `RepoMetrics` struct has 8 fields (4 existing + 4 new).
3. When `RepoMetrics` is marshaled with zero traffic values, the JSON does NOT include `views_14d_total` etc. (due to `omitempty`):
   ```json
   {"stars":12,"forks":3,"watchers":5,"open_issues":2}
   ```
4. When traffic values are set, they appear:
   ```json
   {"stars":12,"forks":3,"watchers":5,"open_issues":2,"views_14d_total":42,"views_14d_unique":15,"clones_14d_total":8,"clones_14d_unique":5}
   ```
5. The prompt template includes the new traffic tone rules.
