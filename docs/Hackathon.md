## DigitalOcean Gradient AI Hackathon – High‑Level Context

### Hackathon overview

- The project is being built for the **DigitalOcean Gradient AI Hackathon**.  
- Goal of the hackathon: create a working AI‑powered application that uses **DigitalOcean Gradient AI** (agents, models, or related features) and is deployed on DigitalOcean infrastructure (e.g., App Platform).  
- The solution should demonstrate a clear, practical use of AI and offer a coherent end‑to‑end user experience (from input, through AI processing, to usable output in a UI).

### Problem space

- Developers often struggle to **promote their GitHub repositories** effectively.  
- Writing good descriptions, tweets, LinkedIn posts, and other promotional content takes time and marketing skills many developers don’t have.  
- Even when promotional content exists, it may be scattered and not easily searchable or reusable across channels and repos.

### High‑level idea

Build a **web app + AI agent** that turns GitHub repositories into **searchable promotional content packages**.

Core concept:

- User provides a **public GitHub repository URL**.  
- Backend fetches basic repository data (name, description, README content, and later stars/forks/other metrics).  
- An **AI agent (hosted on DigitalOcean Gradient)** generates structured promotional content tailored to different channels (Twitter/X, LinkedIn, etc.).  
- Generated content is stored in a local database (SQLite with Full‑Text Search) and exposed through a simple web UI where users can **search, browse, and reuse** promo material.

### Target users

- Open‑source maintainers who want to attract contributors and users.  
- Individual developers building portfolios and side projects.  
- Small teams that maintain multiple repositories and want a central place to manage promotional copy.

### Main goals

- Reduce the effort required to **promote** a GitHub repository.  
- Provide **consistent, structured promotional content** across multiple channels.  
- Make generated content **searchable and discoverable** so users can quickly find good descriptions/posts for future reuse.

***

## Planned architecture (very high level)

### Components

1. **Frontend (web UI)**  
   - Simple HTML/CSS/JavaScript interface.  
   - Key screens:
     - “Generate” page: input field for GitHub repo URL, optional selection of target channel/audience, button to generate content.
     - “Search” page: search bar and filters to browse previously generated promotional content.

2. **Backend (Go service)**  
   - Handles HTTP requests from the frontend.  
   - Integrates with:
     - **GitHub API** to fetch repo name, description, and README from public repositories.  
     - **AI agent endpoint** (DigitalOcean Gradient) to generate promotional content.  
     - **SQLite database with Full‑Text Search** to store and query generated content.
   - Exposes endpoints such as:
     - `POST /api/generate` – generate promotional content for a repo and store it.
     - `GET /api/search` – search generated content by text, tags, or channel.

3. **AI agent (Gradient)**  
   - Configured as a **“GitHub Repo Promotion Agent.”**  
   - Uses a general‑purpose, high‑quality LLM.  
   - Receives structured repo data and instructions, and returns strictly formatted JSON with promotional content.

4. **Storage (SQLite + FTS)**  
   - Local database file managed by the Go backend.  
   - One main table or FTS virtual table for “promotions” containing:
     - Repo metadata (URL, name, short description).  
     - Generated content (headline, summary, tweets, LinkedIn post, tags, call‑to‑action).  
     - Timestamps and possibly simple channel metadata.  
   - Full‑Text Search is used to implement `/search` over summaries, tags, and generated posts.

***

## AI agent responsibility – conceptual description

### Role

- The AI agent acts as a **marketing assistant for GitHub repositories**.  
- It does *not* fetch data on its own; it assumes the backend has already provided all necessary repo information.  
- It focuses on **transforming** structured input into usable, well‑phrased promotional content.

### Input (from backend)

The backend sends a structured object with:

- `repo_url`: the GitHub repository URL.  
- `repo_name`: the repository name.  
- `short_description`: short description from the repository metadata.  
- `readme_summary`: a summary or truncated content of the README file.  
- `topics`: list of repository topics/tags.  
- `metrics`: optional metrics like stars, forks, watchers, open issues, recent clones.  
- `target_channel`: e.g., `"twitter"`, `"linkedin"`, or `"general"`.  
- `target_audience`: short description of the intended audience (e.g., “backend developers learning Go”).

### Output (to backend)

The agent returns a **single JSON object** with fields such as:

- `repo_url` – copy from input.  
- `repo_name` – copy from input.  
- `headline` – short, catchy one‑line pitch for the project.  
- `summary` – one to two paragraphs describing what the project does and why it matters.  
- `key_benefits` – list of bullet‑style benefits.  
- `tags` – list of short, reusable tags/keywords.  
- `twitter_posts` – array of tweet‑length promotional messages.  
- `linkedin_post` – a LinkedIn‑style promotional text.  
- `call_to_action` – one sentence inviting users to star, try, or contribute.

The agent is instructed to:

- Stay faithful to the input and avoid inventing features.  
- Adjust style slightly depending on `target_channel`.  
- Always return valid JSON matching the agreed schema, with no extra text.

***

## User flows (high level)

### 1. Generate promotional content

1. User opens the “Generate” page.  
2. User pastes a public GitHub repository URL and selects target channel/audience (optional).  
3. Frontend calls `POST /api/generate` with repo URL and channel/audience.  
4. Backend:
   - Fetches repo details (name, description, README) from GitHub.  
   - Constructs input JSON and prompt for the AI agent.  
   - Calls the AI agent endpoint and receives JSON promotional content.  
   - Stores the result in SQLite (including all text fields in an FTS‑enabled table).  
   - Returns the generated content to the frontend.  
5. Frontend displays the content in cards/sections with “copy” buttons.

### 2. Search previously generated content

1. User opens the “Search” page.  
2. User enters a search query (e.g., “CLI tool”, “Go microservices”, or “testing helpers”).  
3. Frontend calls `GET /api/search?query=...`.  
4. Backend runs a Full‑Text Search query on the SQLite FTS table.  
5. Matching promotions are returned and rendered as a list of cards showing:
   - Repo name and URL.  
   - Headline and short summary.  
   - A preview of one tweet/LinkedIn post.  
   - Option to expand and copy the full content.

***

## Non‑goals (to keep scope manageable)

- No complex authentication or user accounts for the first version (can assume a single trusted user or later add simple auth).  
- No complex metrics dashboards or analytics beyond basic storage.  
- No need to support private repositories initially.  
- No multi‑agent orchestration; a **single agent** is enough for MVP.
- Currently optimized for the maintainer’s own repos; can be extended to multi‑user via a GitHub App.

***

## Implementation priorities

1. Get the **AI agent** working reliably with a stable JSON schema.  
2. Implement the **Go backend** that:
   - Calls the agent.  
   - Stores results in SQLite with FTS.  
3. Build the **minimal frontend** for:
   - Generating content for a single repo.  
   - Searching previously generated content.  
4. Iteratively improve prompts and UX once the basic flow is functional.

***

This document is intended as **very high‑level context** for an implementation assistant (e.g., Claude Opus 4.6 in VS Code) so it understands the hackathon context, the problem being solved, core components, and the boundaries of the first version.