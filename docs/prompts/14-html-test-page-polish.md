# Prompt: Polish the HTML Frontend

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. The project has a working Go backend and a single-page HTML frontend at `static/index.html`.

The frontend currently has two sections:
1. **Generate** — input repo URL, channel, audience → calls `POST /api/generate` → renders structured result cards
2. **Search & Browse** — search/list stored promotions → expandable compact cards

## Current state of `static/index.html`

The page works but has rough edges. Here is the current file:

> **IMPORTANT**: Paste the full current `static/index.html` content here before running this prompt so the session has exact context.

### What works

- Generate form with repo URL input, channel select, audience text input
- POST to `/api/generate` with JSON body
- Result rendering with headline, summary, key benefits, twitter posts, LinkedIn post, call to action, tags
- Copy buttons on tweets and LinkedIn post (via MutationObserver)
- Raw JSON toggle
- Search input calls `GET /api/search?q=...`
- Compact cards with expand-on-click
- Auto-loads recent promotions on page load
- XSS-safe text escaping via `esc()` helper

### Issues to fix

1. **Copy buttons not rendering on tweets** — the `MutationObserver` approach for adding copy buttons to `.tweet` elements is fragile. Tweets inside the generate result get copy buttons, but tweets inside expanded search results do not (they're added to a different parent). Fix by generating copy buttons inline in the `renderFull` and `renderExpandedBody` functions instead of relying on `MutationObserver`.

2. **LinkedIn copy button placement** — the MutationObserver targets `#li-block` by ID, but IDs must be unique and this breaks when multiple results are visible. Use class-based targeting or inline the copy button.

3. **No loading feedback for search** — when search results are loading, there's no visual feedback beyond the "Searching..." text. Add a brief disabled state on the search button.

4. **Empty state for generate** — when no URL is entered and default sample is used, there's no indication that hardcoded data was used. Add a small info note: "Using sample repo (no URL provided)".

5. **Accessibility** — buttons missing accessible labels, form inputs missing proper association. Do a quick pass for basic a11y (proper `for`/`id` on labels, button `aria-label` where needed).

6. **Compact card date formatting** — `new Date(p.created_at).toLocaleDateString()` may show "Invalid Date" if the SQLite timestamp format doesn't parse correctly. The backend returns `created_at` in `"2006-01-02T15:04:05Z"` or `"2006-01-02 15:04:05"` format. Handle both.

## Your task

Produce an **updated `static/index.html`** that fixes all 6 issues above while keeping the existing design and layout intact.

## Requirements

- Fix all 6 issues listed above.
- Keep inline CSS and JS — no external files.
- Keep the page under ~300 lines.
- Don't change the API contract (same fetch calls, same JSON shapes).
- All text rendering must remain XSS-safe (use the `esc()` helper).
- No external dependencies (no CDN, no frameworks).

## Deliverables

1. **Updated `static/index.html`** — full file replacement.
2. **Brief changelog** — one line per fix explaining what changed.

## Constraints

- Don't redesign the page — just fix and polish.
- Don't add new features beyond what's listed.
- Works in Chrome, Firefox, Edge (modern browsers only).
