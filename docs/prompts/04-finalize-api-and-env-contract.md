# Prompt: Finalize the API & Environment Contract

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. A Go backend calls a Gradient AI agent and returns promotional content to a browser frontend.

This is **Phase 1** — local development only, hardcoded repo data, no database.

## Your task

Finalize three things:

1. The **`/api/generate` endpoint contract** — what the frontend sends and receives.
2. The **environment variables** the Go backend needs.
3. The **agent response envelope handling** — how the backend extracts content from the chat completion response.

---

## Part 1: `/api/generate` endpoint contract

### Current ambiguity

The intent doc says "Read a simple request (or even no body for phase 1)." I need a decision.

### Options

- **(a)** `POST /api/generate` with a JSON body containing at minimum `repo_url` (other fields hardcoded on backend for phase 1). This makes the Phase 2 transition smooth.
- **(b)** `POST /api/generate` with no body — fully hardcoded sample. Simpler but requires a contract change in Phase 2.
- **(c)** `POST /api/generate` with the full input JSON in the body — frontend sends everything. Most flexible but more work for the HTML test page.

### Please decide and deliver

1. **Request contract**: method, path, content-type, body schema (with types and required/optional).
2. **Success response contract**: status code, content-type, body schema.
3. **Error response contract**: status code, body shape for errors (e.g., `{"error": "message"}`).
4. Rationale for the choice.

---

## Part 2: Environment variables

### Known requirements

From the agent API (see curl example below), the backend needs:

```
AGENT_ENDPOINT=https://xxxxx.agents.do-ai.run
AGENT_ACCESS_KEY=xxxxxxxxxxxxx
```

### Curl example for agent API

```bash
curl -i \
  -X POST \
  $AGENT_ENDPOINT/api/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $AGENT_ACCESS_KEY" \
  -d '{
    "messages": [{"role": "user", "content": "Hello"}],
    "stream": false
  }'
```

### Please decide and deliver

1. **Complete list of env vars** the Go backend needs (including any for the HTTP server itself, e.g., `PORT`).
2. **A `.env.example` file** — with placeholder values and comments, safe to commit to git.
3. Whether to add any validation (e.g., fail fast on startup if `AGENT_ENDPOINT` is missing).

---

## Part 3: Agent response envelope handling

### The problem

The agent returns an OpenAI-compatible chat completion response:

```json
{
  "id": "chatcmpl-xxx",
  "object": "chat.completion",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "{ \"repo_url\": \"...\", \"headline\": \"...\", ... }"
      },
      "finish_reason": "stop"
    }
  ],
  "usage": { "prompt_tokens": 500, "completion_tokens": 300, "total_tokens": 800 }
}
```

The promotional JSON is a **string** inside `choices[0].message.content`. The backend must:
1. Parse the outer envelope.
2. Extract the `content` string.
3. Validate it's valid JSON.
4. Return it to the frontend.

### Please decide and deliver

1. **Go type definitions** for the agent response envelope (just the fields we need, not everything).
2. **Extraction logic** — pseudocode or description of the steps, including:
   - What if `choices` is empty?
   - What if `content` is wrapped in markdown fences (```json ... ```)?
   - What if `content` is not valid JSON?
3. **What the backend returns to the frontend**: the raw promotional JSON directly, or wrapped in its own envelope (e.g., `{"data": {...}, "usage": {...}}`)?

---

## Constraints

- Keep it minimal — hackathon MVP.
- Decisions should be **Phase 2 compatible**: adding a request body field or a database write later shouldn't require redesigning the endpoint.
- Use `godotenv` for `.env` loading (already decided).
- The Go backend serves the static HTML file directly (no CORS needed).
