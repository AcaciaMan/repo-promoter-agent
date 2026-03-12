```markdown
# intent-for-day1-phase1

## Objective for this phase

Establish a minimal but working “vertical slice” of the project focused on **AI generation only**, without persistence or search. The goal is to reliably turn a single public GitHub repo URL into structured promotional content via the Gradient agent, using a small Go backend and a very simple UI.

By the end of this phase, I want:

- A stable JSON contract between:
  - Go backend → Gradient agent (input shape + prompt style)
  - Gradient agent → Go backend (output schema)
- A local Go HTTP service that:
  - Accepts a GitHub repo URL
  - Calls the Gradient agent with hardcoded/mock repo data (to avoid GitHub integration on day 1)
  - Returns the agent’s JSON response to the browser
- A minimal HTML page that:
  - Lets me click a button to trigger generation
  - Displays the raw JSON response

No SQLite, no full-text search, no GitHub API calls in this phase.

---

## Constraints and assumptions

- Timebox: this phase should fit into **a few hours** on Day 1.
- I already have:
  - A DigitalOcean account
  - A Gradient agent created and working in the Playground with a known model and instructions
- I will run the Go service **locally** in this phase (no deployment yet).
- I am fine hardcoding or mocking repo metadata (name, description, README snippet) for now.
- see docs/agent-call-curl-example.md and .env file (use godotenv) for connection details to DigitalOcean agent

---

## High-level goals for Claude in this phase

I want Claude to:

1. Help solidify the **data contracts** and prompt shape:
   - Finalize the JSON input structure for the agent.
   - Finalize the JSON output schema for promotional content.
   - Draft the exact text of the `user` message I should send to the agent, embedding both the input JSON and the output schema, in a way that’s robust and easy to maintain.

2. Help design a **minimal Go backend skeleton**:
   - Standard project layout (files, packages) for a small service.
   - Minimal HTTP server with one endpoint, e.g. `POST /api/generate`.
   - Code sketch to:
     - Read a simple request (or even no body for phase 1, just use a hardcoded example).
     - Build the request body for the agent endpoint.
     - Send HTTP request to the agent.
     - Return the response back to the browser.

3. Help design a **very simple HTML/JS test client**:
   - A single HTML file with:
     - One button: “Generate promo for sample repo”.
     - JS `fetch` call to `POST /api/generate`.
     - Response printed as formatted JSON in a `<pre>` element.

---

## Scope for Day 1 / Phase 1

### In scope

- Define and freeze the **input JSON** example for the agent:

  ```json
{  
  "repo_url": "https://github.com/AcaciaMan/village-square",  
  "repo_name": "Village Square",  
  "short_description": "Digital village square for local announcements, garage sales, and connections between neighbors and suppliers.",  
  "readme_summary": "A community web app for a rural village — connecting villagers, local producers (fishermen, farmers, crafters), and the yearly Village Day celebration.",  
  "topics": ["go", "villagers", "cooperation"],  
  "metrics": { "stars": 120000, "forks": 20000, "watchers": 5000, "open_issues": 1000, "recent_clones": 500 },  
  "target_channel": "twitter",  
  "target_audience": "Villagers"  
} 
  ```

- Define and freeze the **output JSON schema**:

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

- Draft a clear **prompt template** that I can use in Go, something like:

  - System/instructions are already configured on the agent.
  - User message that embeds:
    - Repo data JSON
    - Output schema JSON
    - A short directive: “Return only a JSON object that matches the schema.”

- A Go HTTP server that:
  - Has a function to build the `messages` payload for the agent.
  - Sends it to the agent endpoint with the API key set in environment variables.
  - Returns the raw agent JSON as-is to the frontend.

- A basic HTML page with inline JS to:
  - Call the backend.
  - Show JSON.

### Explicitly out of scope for this phase

- GitHub API integration (no network calls to GitHub yet).
- SQLite database and full-text search.
- Search UI or history listing.
- Authentication or user accounts.
- Deployment to DigitalOcean.

---

## Deliverables I want from Claude in this phase

1. **Refined JSON contracts**  
   - If necessary, adjusted versions of the input and output schemas to keep them practical and stable.
   - Suggestions about any fields that are unnecessary or missing for the MVP.

2. **Prompt template text**  
   A concrete `user` message template, parameterized, e.g.:

   - Placeholder markers where I plug in the repo JSON and schema JSON.
   - Wording that strongly emphasizes valid JSON and no extra text.

3. **Go backend skeleton**  
   - Suggested project structure (e.g., `cmd/server/main.go`, `internal/agent/client.go`, etc.) that is simple enough for the hackathon.
   - Example Go code for:
     - Spinning up an HTTP server.
     - A `/api/generate` handler.
     - Calling the agent endpoint (with an HTTP client) and proxying the response.

4. **Minimal HTML test client**  
   - A snippet of HTML + JS that:
     - On button click, calls `/api/generate`.
     - Renders the returned JSON.

---

## How I will use these outputs

- I will paste the suggested schemas, prompt, Go skeleton, and HTML into my project.
- After verifying that the full round-trip works locally (hardcoded repo → agent → JSON response in browser), I will move to **Phase 2**, where we:
  - Replace hardcoded repo data with real GitHub API calls.
  - Introduce SQLite + FTS for persistence and search.
  - Incrementally improve prompts and UX.

For now, Claude should treat this document as the **single source of truth** for the Day 1 / Phase 1 intent and optimize all suggestions toward getting this minimal end-to-end flow working as quickly and reliably as possible.
```