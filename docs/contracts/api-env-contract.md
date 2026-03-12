# API & Environment Contract — v1 (Phase 1)

---

## Part 1: `/api/generate` Endpoint Contract

### Decision: Option (a) — `POST` with `repo_url` in body

**Request**

| Property       | Value                          |
|----------------|--------------------------------|
| Method         | `POST`                         |
| Path           | `/api/generate`                |
| Content-Type   | `application/json`             |

**Request body schema**

| Field            | Type     | Required | Description                                                              |
|------------------|----------|----------|--------------------------------------------------------------------------|
| `repo_url`       | `string` | yes      | GitHub repo URL. In Phase 1 the backend ignores this and uses hardcoded data. In Phase 2 it drives a GitHub API lookup. |
| `target_channel` | `string` | no       | `"twitter"`, `"linkedin"`, or `"all"`. Default: `"all"`.                |
| `target_audience`| `string` | no       | Free-text audience hint. Default: empty (agent decides).                |

Phase 1 example:
```json
{
  "repo_url": "https://github.com/AcaciaMan/village-square"
}
```

**Success response**

| Property     | Value              |
|--------------|--------------------|
| Status       | `200 OK`           |
| Content-Type | `application/json` |

Body: the promotional JSON directly — **no wrapper envelope**. The frontend receives the exact output contract schema:

```json
{
  "repo_url": "...",
  "repo_name": "...",
  "headline": "...",
  "summary": "...",
  "key_benefits": ["..."],
  "tags": ["..."],
  "twitter_posts": ["..."],
  "linkedin_post": "...",
  "call_to_action": "..."
}
```

**Error response**

| Status | When                                           |
|--------|------------------------------------------------|
| `400`  | Missing `repo_url`, invalid JSON body          |
| `502`  | Agent returned invalid/unparseable response    |
| `500`  | Internal error (env not loaded, template fail) |

Body shape (all errors):
```json
{
  "error": "human-readable error message"
}
```

### Rationale

Option (a) is the right balance. Sending `repo_url` in the body means the frontend already has the right contract shape for Phase 2 — just add more fields to the request. Option (b) would require a breaking contract change. Option (c) is over-engineered for a test page that only needs a button. The backend hardcodes all other fields for Phase 1 but the endpoint shape doesn't change when Phase 2 adds GitHub API lookups.

The response is returned **unwrapped** (no `{"data": ..., "usage": ...}` envelope). Simpler for the frontend to consume, and usage metadata isn't needed for Phase 1. If Phase 2 needs usage stats, a `X-Token-Usage` response header or optional query parameter can add them without breaking the body contract.

---

## Part 2: Environment Variables

### Complete list

| Variable           | Required | Description                                              |
|--------------------|----------|----------------------------------------------------------|
| `AGENT_ENDPOINT`   | yes      | Base URL of the Gradient agent (e.g., `https://xxxxx.agents.do-ai.run`) |
| `AGENT_ACCESS_KEY` | yes      | Bearer token for the agent API                           |
| `PORT`             | no       | HTTP server listen port. Default: `8080`                 |

### `.env.example` file

```env
# DigitalOcean Gradient AI Agent
# Get these from the DigitalOcean Gradient dashboard
AGENT_ENDPOINT=https://your-agent-id.agents.do-ai.run
AGENT_ACCESS_KEY=your-access-key-here

# HTTP server port (optional, default: 8080)
PORT=8080
```

### Startup validation

**Yes — fail fast.** The backend should check on startup that `AGENT_ENDPOINT` and `AGENT_ACCESS_KEY` are set and non-empty. If either is missing, log a clear error message and exit immediately. This prevents confusing runtime errors 30 seconds later when the first request fails.

```go
func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required environment variable %s is not set", key)
	}
	return v
}
```

---

## Part 3: Agent Response Envelope Handling

### Go type definitions

Only the fields we need — ignore `id`, `object`, `usage`, etc.

```go
// ChatCompletion is the minimal envelope for the agent's response.
type ChatCompletion struct {
	Choices []Choice `json:"choices"`
}

type Choice struct {
	Message Message `json:"message"`
}

type Message struct {
	Content string `json:"content"`
}
```

### Extraction logic

```
1. json.Unmarshal response body into ChatCompletion
   → fail? return 502 "failed to parse agent response envelope"

2. Check len(Choices) > 0
   → empty? return 502 "agent returned no choices"

3. Extract content = Choices[0].Message.Content

4. Try json.Unmarshal content into PromoOutput
   → success? return it

5. Fallback: strip markdown fences
   - Regex: (?s)```(?:json)?\s*(.*?)\s*```
   - If match found, extract group 1, retry json.Unmarshal
   → success? return it

6. Fallback: find first '{' and last '}'
   - Extract substring, retry json.Unmarshal
   → success? return it

7. All attempts failed → return 502 "agent response is not valid JSON"
```

### What the backend returns to the frontend

The **raw promotional JSON directly** — the parsed output contract object, re-serialized with `json.Marshal`. No wrapper envelope.

Rationale: The frontend only cares about the promotional content. Wrapping it in `{"data": ...}` adds complexity on both sides for no Phase 1 benefit. The `Content-Type: application/json` header and HTTP status code provide all the metadata the frontend needs.
