# Input JSON Contract — v1 (Phase 1)

This document defines the structured repo data embedded in the `messages[].content` field of the Gradient agent chat completion request.

---

## 1. Final Input JSON Schema

| Field              | Type       | Required | Description                                                      |
|--------------------|------------|----------|------------------------------------------------------------------|
| `repo_url`         | `string`   | yes      | Full GitHub URL of the repository.                               |
| `repo_name`        | `string`   | yes      | Human-readable project name.                                     |
| `short_description`| `string`   | yes      | One-line project description (from GitHub "About").              |
| `readme_summary`   | `string`   | yes      | Condensed README content. Max 500 characters.                    |
| `primary_language` | `string`   | no       | Main programming language of the repo (e.g., `"Go"`).           |
| `topics`           | `[]string` | no       | GitHub topics/tags. Empty array if none.                         |
| `metrics`          | `object`   | yes      | Basic repo popularity metrics (see sub-fields below).            |
| `metrics.stars`    | `int`      | yes      | GitHub star count.                                               |
| `metrics.forks`    | `int`      | yes      | Fork count.                                                      |
| `metrics.watchers` | `int`      | yes      | Watcher count.                                                   |
| `metrics.open_issues` | `int`   | yes      | Open issue count.                                                |
| `target_channel`   | `string`   | no       | Primary channel hint: `"twitter"`, `"linkedin"`, or `"all"`. Default: `"all"`. Agent always generates all output fields; this biases tone and emphasis. |
| `target_audience`  | `string`   | no       | Free-text description of the intended audience.                  |

---

## 2. Hardcoded Sample (Village Square)

```json
{
  "repo_url": "https://github.com/AcaciaMan/village-square",
  "repo_name": "Village Square",
  "short_description": "Digital village square for local announcements, garage sales, and connections between neighbors and suppliers.",
  "readme_summary": "A community web app for a rural village — connecting villagers, local producers (fishermen, farmers, crafters), and the yearly Village Day celebration. Built with Go and designed for simplicity.",
  "primary_language": "Go",
  "topics": ["go", "community", "local"],
  "metrics": {
    "stars": 12,
    "forks": 3,
    "watchers": 5,
    "open_issues": 2
  },
  "target_channel": "twitter",
  "target_audience": "Villagers and small-community organizers"
}
```

---

## 3. Rationale for Each Decision

### Q1 — Metrics realism
**Decision:** Use realistic small-project numbers (12 stars, 3 forks, 5 watchers, 2 open issues).

120k stars for a village app would produce misleading promo copy ("Join 120,000 developers!"). Realistic metrics let the agent generate honest, representative content. The agent can still write compelling copy for a small project — that's a better test of its capability.

### Q2 — `recent_clones` field
**Decision:** Dropped entirely.

The GitHub Traffic API requires push access to the repo, which won't be available for arbitrary public repos in Phase 2. The field adds integration complexity for marginal output quality improvement. If clones data becomes available later, it can be added as an optional field inside `metrics` without breaking changes.

### Q3 — `target_channel` semantics
**Decision:** Keep as an optional string hint. The agent always generates all output fields (`twitter_posts`, `linkedin_post`, etc.) regardless of this value. When set to `"twitter"`, the agent should optimize tone and emphasis for Twitter but still produce LinkedIn content. Default is `"all"` (no bias).

This avoids the complexity of conditional output schemas while still giving the user a way to signal intent.

### Q4 — `target_audience` free text vs enum
**Decision:** Free text.

An enum would be premature — we don't know what audiences users will care about. Free text gives maximum flexibility at zero maintenance cost. The agent handles natural language well, so `"Villagers and small-community organizers"` works just as well as a predefined enum value. Can constrain later if patterns emerge.

### Q5 — `readme_summary` length
**Decision:** Max 500 characters.

500 chars is roughly 75–100 words — enough for a meaningful summary, short enough to avoid blowing up the prompt context. The Go backend should truncate at 500 chars when building the input. This cap also prevents accidental injection of entire multi-page READMEs.

### Q6 — Language/locale field
**Decision:** Omitted for Phase 1.

Internationalization is out of scope for an MVP. All output will be in English. A `language` field can be added later as an optional string without breaking changes.

### Q7 — Missing fields
**Decision:** Added `primary_language` only.

- **`primary_language`** (added): Cheap to include and directly improves output quality — the agent can tailor content to the right developer audience ("Built with Go", "A Python library for...").
- **`license`**: Omitted. Rarely affects promotional tone. Can add later.
- **`last_commit_date`**: Omitted. Useful for "actively maintained" messaging but not worth the complexity in Phase 1. Can add later as optional.

Every field in the final schema either directly improves agent output quality or is trivial to populate from the GitHub API in Phase 2.
