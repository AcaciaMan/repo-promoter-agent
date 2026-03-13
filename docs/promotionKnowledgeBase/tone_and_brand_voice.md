# Tone and Brand Voice

## Personality

- Developer‑centric, practical, and respectful of engineers’ time.
- Curious and helpful, like a teammate who “gets” both code and communication.
- Confident but modest; the project is an MVP from a hackathon, not a fully fledged platform (yet).

## Core voice principles

- Clarity over cleverness: explain what Repo Promoter Agent does in plain language first.
- Benefits over buzzwords: talk about saving time, reducing friction, and reusing good copy.
- Honest about limitations: it works with public repos, is still evolving, and may not get every nuance right.
- GitHub‑aware: language should acknowledge repos, READMEs, maintainers, stars, issues, etc.

## Do write like this

- “Paste your GitHub repo URL and get ready‑to‑use tweets and a LinkedIn post in seconds.”
- “You keep control over what gets published; Repo Promoter Agent just gives you a strong starting point.”
- “This is a hackathon MVP, so feedback and bug reports are very welcome.”

Characteristics:

- Use “you” and “we” where appropriate.
- Use concrete nouns and verbs (“paste a URL”, “copy a tweet”) instead of vague abstractions.
- Mention GitHub and DigitalOcean Gradient when relevant, but keep explanations short.

## Don’t write like this

- “Revolutionary AI that will replace all marketers forever.”
- “Unlock unprecedented synergies with our next‑gen social media paradigm.”
- Overpromising guarantees (“will 10x your stars overnight”) or implying official endorsement by GitHub or DigitalOcean.

Avoid:

- Heavy marketing jargon.
- Clickbait or deceptive claims.
- Speaking as if the tool is production‑grade enterprise software (unless/when it actually is).

## Style guidelines

- Spelling: Prefer US English.
- Tense: Present tense for current capabilities, future tense only for clearly marked roadmap ideas.
- Emojis:  
  - Allowed in social posts (Twitter/X, LinkedIn) in moderation (e.g., 🚀, 💡, 📢).  
  - Avoid in documentation and error messages.
- Hashtags:  
  - Use 2–4 relevant tags in social posts, e.g. `#GitHub`, `#DevTools`, `#DigitalOcean`, `#GradientAI`, `#Hackathon`.
- Links:  
  - When possible, link directly to the repo or live demo.  
  - Prefer descriptive anchors (“See the repo”) over bare URLs in longer copy.

## Positioning cues to reinforce in tone

- Built for developers: assume the reader understands Git, repos, and basic tooling.
- Helper, not replacement: “assists you with promo copy” rather than “does everything for you.”
- Lightweight and hackable: simple Go backend, SQLite, and a straightforward UI; easy to run and extend.

Example phrasing:

- “Repo Promoter Agent is a small tool that helps you talk about your projects without leaving GitHub behind.”
- “If you already keep your project story in the README, this agent turns that into shareable posts.”

## Example on‑brand snippets

- Social post (Twitter/X):  
  “Maintainer brain: great at code, tired of writing launch posts.  
  Repo Promoter Agent turns a GitHub repo URL into headlines, tweets, and a LinkedIn post using @digitalocean Gradient.  
  Ship code, not marketing copy.  
  https://github.com/AcaciaMan/repo-promoter-agent”

- Social post (LinkedIn):  
  “I built Repo Promoter Agent for developers who are strong on code but short on time for promotion.  
  Paste a public GitHub repo URL, and an AI agent on DigitalOcean Gradient generates headlines, tweets, a LinkedIn‑style post, and tags.  
  Everything is saved in a searchable SQLite library so you can reuse and refine your promo copy over time.”

- Documentation tone example:  
  “To get started, provide a public GitHub repository URL on the Generate page. The backend will fetch the repo metadata and README, send it to the Gradient AI agent, and return a promotion bundle. You can copy individual pieces of text or search for them later on the Search page.”

## Guardrails

- Always make it clear that the user should review and edit generated copy before posting.
- Never imply that Repo Promoter Agent has access to private repositories or private data.
- Do not claim official partnership or endorsement from GitHub or DigitalOcean unless that becomes explicitly true.

