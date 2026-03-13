# Intent: Add Views & Clones Metrics to Repo Promoter Agent

This document describes the planned feature to integrate GitHub **views** and **clones** traffic metrics into the repo‑promoter app, and how to implement it in phases.

## High‑Level Goal

Extend the existing DigitalOcean Gradient AI hackathon project so that:

- For **AcaciaMan GitHub repositories**, the app fetches GitHub traffic metrics (views and clones).
- These metrics are **stored in the local SQLite database**, shown in the **UI** alongside promotion materials, and used to **influence the tone** of generated copy.
- The **README.md** is updated to describe the new behavior and configuration.

This document is an implementation guide for planning and executing the changes in **small, testable phases**.

***

## Functional Requirements

1. **Scope: AcaciaMan repositories only**
   - The feature should only activate when the user provides a URL to a **GitHub repository owned by the `AcaciaMan` account**.
   - For non‑AcaciaMan repos, the system should behave as today (no traffic metrics, no traffic‑based tone adjustments).

2. **GitHub token from environment**
   - A **GitHub access token** (fine‑grained or PAT) will be provided via an environment variable (e.g. `GITHUB_TOKEN`).
   - The backend must:
     - Read the token on startup (or lazily on first use).
     - Use it for authenticated calls to the GitHub REST API traffic endpoints.
     - Never expose the token to the frontend.

3. **Fetch views and clones**
   - For eligible repos (AcaciaMan + token available), the backend should call the GitHub **traffic** endpoints:
     - Views: `/repos/{owner}/{repo}/traffic/views`
     - Clones: `/repos/{owner}/{repo}/traffic/clones`
   - Minimal metrics to derive:
     - `total_views_14d`
     - `unique_views_14d`
     - `total_clones_14d`
     - `unique_clones_14d`
   - Optionally keep the per‑day breakdown in memory or a separate table if needed later, but **MVP can just store aggregates**.

4. **Store metrics in SQLite**
   - Extend the existing SQLite schema to include traffic metrics for each promotion record (or a related table keyed by repo URL).
   - When a new promotion is generated for a repo:
     - Fetch metrics.
     - Store them together with the promotion content.
   - When searching/browsing promotions:
     - Include metrics in the returned JSON so they can be displayed in the UI.

5. **Show metrics in the UI**
   - On the **Generate** result view:
     - Display small metrics badges or a section like:
       - “Views (last 14 days): total / unique”
       - “Clones (last 14 days): total / unique”
   - On **Search results**:
     - Optionally show a compact version of the same metrics for each repo card (e.g. in a footer or tooltip).
   - Make sure the UI clearly handles the case where metrics are **not available** (no token, not AcaciaMan repo, or GitHub API error).

6. **Send metrics to the AI agent**
   - Extend the agent input structure (the `metrics` field already mentioned in the hackathon doc) to include:
     - `stars`, `forks`, `watchers` (existing or soon‑to‑be implemented).
     - `views_14d_total`, `views_14d_unique`.
     - `clones_14d_total`, `clones_14d_unique`.
   - Update the prompt / system instructions so the agent:
     - Treats higher views/clones as a signal that the repo is attracting attention.
     - Can use wording like “actively discovered project”, “getting regular traffic”, etc., when metrics are non‑zero.
     - Stays conservative and neutral if metrics are low or zero (avoid over‑hyping).

7. **Update README.md**
   - Add a dedicated section explaining:
     - That the app can optionally fetch GitHub traffic metrics (views/clones) for **AcaciaMan** repos only (for now).
     - How to configure `GITHUB_TOKEN` and what scopes/permissions it needs.
     - How these metrics are used:
       - Displayed in the UI.
       - Used as part of the context for AI‑generated promotional content.
     - What happens when:
       - No token is set.
       - The repo is not owned by AcaciaMan.
       - The GitHub API rate‑limits or errors.

***

## Non‑Goals (for this iteration)

- No support yet for:
  - Arbitrary third‑party users authenticating with their own GitHub accounts.
  - Installing a GitHub App or OAuth flows.
  - Complex analytics dashboards beyond basic aggregated numbers.
- No need to persist full daily time series in this phase (only aggregates required).
- No separate “metrics only” API; metrics are surfaced via existing generate/search flows.

***

## Phase Plan

### Phase 1 – Backend plumbing (no UI, no agent changes)

Objective: Add the ability to detect AcaciaMan repos, read `GITHUB_TOKEN`, and fetch/store traffic metrics.

Tasks:

1. **Repo owner detection**
   - Implement a small helper that, given a `repo_url`, extracts `{owner, repo}`.
   - Add a check: `owner == "AcaciaMan"` to gate traffic metrics logic.

2. **Read GitHub token from env**
   - Add config/env parsing for `GITHUB_TOKEN`.
   - Wire it into the existing GitHub client (e.g., set `Authorization: Bearer <token>` and required API headers).
   - If no token is set, log a warning and disable traffic calls.

