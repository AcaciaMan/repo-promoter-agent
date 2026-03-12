# Prompt: Finalize the Agent Prompt Template

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. A Go backend sends a chat completion request to a DigitalOcean Gradient AI agent, which generates promotional content for GitHub repos.

The agent already has **system instructions configured on the DigitalOcean side** (I set them up in the Gradient Playground). The system prompt tells the agent it's a "GitHub Repo Promotion Agent" that generates marketing content.

What I need from this session is the **user message template** — the text that goes into `messages[0].content` in each request.

## Agent API format

The request body looks like this (OpenAI-compatible chat completions):

```json
{
  "messages": [
    {
      "role": "user",
      "content": "<THIS IS WHAT I NEED YOU TO WRITE>"
    }
  ],
  "stream": false
}
```

The response comes back as:

```json
{
  "choices": [
    {
      "message": {
        "content": "<the agent's text response — should be pure JSON>"
      }
    }
  ]
}
```

## Input contract (finalized)

> **Note**: Paste the final input schema from prompt 01 here before running this session. For now, here's the draft:

```json
{
  "repo_url": "https://github.com/AcaciaMan/village-square",
  "repo_name": "Village Square",
  "short_description": "...",
  "readme_summary": "...",
  "topics": ["go", "villagers", "cooperation"],
  "metrics": { "stars": 12, "forks": 3, "watchers": 5, "open_issues": 2 },
  "target_channel": "twitter",
  "target_audience": "Villagers"
}
```

## Output contract (finalized)

> **Note**: Paste the final output schema from prompt 02 here before running this session. For now, here's the draft:

```json
{
  "repo_url": "string",
  "repo_name": "string",
  "headline": "string",
  "summary": "string",
  "key_benefits": ["string"],
  "tags": ["string"],
  "twitter_posts": ["string"],
  "linkedin_post": "string",
  "call_to_action": "string"
}
```

## Your task

Write the **exact user message template** that I'll use in my Go code. This template will be a Go string with placeholders (e.g., `%s` for `fmt.Sprintf`, or template markers) where I inject the repo data JSON.

## Requirements for the prompt template

1. **Embed the repo data** — the full input JSON should be included in the message so the agent has all context.

2. **Embed the output schema** — show the agent the exact JSON structure it must return.

3. **Strict JSON-only output** — the prompt must strongly instruct the agent to:
   - Return ONLY a valid JSON object.
   - No markdown fences, no explanatory text, no preamble.
   - No trailing commas or comments.

4. **Content guidelines** — the prompt should tell the agent to:
   - Stay faithful to the input data; don't invent features not in the README.
   - Adjust tone based on `target_channel` (e.g., casual/punchy for Twitter, professional for LinkedIn).
   - Keep `twitter_posts` within 280 characters each.

5. **Robust against common LLM failure modes**:
   - Wrapping JSON in ```json ... ``` fences (most common failure).
   - Adding "Here is the JSON:" preamble.
   - Truncating long outputs.
   - Hallucinating repo features.

## Deliverables

1. **The exact prompt template text** — ready to paste into a Go `const` or string literal. Use `{{.RepoDataJSON}}` and `{{.OutputSchemaJSON}}` as the two placeholder markers (Go template style).

2. **A Go code snippet** showing how to render the template with `text/template` or `fmt.Sprintf` — whichever you recommend and why.

3. **A fallback extraction strategy** — a short description (2-3 sentences) of how the Go backend should handle the response if the agent wraps the JSON in markdown fences or adds preamble text. This will be implemented in a later prompt but I want the strategy decided now.

## Constraints

- The prompt should be **as short as practical** — longer prompts cost more tokens and increase latency.
- Don't repeat the system instructions (the agent already knows its role).
- Optimize for **reliability of valid JSON output** over creativity of the content.
