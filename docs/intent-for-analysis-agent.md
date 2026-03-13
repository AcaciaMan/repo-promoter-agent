# Intent: Automatic Repo Analysis Agent + Frontend Display

## High-level goal

Extend the existing **repo-promoter-agent** so that every generated promotion also includes a structured “analysis” of the repository’s key selling points, and expose this analysis in the frontend (e.g., a “Why this repo?” panel on generate + search views).

The analysis is produced by a dedicated **Analysis Agent** (Gradient), called automatically for every repo before the Promotion Agent, and its JSON output is stored alongside promotions in SQLite.

***

## Current context (summary for you, Claude)

- Backend: Go HTTP server with endpoints `/api/generate` and `/api/search`.
- Data sources: GitHub API for repo metadata, README, optional traffic metrics (14-day views/clones for AcaciaMan). 
- AI: A single Promotion Agent on Gradient that accepts structured repo data and returns promotional JSON (headline, summary, key_benefits, tags, tweets, LinkedIn post, CTA). 
- Storage: SQLite with FTS5, storing repo metadata and generated promotional content, queried via `/api/search`. 
- Frontend: Single-page HTML UI that lets the user paste a repo URL, trigger generation, and browse/search past promotions. 

You will extend this with a second agent and some schema/UI adjustments, implemented in **small, verifiable phases**.

***

## Desired agent responsibilities

### Analysis Agent (new)

**Role**

Act as a **product marketing analyst for GitHub repositories**. 
Given structured repo input (metadata, README, topics, metrics, optional target audience), produce a concise JSON description of the repo’s key selling points, positioning angles, and caveats. 

**Input JSON (from backend)**

```json
{
  "repo_url": "https://github.com/owner/repo",
  "repo_name": "repo-name",
  "short_description": "short GitHub description",
  "readme_text": "full or truncated README content",
  "topics": ["go", "cli", "testing"],
  "metrics": {
    "stars": 123,
    "forks": 10,
    "watchers": 5,
    "views_14d_total": 250,
    "views_14d_unique": 150,
    "clones_14d_total": 40,
    "clones_14d_unique": 25
  },
  "target_audience": "optional audience description"
}
```

You can reuse existing GitHub data-fetch logic and traffic metrics plumbing already in the project. 

**Output JSON schema (strict)**

```json
{
  "repo_url": "string",
  "repo_name": "string",
  "primary_value_proposition": "One sentence explaining what this repo helps users achieve.",
  "ideal_audience": [
    "Short description of audience segment 1",
    "Short description of audience segment 2"
  ],
  "key_features": [
    "Feature written as a user-facing benefit, not just a technical detail",
    "Another feature as a clear benefit"
  ],
  "differentiators": [
    "What makes this repo special vs. typical alternatives, based only on the input"
  ],
  "risk_or_limitations": [
    "Important caveats such as early-stage status, limited docs, or narrow scope; say \"none clearly indicated\" if not obvious"
  ],
  "social_proof_signals": [
    "Interpret stars/traffic concisely, e.g. \"early-stage project with modest traction\" or \"actively visited in the last 14 days\""
  ],
  "recommended_positioning_angle": [
    "A suggested marketing angle, e.g. \"time-saver for busy maintainers\"",
    "Another possible angle if applicable"
  ]
}
```

**Style and constraints (for the agent/system prompt)**

- Base every statement strictly on the provided input; do not invent features, integrations, or metrics. 
- If something is uncertain (e.g., docs quality), say that it is unclear rather than guessing.  
- Use concise, **developer-friendly** language; avoid generic marketing buzzwords.  
- Keep each string item to **1–2 short sentences**.  
- Output **only** a valid JSON object matching the schema, with no surrounding text or Markdown. 

***

### Promotion Agent (existing, with extended input)

The existing Promotion Agent already produces: `headline`, `summary`, `key_benefits`, `tags`, `twitter_posts`, `linkedin_post`, `call_to_action`, and optionally traffic fields. 

Extend its **input**, not its public API, by adding an `analysis` field that contains the Analysis Agent’s output:

