# Prompt: Finalize the Agent Output JSON Contract

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. A Go backend sends repo data to a Gradient AI agent, which returns promotional content as JSON inside a chat completion response.

This is **Phase 1** — proving the agent round-trip works with hardcoded data.

The agent endpoint follows the OpenAI chat completions API format. The promotional JSON will be a **string** inside `choices[0].message.content`. My Go backend must extract and parse it.

## Your task

Finalize the **output JSON schema** — the structure the agent must return inside its message content. This schema will be embedded in the prompt to instruct the agent what to produce.

## Current draft

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

## Open questions — please resolve each one

1. **`repo_url` and `repo_name` echo-back**: These are copied from input. Is this useful in the output (e.g., for storage convenience), or is it redundant noise the agent might get wrong? Should they stay or be added by the backend instead?

2. **`target_channel` vs always generating all channels**: The input has `target_channel`, but the output always includes both `twitter_posts` and `linkedin_post`. Options:
   - (a) Always generate all channels regardless of `target_channel` — simplest.
   - (b) Only populate the requested channel, leave others as `null` or empty.
   - (c) Drop `target_channel` from input entirely.
   - Recommend one approach and explain why.

3. **`twitter_posts` — how many?**: Should the schema specify a count (e.g., "generate exactly 3 tweets") or leave it flexible? What's a good default count?

4. **`linkedin_post` — length guidance**: Should the schema include guidance on length (e.g., "200-500 words") or leave it to the agent?

5. **`key_benefits` — how many?**: Should I specify a count (e.g., 3-5 bullet points)?

6. **`tags` — relationship to input `topics`**: Should `tags` be an expansion of the input `topics`, or independently generated? Should there be a count limit?

7. **Missing fields**: Consider whether any of these would add value:
   - `elevator_pitch` (one-sentence version, distinct from `headline`)
   - `github_description` (optimized repo description for GitHub's About field)
   - `hashtags` (separate from `tags`, specifically for social media)
   - `target_channel` echo-back (so the stored record knows which channel was requested)

8. **Error/confidence signal**: Should the schema include a field where the agent can signal low confidence or flag issues (e.g., "README was too short to generate meaningful content")?

## Deliverables

1. **Final output JSON schema** — with field names, types, required/optional markers, and a one-line description of each field.
2. **One realistic example output** — filled in as the agent would produce it for a small community project, ready to use as a test fixture.
3. **Brief rationale** for each decision made on the open questions.

## Constraints

- Keep it simple — hackathon MVP.
- The schema should be small enough that the agent can reliably produce valid JSON every time. More fields = more chance of errors.
- Fields should serve the frontend display or future search. Don't add fields just because they're possible.
- Schema must be stable enough to store in SQLite in Phase 2 without breaking changes.
