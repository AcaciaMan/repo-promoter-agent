# Prompt: Finalize the Agent Input JSON Contract

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. The app takes a GitHub repo URL and generates promotional content via a DigitalOcean Gradient AI agent.

This is **Phase 1** — no GitHub API calls, no database, no deployment. I'm hardcoding sample repo data to prove the agent round-trip works.

## Your task

Finalize the **input JSON contract** — the structured data my Go backend sends inside the `messages[].content` field of the agent chat completion request. This is NOT the HTTP request body itself; it's the repo data embedded in the user message text.

## Current draft

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

## Open questions — please resolve each one

1. **Metrics realism**: The draft uses 120k stars for a small village app. Should I use realistic numbers (e.g., 12 stars, 3 forks) so the agent generates representative content? What values do you recommend for the hardcoded sample?

2. **`recent_clones` field**: This requires the GitHub Traffic API, which needs push access. Should I drop this field entirely from the contract, or keep it as optional (nullable)?

3. **`target_channel` semantics**: If this is `"twitter"`, does the agent still generate `linkedin_post`? Or should I use `"all"` as default? Clarify how this field should be interpreted and document it.

4. **`target_audience` free text vs enum**: Is free text the right choice, or should there be a small set of predefined audiences? What's more practical for an MVP?

5. **`readme_summary` length**: Should I set a max character/token limit for this field to avoid blowing up the prompt? What's a reasonable cap?

6. **Language/locale field**: Should I add a `language` field (e.g., `"en"`) for future internationalization, or is that over-engineering for phase 1?

7. **Missing fields**: Are there any fields I should add that would meaningfully improve the agent's output quality? (e.g., `primary_language`, `license`, `last_commit_date`)

## Deliverables

1. **Final input JSON schema** — with field names, types, required/optional markers, and a one-line description of each field.
2. **One hardcoded sample** — with realistic values for the Village Square repo, ready to paste into Go code.
3. **A brief rationale** for each decision made on the open questions above.

## Constraints

- Keep it simple — this is a hackathon MVP.
- Every field must earn its place: if it doesn't meaningfully affect agent output quality, drop it.
- The schema should be stable enough that adding GitHub API data in Phase 2 doesn't require breaking changes.
