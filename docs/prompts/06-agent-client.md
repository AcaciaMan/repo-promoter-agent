# Prompt: Implement the Agent HTTP Client

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. This is **Phase 1** — a local Go service that calls a DigitalOcean Gradient AI agent with hardcoded repo data.

The project structure and `main.go` were created in prompt 05. This prompt fills in the agent client implementation.

## Existing project state

> **IMPORTANT**: Before running this prompt, paste your actual project file listing and key type definitions here. For now, here's the expected structure:

```
cmd/server/main.go          # entry point (done — prompt 05)
internal/agent/client.go    # THIS FILE — implement now
internal/handler/generate.go # stub (prompt 07)
static/index.html           # not yet created (prompt 08)
```

## Agent API details

The agent follows the **OpenAI chat completions API** format:

**Endpoint**: `$AGENT_ENDPOINT/api/v1/chat/completions`

**Request**:
```json
{
  "messages": [
    {
      "role": "user",
      "content": "<user message with embedded repo data and output schema>"
    }
  ],
  "stream": false
}
```

**Response**:
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
  "usage": {
    "prompt_tokens": 500,
    "completion_tokens": 300,
    "total_tokens": 800
  }
}
```

The promotional content is a **JSON string** inside `choices[0].message.content`.

## Finalized contracts (from previous sessions)

> **IMPORTANT**: Paste your finalized input contract, output contract, and prompt template here before running.

### Input contract (from prompt 01)
```json
{
  "repo_url": "string",
  "repo_name": "string",
  "short_description": "string",
  "readme_summary": "string",
  "topics": ["string"],
  "metrics": { "stars": 0, "forks": 0, "watchers": 0, "open_issues": 0 },
  "target_channel": "string",
  "target_audience": "string"
}
```

### Output contract (from prompt 02)
> Paste finalized output schema here.

### Prompt template (from prompt 03)
> Paste finalized prompt template here (Go template string with `{{.RepoDataJSON}}` etc.).

## Your task

Implement `internal/agent/client.go` — the complete agent HTTP client.

## Requirements

### 1. Client struct

```go
type Client struct {
    endpoint   string       // base URL (e.g., "https://xxx.agents.do-ai.run")
    accessKey  string       // Bearer token
    httpClient *http.Client // with a reasonable timeout
}
```

- Constructor: `NewClient(endpoint, accessKey string) *Client`
- Set an HTTP client timeout (e.g., 30 seconds — agent responses can be slow).

### 2. Request types

Define Go structs for:
- The chat completion **request body** (messages array, stream flag).
- The chat completion **response envelope** (only the fields we need: `choices[0].message.content`).

### 3. Input data type

Define a Go struct for the **repo input data** that matches the finalized input contract. Use JSON struct tags.

### 4. Core method: `Generate`

```go
func (c *Client) Generate(ctx context.Context, input RepoInput) (json.RawMessage, error)
```

This method should:
1. Marshal the `input` to JSON.
2. Render the prompt template, embedding the input JSON and output schema.
3. Build the chat completion request body.
4. Send the HTTP POST to `$AGENT_ENDPOINT/api/v1/chat/completions` with:
   - `Content-Type: application/json`
   - `Authorization: Bearer $AGENT_ACCESS_KEY`
5. Parse the response envelope.
6. Extract `choices[0].message.content`.
7. **Clean the content**: strip markdown fences (```json ... ```) if present, trim whitespace.
8. Validate that the result is valid JSON (`json.Valid`).
9. Return the raw JSON bytes.

### 5. Error handling

- Return clear errors for: HTTP failures, non-2xx status, empty choices, invalid JSON content.
- Include the HTTP status code in error messages when relevant.
- Do NOT panic — always return errors.

### 6. Prompt template

Embed the finalized prompt template as a Go constant or `text/template`. The `Generate` method renders it with the input data.

## Deliverables

1. **`internal/agent/client.go`** — full, working Go code with all types, constructor, and `Generate` method.
2. **Any necessary helper functions** (e.g., `stripMarkdownFences`).
3. **Update to `main.go`** — if the constructor signature changed from the stub, show the diff.

## Constraints

- Standard library only (no external HTTP client libraries).
- The code must compile together with the existing `main.go` and handler stub.
- Use `context.Context` for the HTTP request (to support future cancellation/timeouts).
- Keep the code straightforward — one file, no unnecessary abstractions.
- The `Generate` method returns `json.RawMessage` (not a parsed struct) so the handler can pass it through without re-marshaling.
