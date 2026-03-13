# Prompt: Add Analysis Agent Env Vars to .env.example and README

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. I've just provisioned a second AI agent — the **Analysis Agent** — on DigitalOcean Gradient (see `docs/prompts/28-provision-analysis-agent.md`).

Now I need to document the two new environment variables so other developers (and hackathon judges) know they exist.

The full intent document is at `docs/intent-for-analysis-agent.md` — read it for high-level context.

## Current state

### `.env.example`

```env
# DigitalOcean Gradient AI Agent
# Get these from the DigitalOcean Gradient dashboard
AGENT_ENDPOINT=https://your-agent-id.agents.do-ai.run
AGENT_ACCESS_KEY=your-access-key-here

# HTTP server port (optional, default: 8080)
PORT=8080

# SQLite database path (optional, default: promotions.db)
DB_PATH=promotions.db
```

Note: `GITHUB_TOKEN` is missing from `.env.example` — it should already be there but isn't. Add it while you're here.

### README.md environment variables table (in "Getting Started" section)

```markdown
| Variable           | Required | Default          | Description                        |
|--------------------|----------|------------------|------------------------------------|
| `AGENT_ENDPOINT`   | Yes      | —                | URL of the AI agent service        |
| `AGENT_ACCESS_KEY` | Yes      | —                | API key for agent authentication   |
| `PORT`             | No       | `8080`           | HTTP server port                   |
| `DB_PATH`          | No       | `promotions.db`  | SQLite database file path          |
| `GITHUB_TOKEN`     | No       | —                | GitHub PAT for traffic metrics (views/clones) on AcaciaMan repos         |
```

## Your task

### 1. Update `.env.example`

Add the following entries to `.env.example`:

```env
# GitHub API token (optional — enables traffic metrics for AcaciaMan repos)
# GITHUB_TOKEN=ghp_your_token_here

# Analysis Agent (optional — enables repo analysis feature)
# Get these from the DigitalOcean Gradient dashboard for the analysis agent
# ANALYSIS_AGENT_ENDPOINT=https://your-analysis-agent-id.agents.do-ai.run
# ANALYSIS_AGENT_ACCESS_KEY=your-analysis-access-key-here
```

Comment them out (with `#`) since they're optional — this is the standard `.env.example` convention for optional vars. Place them after the existing `DB_PATH` line.

### 2. Update the README.md environment variables table

Add two new rows to the existing table in the "Getting Started > Environment Variables" section:

| Variable                     | Required | Default | Description                                                  |
|------------------------------|----------|---------|--------------------------------------------------------------|
| `ANALYSIS_AGENT_ENDPOINT`    | No       | —       | URL of the Analysis Agent service (enables repo analysis)    |
| `ANALYSIS_AGENT_ACCESS_KEY`  | No       | —       | API key for the Analysis Agent                               |

Place them after the `GITHUB_TOKEN` row. Both are **optional** — when not set, the analysis feature is silently disabled.

### 3. Add a brief note about the Analysis Agent in README.md

After the existing "GitHub Traffic Metrics" section and before the "Tech Stack" section, add a new section:

```markdown
## Repo Analysis (Optional)

The app can optionally call a second AI agent — the **Analysis Agent** — to produce a structured marketing analysis of each repository before generating promotional content. The analysis includes:

- **Primary value proposition** — what the repo helps users achieve.
- **Ideal audience** — who would benefit most.
- **Key features** — user-facing benefits.
- **Differentiators** — what makes it stand out.
- **Risks/limitations** — caveats like early-stage status or narrow scope.
- **Social proof signals** — interpretation of stars and traffic metrics.
- **Positioning angles** — suggested marketing angles.

When configured, this analysis is displayed in a "Why this repo?" panel on the Generate view and as compact summaries on Search result cards. It also feeds into the Promotion Agent to sharpen headlines, summaries, and CTAs.

### Setup

1. Create an Analysis Agent on the [DigitalOcean Gradient dashboard](https://cloud.digitalocean.com/gen-ai/agents) using the instructions in `docs/analysis-agent-model-instructions.md`.
2. Set the environment variables:

```bash
ANALYSIS_AGENT_ENDPOINT=https://your-analysis-agent-id.agents.do-ai.run
ANALYSIS_AGENT_ACCESS_KEY=your-analysis-access-key-here
```

### Behavior by scenario

| Scenario                              | Analysis behavior                              |
|---------------------------------------|------------------------------------------------|
| Both env vars set + agent responds    | Analysis generated, stored, displayed, fed to Promotion Agent |
| Both env vars set + agent fails       | Warning logged, promotion proceeds without analysis |
| Env vars not set                      | Feature disabled — no analysis call, no UI panel |
```

## What NOT to do

- Do NOT modify any Go source code (that's the next prompt).
- Do NOT change the Promotion Agent documentation.
- Do NOT add the analysis model instructions — they already exist at `docs/analysis-agent-model-instructions.md`.

## Verification

1. `.env.example` contains the new env vars (commented out).
2. `README.md` env table has the two new rows.
3. `README.md` has the "Repo Analysis (Optional)" section.
4. No existing content is broken or removed.
