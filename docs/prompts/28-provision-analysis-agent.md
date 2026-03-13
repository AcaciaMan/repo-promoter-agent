# Prompt: Provision the Analysis Agent on DigitalOcean Gradient

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. The app already has a **Promotion Agent** on Gradient that generates promotional content for GitHub repos.

I'm now adding a **second agent** — an **Analysis Agent** — that produces a structured marketing analysis of a repo (value proposition, audience, differentiators, risks, etc.). This analysis feeds into the Promotion Agent to improve output quality.

The full intent document is at `docs/intent-for-analysis-agent.md` — read it for high-level context.

## Current state

- The existing Promotion Agent is configured on Gradient with the model instructions in `docs/promotion-agent-model-instructions.md`.
- The Analysis Agent's model instructions are already written in `docs/analysis-agent-model-instructions.md`.
- The backend currently uses two env vars for the Promotion Agent: `AGENT_ENDPOINT` and `AGENT_ACCESS_KEY`.

## Your task

This is a **manual setup task** on the DigitalOcean Gradient dashboard — no code changes.

### Step 1 — Create the Analysis Agent on Gradient

1. Go to the DigitalOcean Gradient dashboard (https://cloud.digitalocean.com/gen-ai/agents).
2. Create a **new agent** with the following settings:
   - **Name:** `repo-analysis-agent` (or similar descriptive name).
   - **Model:** Use the same model as the Promotion Agent (or a comparable general-purpose LLM available on Gradient).
   - **Model instructions:** Copy the full content from `docs/analysis-agent-model-instructions.md` into the agent's system prompt / model instructions field.
   - **Max Tokens:** 4096 (analysis output is smaller than promotional content).
   - **Temperature:** 0.7 (slightly lower than the Promotion Agent's 1.0 — we want more deterministic analytical output).
   - **Top P:** 1.0 (same as Promotion Agent).
   - **Retrieval Method:** None (no RAG needed).

### Step 2 — Record the credentials

After creating the agent, note the following from the Gradient dashboard:

- **Agent Endpoint URL** — will look like `https://xxxxx.agents.do-ai.run`
- **Access Key** — the bearer token for authentication

### Step 3 — Add to your local `.env` file

Add these two lines to your `.env` file (do **not** commit this file):

```env
# Analysis Agent (DigitalOcean Gradient)
ANALYSIS_AGENT_ENDPOINT=https://your-analysis-agent-id.agents.do-ai.run
ANALYSIS_AGENT_ACCESS_KEY=your-analysis-access-key-here
```

### Step 4 — Verify the agent responds

Test the agent with a quick curl call to confirm it's working:

```bash
curl -s -X POST "$ANALYSIS_AGENT_ENDPOINT/api/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ANALYSIS_AGENT_ACCESS_KEY" \
  -d '{
    "messages": [{
      "role": "user",
      "content": "{\"repo_url\":\"https://github.com/AcaciaMan/repo-promoter-agent\",\"repo_name\":\"repo-promoter-agent\",\"short_description\":\"AI-powered promotional content generator for GitHub repositories\",\"readme_text\":\"Generates tweets, LinkedIn posts, and marketing copy for open-source projects using a Gradient AI agent.\",\"topics\":[\"go\",\"ai\",\"github\",\"marketing\"],\"metrics\":{\"stars\":5,\"forks\":1,\"watchers\":2,\"views_14d_total\":50,\"views_14d_unique\":30,\"clones_14d_total\":10,\"clones_14d_unique\":8},\"target_audience\":\"open-source maintainers\"}"
    }]
  }' | python -m json.tool
```

**Expected:** The response should contain a JSON object in the `choices[0].message.content` field with the analysis schema fields: `primary_value_proposition`, `ideal_audience`, `key_features`, `differentiators`, `risk_or_limitations`, `social_proof_signals`, `recommended_positioning_angle`.

If the agent wraps the output in markdown fences (`` ```json ... ``` ``), that's OK — the backend client will strip them (same as the Promotion Agent client).

## What NOT to do

- Do NOT modify any Go code, HTML, or other source files.
- Do NOT commit the `.env` file with real credentials.
- Do NOT change the Promotion Agent's configuration.

## Verification checklist

- [ ] Analysis Agent exists on Gradient dashboard with correct model instructions.
- [ ] Curl test returns a valid JSON analysis matching the expected schema.
- [ ] `ANALYSIS_AGENT_ENDPOINT` and `ANALYSIS_AGENT_ACCESS_KEY` are in your local `.env` file.
- [ ] Promotion Agent still works (unchanged).