```json
{
  "repo_url": "...",
  "repo_name": "...",
  "short_description": "...",
  "readme_summary": "...",
  "topics": ["..."],
  "metrics": { ... },
  "target_channel": "twitter|linkedin|general",
  "target_audience": "optional",
  "analysis": { /* full Analysis Agent JSON output */ }
}
```

Add to its system prompt something like:

- “You may receive an `analysis` object summarizing value proposition, ideal audience, differentiators, and limitations. Use it to sharpen headlines, summaries, and CTAs, but do not contradict the raw repo data. In case of conflict, prefer the raw repo data.” 

The **HTTP API response shape to the browser** should remain the same, but we’ll additionally include a serialized `analysis` block so the UI can render it (see below).

***

## Desired UX changes

- On the **Generate** view (current single-page UI), below the promotional content, show a “Why this repo?” panel that displays selected fields from the analysis JSON:
  - Primary value proposition (1 sentence)
  - Ideal audience (list)
  - Key features (list)
  - Differentiators
  - Risk/limitations (if any)
  - Social proof signals  
- On the **Search** view, in each card for a promotion, show a compact summary from analysis, for example:
  - One-line primary value proposition.
  - Maybe one bullet of “Who it’s for” or “What makes it special”. 

The UI should treat this as read-only informational content, with no extra user interaction beyond copying text where appropriate.

***

## Data model and API expectations

### Backend / DB

- Extend the SQLite schema to store the analysis for each promotion. Options:
  - Add an `analysis_json` TEXT column to the `promotions` table, storing the full JSON. 
  - Or create a separate `promotion_analysis` table keyed by `promotion_id`.  

Prefer the **simplest** approach that gives you:

- Fast retrieval for `/api/generate` (return the freshly created analysis along with promotion). 
- Reasonable access for `/api/search` (either join or secondary lookup by `promotion_id`). 

### Public API changes

For `/api/generate`:

- Keep all existing fields. 
- Add a new field, e.g.:

```json
"analysis": {
  "...": "full analysis JSON as defined above"
}
```

For `/api/search`:

- When returning promotions, also include their `analysis` (same shape) if present.
- If analysis does not exist (e.g., legacy rows), return `analysis: null` and let UI degrade gracefully.

***

## Implementation phases (for you, Claude)

Please work in these **phases**, verifying at each step.

### Phase 0 – Provision the Analysis Agent on Gradient and decide env vars

Before any code, set up the infrastructure:

1. **Create the Analysis Agent** on DigitalOcean Gradient:
   - Use the system prompt from `docs/analysis-agent-model-instructions.md`.
   - Note the resulting agent endpoint and access key.
2. **Decide the env var strategy.** The existing codebase uses a single `AGENT_ENDPOINT` + `AGENT_ACCESS_KEY` pair for the Promotion Agent. Two options:
   - **(a) Separate env vars** — `ANALYSIS_AGENT_ENDPOINT` + `ANALYSIS_AGENT_ACCESS_KEY`. Clearest separation; the Analysis Agent can live on a different Gradient agent or model.
   - **(b) Shared endpoint** — reuse `AGENT_ENDPOINT` / `AGENT_ACCESS_KEY` if Gradient supports routing to different agents by system prompt alone (less likely).
   - **Decision: use option (a)** — separate env vars. The Analysis Agent is a distinct Gradient agent with its own credentials. If they happen to share the same endpoint, the user just sets both to the same value.
3. Add the new env vars to `.env.example` (or equivalent) and document them in the README.
4. In `cmd/server/main.go`, the Analysis Agent env vars should be **optional**. If they are not set, the analysis feature is silently disabled (no analysis call, `analysis: null` in responses). This preserves backward compatibility.

Deliverables:

- A live Analysis Agent on Gradient with the system prompt applied.
- Updated `.env.example` and README documenting `ANALYSIS_AGENT_ENDPOINT` and `ANALYSIS_AGENT_ACCESS_KEY`.
- `main.go` loads the new env vars but does not fail if they are absent.

### Phase 1 – Define Analysis Agent contract and backend client