3. **Traffic client functions**
   - Implement functions in the GitHub client module, e.g.:
     - `GetRepoViews(owner, repo string) (ViewsMetrics, error)`
     - `GetRepoClones(owner, repo string) (ClonesMetrics, error)`
   - Define small structs for the aggregated metrics needed for storage.

4. **Schema changes**
   - Extend SQLite schema (migration step) to store:
     - `views_14d_total`
     - `views_14d_unique`
     - `clones_14d_total`
     - `clones_14d_unique`
   - Wire this into the store layer so that when saving a promotion, metrics can be saved as well.

5. **Generate flow integration (backend only)**
   - In the `POST /api/generate` handler:
     - Detect if repo is AcaciaMan + token present.
     - If yes, fetch views/clones and pass them into the store along with the promotion.
   - For now, do not send metrics to the agent and do not change response JSON.

Deliverable: Traffic metrics are fetched and stored in the DB for AcaciaMan repos, but not yet visible in UI or AI output.

***

### Phase 2 – Expose metrics in API and UI

Objective: Make metrics visible to the frontend and shown in both Generate and Search views.

Tasks:

1. **API response extension**
   - Extend the `Promotion` JSON schema returned by:
     - `POST /api/generate`
     - `GET /api/search`
   - Add fields for:
     - `views_14d_total`
     - `views_14d_unique`
     - `clones_14d_total`
     - `clones_14d_unique`

2. **Generate page UI**
   - Update the frontend rendering of the generation result to show a small **“Traffic (last 14 days)”** block.
   - Show a clear “Not available” state (e.g., when metrics are `null` or `0` because no token or not an AcaciaMan repo).

3. **Search page UI**
   - Update search result cards to include a compact representation of traffic metrics (e.g., icons + short numbers).
   - Ensure layout remains clean and responsive.

4. **Graceful fallback**
   - Verify that the UI still works when:
     - Metrics are missing.
     - Backend returns older records without metrics (backward compatibility after schema changes).

Deliverable: Users can see views/clones metrics in the UI for promotions associated with AcaciaMan repos.

***

### Phase 3 – Agent integration and tone adjustment

Objective: Use traffic metrics as part of the AI input so they can influence promotional tone.

Tasks:

1. **Extend agent input schema**
   - Update the JSON payload sent to the Gradient agent to include:
     - `metrics.views_14d_total`
     - `metrics.views_14d_unique`
     - `metrics.clones_14d_total`
     - `metrics.clones_14d_unique`
   - Ensure this is optional and safely omitted when metrics are unavailable.

2. **Prompt updates**
   - Update the system / tool instructions for the agent to say (in natural language):
     - If traffic metrics are present and non‑zero, you may describe the project as being actively discovered or getting regular attention.
     - Do not fabricate numbers; only reflect the relative level (low/medium/high) implied by the metrics.
     - If metrics are missing or very low, keep tone neutral and avoid claims about popularity.

3. **Testing scenarios**
   - Create test cases for:
     - High‑traffic repo example.
     - Low‑traffic repo example.
     - No‑metrics repo (non‑AcaciaMan or no token).
   - Verify that the generated promotional copy adjusts tone accordingly but stays truthful and consistent.

Deliverable: AI‑generated promotional content subtly incorporates traffic metrics into its tone.

***

### Phase 4 – Documentation updates (README.md and developer notes)

Objective: Document configuration and behavior so other developers and hackathon judges understand the feature.

Tasks:

1. **README.md updates**
   - Add a section like “GitHub Traffic Metrics (Views & Clones)” explaining:
     - The feature is currently limited to AcaciaMan repositories.
     - How to set `GITHUB_TOKEN`.
     - What permissions are required for the token.
     - How the metrics are used in UI and AI.
   - Mention any limitations:
     - 14‑day window.
     - Only for repos where the token has access.
     - Possible API rate limiting.

2. **Developer notes (optional)**
   - Add a short explanation in a dev‑oriented doc (or code comments) that:
     - Phase 1–3 structure is intentional.
     - There is room to expand to multi‑user support using GitHub Apps in future iterations.

Deliverable: README and any helper docs fully describe the traffic metrics behavior and setup.

***

## Implementation Constraints and Notes

- Keep changes incremental and safe:
  - Start with backend + DB, then API + UI, then agent prompt changes.
- Preserve existing behavior:
  - If there is no `GITHUB_TOKEN` or repo is not owned by AcaciaMan, everything should work exactly as before, just without traffic metrics.
- Reliability:
  - If GitHub traffic API calls fail, log the error and continue without metrics rather than failing the entire generation flow.
- Hackathon focus:
  - Prioritize a robust, demonstrable flow for **AcaciaMan repos** over generic multi‑user support.

***