# Prompt: Update README for Traffic Metrics Feature

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. I've completed Phases 1–3 which added GitHub traffic metrics (views & clones) for AcaciaMan repositories:

- **Phase 1:** Backend — token support, traffic API client, schema migration, handler integration.
- **Phase 2:** UI — traffic block on generate result cards, compact + expanded metrics on search cards.
- **Phase 3:** Agent — traffic fields in `RepoMetrics`, prompt tone rules, handler rewired to fetch before agent call.

Now I need to update `README.md` to document this feature for hackathon judges and other developers.

The full intent document is at `docs/intent-for-views-clones.md`.

## Current README (`README.md`)

The README has these sections:
1. Title + one-line description
2. Features (bullet list)
3. Architecture (ASCII diagram)
4. API Endpoints (table + request/response examples)
5. Generated Content (field table)
6. Project Structure (file tree)
7. Getting Started (prerequisites, env vars table, run command)
8. Tech Stack

The Environment Variables table currently has:

| Variable           | Required | Default          | Description                        |
|--------------------|----------|------------------|------------------------------------|
| `AGENT_ENDPOINT`   | Yes      | —                | URL of the AI agent service        |
| `AGENT_ACCESS_KEY` | Yes      | —                | API key for agent authentication   |
| `PORT`             | No       | `8080`           | HTTP server port                   |
| `DB_PATH`          | No       | `promotions.db`  | SQLite database file path          |

## Your task

Update `README.md` with the following changes:

### 1. Add a feature bullet

In the **Features** section, add a new bullet after the GitHub metadata bullet:

```markdown
- **GitHub traffic metrics** — Optionally fetches 14-day views and clones for AcaciaMan repositories, displays them in the UI, and uses them to influence AI-generated promotional tone
```

### 2. Add `GITHUB_TOKEN` to the environment variables table

Add a row to the existing env vars table:

| Variable           | Required | Default          | Description                                                              |
|--------------------|----------|------------------|--------------------------------------------------------------------------|
| `GITHUB_TOKEN`     | No       | —                | GitHub PAT for traffic metrics (views/clones) on AcaciaMan repos         |

### 3. Add a new section: "GitHub Traffic Metrics"

Insert a new section **after** "Getting Started" and **before** "Tech Stack". Use this content:

```markdown
## GitHub Traffic Metrics (Views & Clones)

The app can optionally fetch GitHub **traffic metrics** (14-day views and clones) for repositories owned by the **AcaciaMan** account. This feature requires a `GITHUB_TOKEN` environment variable.

### How it works

1. When generating promotional content for an AcaciaMan repo, the backend fetches traffic data from the GitHub API:
   - `GET /repos/{owner}/{repo}/traffic/views`
   - `GET /repos/{owner}/{repo}/traffic/clones`
2. The metrics (total/unique views and clones over the last 14 days) are:
   - **Stored** alongside the promotion in the SQLite database.
   - **Displayed** in the web UI on both generate results and search cards.
   - **Sent to the AI agent** as context, subtly influencing the promotional tone (e.g., "gaining traction" for repos with active traffic).

### Token setup

Create a GitHub Personal Access Token (classic or fine-grained) with the following permissions:

- **Classic PAT:** `repo` scope (includes traffic access for your repos).
- **Fine-grained PAT:** Repository permissions → Administration → Read-only (for traffic endpoints on your own repos).

Set it in your environment or `.env` file:

```bash
GITHUB_TOKEN=ghp_your_token_here
```

### Behavior by scenario

| Scenario                          | Traffic metrics behavior                     |
|-----------------------------------|----------------------------------------------|
| AcaciaMan repo + token set        | Metrics fetched, stored, displayed, sent to AI |
| AcaciaMan repo + no token         | No metrics — all traffic fields are 0         |
| Non-AcaciaMan repo (any token)    | No metrics — feature only activates for AcaciaMan |
| Token set + GitHub API error      | Warning logged, generation continues without metrics |
| Token set + rate limited          | Warning logged, generation continues without metrics |

### Limitations

- GitHub only provides traffic data for the **last 14 days**.
- Traffic endpoints require **push access** to the repository — the token must belong to the repo owner or a collaborator.
- Traffic metrics are fetched per-generation, not cached. Rapid re-generation of the same repo will make multiple API calls.
```

### 4. Update the API response documentation

In the **POST /api/generate** section, update the response description to mention traffic fields:

```markdown
**Response:** A `Promotion` object containing `id`, `headline`, `summary`, `key_benefits`, `tags`, `twitter_posts`, `linkedin_post`, `call_to_action`, `created_at`, and optionally `views_14d_total`, `views_14d_unique`, `clones_14d_total`, `clones_14d_unique` (for AcaciaMan repos with a configured token).
```

### 5. Update the Generated Content table

Add a row note after the table or extend it:

```markdown
Additionally, for AcaciaMan repositories with a configured `GITHUB_TOKEN`, the response includes:

| Field               | Description                              |
|---------------------|------------------------------------------|
| `views_14d_total`   | Total page views in the last 14 days     |
| `views_14d_unique`  | Unique visitors in the last 14 days      |
| `clones_14d_total`  | Total clones in the last 14 days         |
| `clones_14d_unique` | Unique cloners in the last 14 days       |
```

### 6. Update the architecture diagram

Add a note about the GitHub token connection. Update the GitHub API box to show it handles both public metadata and authenticated traffic data:

```
                    ┌──────────────┐
                    │  GitHub API  │
                    │ (repo data + │
                    │  traffic)    │
                    └──────┬───────┘
```

## What NOT to do

- Do NOT modify any Go code, HTML, or other source files.
- Do NOT remove or rewrite existing README sections — only add to them.
- Do NOT add developer/internal notes about the phased implementation approach (keep README user-facing).
- Do NOT add setup instructions for non-AcaciaMan users (out of scope for this iteration).

## Verification

After implementation:

1. The README reads well from top to bottom as a coherent document.
2. A hackathon judge reading it understands:
   - What the app does.
   - That traffic metrics are an optional, scoped feature.
   - How to set up `GITHUB_TOKEN`.
   - What happens in each scenario (token/no token, AcaciaMan/other).
3. The env vars table includes `GITHUB_TOKEN`.
4. The architecture diagram reflects the traffic data flow.
5. No broken markdown formatting (tables render correctly, code blocks are closed).