1. Create Go types representing the Analysis Agent **input** and **output** JSON schemas.
2. Add a new analysis client in `internal/agent` (e.g., `analysis_client.go`) or extend the existing client in a clean way:
   - Function: `CallAnalysisAgent(ctx, input) (AnalysisOutput, error)`.
   - Use `ANALYSIS_AGENT_ENDPOINT` / `ANALYSIS_AGENT_ACCESS_KEY` (from Phase 0).
   - Use a **30-second timeout** (shorter than the promotion agent's 60 s) to fail fast — analysis is supplementary and should not dominate total latency.
3. Add **structured logging** (timing, success/failure, input/output sizes) to the analysis client call. This is essential for prompt debugging in later phases.
4. For now, **do not** wire it to HTTP handlers; just ensure the client can be unit-tested with a mocked HTTP server.
5. Include test cases for **edge-case inputs**:
   - Repos with no README (`readme_text` is empty string).
   - Repos with no topics (empty array).
   - Agent returning invalid JSON.
   - Agent timeout (context cancelled).

Deliverables:

- New Go structs/types and client function.
- System prompt text for the Analysis Agent embedded or referenced clearly.
- Unit tests against a fake HTTP server validating JSON contract, error handling, and edge cases.

### Phase 2 – Schema, model, and persistence for analysis

Combine model changes and DB schema changes in one phase to avoid an awkward gap where analysis data is generated but silently dropped on save.

1. Extend the `Promotion` struct to include an `AnalysisJSON` field (`json.RawMessage` or `*string`), representing the full serialized analysis output.
2. Modify `internal/store/store.go`:
   - Add an `analysis_json TEXT DEFAULT NULL` column to the `promotions` table via migration (same `ALTER TABLE ADD COLUMN` pattern used for traffic metrics).
   - Update `Save()` to persist `analysis_json`.
   - Update all retrieval paths (`Search()`, `List()`, and any `scanPromotion` helper) to load `analysis_json`.
3. Ensure FTS5 search behavior remains unchanged — do **not** index `analysis_json` in FTS.
4. Legacy rows without analysis will have `analysis_json = NULL`; this is the default and requires no backfill.

Deliverables:

- Schema migration code and updated store functions.
- Tests confirming promotions are still searchable, analysis is persisted and retrievable, and `NULL` analysis for legacy rows is handled.

### Phase 3 – Wire Analysis Agent into /api/generate flow

1. In `internal/handler/generate.go`, modify the generation flow:
   - Fetch repo metadata + README + metrics as before.
   - If the analysis client is configured (non-nil), construct Analysis Agent input and call it.
   - On success, pass the resulting `analysis` into the Promotion Agent input and store the serialized analysis JSON.
   - On failure, log a warning with timing info, proceed without `analysis`, and set `analysis: null` in the response (**fail-soft**).
   - If the analysis client is not configured (env vars missing), skip the call entirely.
2. Update the Promotion Agent input type to include an optional `analysis` field.
3. Return the `analysis` field in the `/api/generate` JSON response (full analysis object, or `null`).

**Latency note:** The analysis call is sequential (its output feeds the promotion agent), so total generation time roughly doubles. This is an inherent constraint. Phase 5 addresses the UX impact with progress indicators.

Deliverables:

- End-to-end generation path where `/api/generate` now calls both agents (when configured) and returns promotion + analysis JSON.
- Verification that generation still works when `ANALYSIS_AGENT_ENDPOINT` is not set (analysis feature disabled).
- Manual test instructions for a happy-path repo.

### Phase 4 – Expose analysis via /api/search

1. Update `/api/search` handler to include `analysis` in each returned promotion object. 
2. Confirm that:
   - Existing clients (frontend) still work, ignoring the new field.
   - New UI code can rely on `analysis` being present or `null`.

Deliverables:

- JSON responses from `/api/search` containing `analysis` for new promotions.
- Backward compatibility maintained.

### Phase 5 – Frontend "Why this repo?" panel + loading UX

1. **Loading states.** With two sequential agent calls, the user may wait 15–30 seconds. Update the generate UI to show:
   - A multi-step progress indicator or status text (e.g., "Analyzing repository…" → "Generating promotion…").
   - Or at minimum, an enhanced spinner/message that communicates work is happening.
2. **Generate view — "Why this repo?" panel.** Below the promotional content, render a bordered section:
   - Primary value proposition.
   - Ideal audience (list).
   - Key features (list).
   - Differentiators.
   - Risks/limitations.
   - Social proof signals.
   - Recommended positioning angles.
   - A small label: "AI-generated analysis" to set expectations.
3. **When `analysis` is `null`:** hide the "Why this repo?" panel entirely (do not show an empty box or error message).
4. **Search view.** In each promotion card, if `analysis` is present:
   - Show primary value proposition as a subtitle.
   - Optionally one line of "ideal audience" or "differentiator".
   - Keep cards compact, prioritizing scan-ability.
   - If `analysis` is `null`, show just the existing card content (graceful degradation).

Deliverables:

- Updated HTML/CSS/JS with the new panel, card snippets, and loading UX.
- Manual test instructions: paste a repo URL, confirm loading feedback is visible, analysis appears on Generate view and in Search cards.
- Test with analysis disabled (env vars unset) to confirm the panel hides cleanly.

### Phase 6 – Prompt and UX refinement

1. Tweak Analysis Agent system prompt based on observed output quality:
   - Reduce verbosity if outputs are too long.
   - Better handling of **sparse / low-signal repos** (no README, no topics, few stars) — prompt should acknowledge limited data rather than padding with generic statements.
   - Test with archived or unmaintained repos — ensure the agent doesn't mischaracterize activity level.
2. Optionally add:
   - Copy buttons for analysis text blocks (matching existing promo content UX).
   - Tooltips explaining individual analysis fields.

***

## Explicit non-goals (to keep scope manageable)

These items were considered and intentionally excluded from the initial implementation:

- **Analysis caching / deduplication.** Every `/api/generate` call currently deletes and replaces the previous promotion for the same `repo_url`. The same will apply to analysis. Reusing a cached analysis when the repo hasn't changed would halve latency on re-generation, but adds complexity around cache invalidation. **Defer** to a future iteration.
- **Standalone `/api/analyze` endpoint.** A dedicated endpoint for analysis-only (without promotion) could be useful for previews or debugging, but adds scope. **Defer** unless needed for debugging during development.
- **FTS indexing of analysis.** Analysis fields are not indexed in full-text search. This could be revisited if users want to search by value proposition or audience segments.

***

## Quality & robustness requirements

### Fail-soft behavior

The system must still work if the Analysis Agent:
- Times out (30-second client timeout).
- Returns invalid JSON.
- Returns partial or malformed data.
- Is not configured at all (env vars missing).

In all of these cases:
- Log the issue with timing and error details.
- Proceed to call the Promotion Agent without analysis.
- Return `analysis: null` to the frontend.
- The UI hides the "Why this repo?" panel (not an error state — just absent).

### Backward compatibility

- `/api/generate` and `/api/search` remain compatible with clients that ignore the `analysis` field.
- The analysis feature is **opt-in** via env vars: if `ANALYSIS_AGENT_ENDPOINT` is not set, the feature is completely disabled with no impact on existing behavior.

### Observability

- Log analysis agent calls with: duration (ms), success/failure, input size (bytes), output size (bytes).
- On failure, log the error category (timeout, invalid JSON, HTTP error, missing config).
- This logging is critical for Phase 6 prompt refinement — you need to see what the agent actually produces.

### Edge cases to handle

| Scenario | Expected behavior |
|----------|-------------------|
| Repo with no README | Analysis agent receives empty `readme_text`; prompt should handle gracefully |
| Repo with no topics | Analysis agent receives empty array; should still produce useful output |
| Very large README (>2000 chars) | Already truncated by GitHub client; no change needed |
| Archived / unmaintained repo | Analysis should reflect activity level from metrics, not assume "active" |
| Agent returns extra fields | Ignore unknown fields; parse only the expected schema |
| Agent returns fewer fields | Treat missing fields as empty/null; don't fail the whole flow |

***

## What I want from you, Claude

- Implement the above phases sequentially, keeping commits small and scoped by phase.
- At each phase, summarize:
  - What changed.
  - How to run and manually test it (commands, example curl or browser steps).
- Propose small prompt tweaks for the Analysis Agent once you see real outputs from a few different repositories.

***

If any part of this intent is ambiguous (especially around DB schema changes or env var naming for a second agent), please propose 1–2 concrete options and pick the simplest one that preserves current behavior by default.  