# Output JSON Contract — v1 (Phase 1)

This document defines the JSON structure the Gradient agent must return inside `choices[0].message.content`. The Go backend extracts this string and parses it.

---

## 1. Final Output JSON Schema

| Field              | Type       | Required | Description                                                                 |
|--------------------|------------|----------|-----------------------------------------------------------------------------|
| `repo_url`         | `string`   | yes      | Echo of the input repo URL. Included for storage convenience.               |
| `repo_name`        | `string`   | yes      | Echo of the input repo name. Included for storage convenience.              |
| `headline`         | `string`   | yes      | Attention-grabbing one-line headline for the project.                       |
| `summary`          | `string`   | yes      | 2–3 sentence promotional summary of the project.                           |
| `key_benefits`     | `[]string` | yes      | 3–5 bullet-style benefit statements.                                        |
| `tags`             | `[]string` | yes      | 5–8 discoverable tags/keywords, expanding on input topics.                  |
| `twitter_posts`    | `[]string` | yes      | 3 ready-to-post tweets (≤280 chars each, including hashtags).               |
| `linkedin_post`    | `string`   | yes      | LinkedIn post, 150–300 words, professional tone.                            |
| `call_to_action`   | `string`   | yes      | One-sentence CTA directing readers to the repo.                             |

---

## 2. Realistic Example Output

```json
{
  "repo_url": "https://github.com/AcaciaMan/village-square",
  "repo_name": "Village Square",
  "headline": "Bring Your Village Online — A Digital Town Square for Local Communities",
  "summary": "Village Square is a lightweight community web app that helps rural villages share announcements, organize garage sales, and connect neighbors with local producers. Built with Go for simplicity and speed, it's the easiest way to give your community a digital home.",
  "key_benefits": [
    "Centralized announcements — no more missed village news or events",
    "Connect local producers (farmers, crafters, fishermen) directly with neighbors",
    "Simple and lightweight — runs anywhere, no complex setup required",
    "Organize community events like Village Day with built-in coordination tools"
  ],
  "tags": ["community", "local", "go", "village", "announcements", "open-source", "neighbors"],
  "twitter_posts": [
    "Your village deserves a digital town square. 🏘️ Village Square connects neighbors, local producers, and community events in one simple app. Built with Go. #OpenSource #Community\nhttps://github.com/AcaciaMan/village-square",
    "Tired of missing local announcements? Village Square puts garage sales, farmer updates, and village news in one place. Check it out 👇 #GoLang #CommunityTech\nhttps://github.com/AcaciaMan/village-square",
    "A small Go app making a big difference for rural communities. 🌾 Village Square helps villagers connect, trade, and organize — no tech skills required. #OpenSource\nhttps://github.com/AcaciaMan/village-square"
  ],
  "linkedin_post": "Every village has a heartbeat — the morning greetings at the market, the flyer on the community board, the word-of-mouth about a neighbor's fresh catch. But as life gets busier, these connections can fade.\n\nVillage Square is an open-source web app designed to keep that heartbeat strong. Built with Go and designed for simplicity, it gives rural communities a digital space to:\n\n• Share local announcements and event updates\n• Connect producers — farmers, fishermen, crafters — directly with their neighbors\n• Organize community traditions like the yearly Village Day celebration\n• Post and discover garage sales and local services\n\nThis isn't another social network trying to replace human connection. It's a simple, focused tool that mirrors the real-world village square — a place where everyone knows what's happening and how to find each other.\n\nThe project is early-stage and open source, which means it's a great time to get involved if you care about community tech, Go development, or building tools for underserved use cases.\n\nCheck it out on GitHub: https://github.com/AcaciaMan/village-square\n\n#OpenSource #GoLang #CommunityTech #RuralTech #LocalFirst",
  "call_to_action": "Star the repo on GitHub and bring your village online — every community deserves a digital town square."
}
```

---

## 3. Rationale for Each Decision

### Q1 — `repo_url` and `repo_name` echo-back
**Decision:** Keep them in the output.

Yes, they're redundant with the input — but they make the output self-contained. In Phase 2, each output record gets stored in Solr. Having `repo_url` and `repo_name` already present means the stored record is immediately useful without joining back to the request. The agent copies them verbatim from the input, so hallucination risk is negligible. Cost: two trivial string fields.

### Q2 — `target_channel` vs always generating all channels
**Decision:** Option (a) — always generate all channels.

The input contract already defined `target_channel` as a tone/emphasis hint, not a filter. The agent always populates both `twitter_posts` and `linkedin_post`. This is the simplest approach: one fixed output shape, no conditional null-handling, and the frontend can display whichever channel the user cares about. Dropping fields based on `target_channel` would require conditional parsing and validation logic that isn't worth it for an MVP.

### Q3 — `twitter_posts` count
**Decision:** Exactly 3 tweets.

Three gives variety without bloat. The prompt will instruct "generate exactly 3 tweets." A fixed count makes parsing predictable — the Go backend knows to expect a 3-element array. If the user wants more options later, bumping to 5 is a non-breaking change (just array length).

### Q4 — `linkedin_post` length
**Decision:** 150–300 words, specified in the prompt.

LinkedIn posts perform best at this length — long enough to tell a story, short enough to avoid the "see more" truncation killing engagement. The prompt will include this guidance. No need for a separate schema field to enforce it; it's a prompt instruction.

### Q5 — `key_benefits` count
**Decision:** 3–5 items.

A range rather than a fixed count gives the agent flexibility based on how feature-rich the project is. A tiny project might only have 3 compelling benefits; a larger one might warrant 5. The prompt will say "3 to 5 bullet points."

### Q6 — `tags` relationship to input `topics`
**Decision:** Tags are an expansion of input topics, not a copy.

The agent should use the input `topics` as a starting point and generate additional relevant tags for discoverability. Limit: 5–8 tags total. This keeps the tag list useful for search (Phase 2) without being unwieldy.

### Q7 — Missing fields considered
- **`elevator_pitch`**: Omitted. Too similar to `headline` — the distinction would confuse the agent and produce near-duplicate content. The `summary` already serves as a longer pitch.
- **`github_description`**: Omitted. The input `short_description` already is the GitHub description. Auto-generating a new one adds complexity for unclear value.
- **`hashtags`**: Omitted as separate field. Hashtags are embedded directly in `twitter_posts` and `linkedin_post` where they're actually used. A separate list would be redundant.
- **`target_channel` echo-back**: Omitted from output. The backend already knows what `target_channel` it sent — it can attach this metadata when storing. No need for the agent to echo it back.

### Q8 — Error/confidence signal
**Decision:** Omitted.

Adding a confidence field increases schema complexity and gives the agent an escape hatch to produce weaker content ("confidence: low" instead of trying harder). In Phase 1 with hardcoded data, confidence is meaningless. If the agent produces bad output, we'll see it in the response and adjust the prompt. In Phase 2, input validation (e.g., minimum README length) is a backend concern, not an agent concern.
