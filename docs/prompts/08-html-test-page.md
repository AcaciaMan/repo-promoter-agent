# Prompt: Create the HTML Test Page and Smoke Test

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. This is **Phase 1** — a local Go service that calls a Gradient AI agent and returns promotional content.

All Go code is now implemented:
- `cmd/server/main.go` — entry point, serves static files from `static/`
- `internal/agent/client.go` — agent HTTP client
- `internal/handler/generate.go` — `POST /api/generate` handler

This prompt creates the **frontend test page** and defines the **manual smoke test** procedure.

## Your task

Two things:

1. Create `static/index.html` — a minimal test page.
2. Write a step-by-step **smoke test checklist** I can follow to verify the full round-trip.

## Requirements for `static/index.html`

### Layout

A single HTML file with inline CSS and JS. No build tools, no frameworks, no external CDN dependencies.

Sections:
1. **Header**: project name ("Repo Promoter Agent") and a one-line description.
2. **Generate button**: labeled "Generate promo for sample repo". Clicking it calls `POST /api/generate`.
3. **Status indicator**: shows "Generating..." while the request is in flight, disappears on completion.
4. **Response display area**: a `<pre>` block that shows the returned JSON, pretty-printed with `JSON.stringify(data, null, 2)`.
5. **Error display**: if the request fails, show the error message in red text.

### JavaScript behavior

```js
// Pseudocode:
button.onclick = async () => {
  clearPreviousResults();
  showLoading();
  try {
    const res = await fetch('/api/generate', { method: 'POST' });
    if (!res.ok) {
      const err = await res.json();
      showError(err.error || res.statusText);
      return;
    }
    const data = await res.json();
    showResult(data);
  } catch (e) {
    showError(e.message);
  } finally {
    hideLoading();
  }
};
```

### Styling

- Minimal, clean CSS — no need for beauty, just readability.
- Monospace font for the JSON output.
- The button should be clearly clickable.
- Mobile-friendly is NOT required.

### Optional enhancement (only if trivial)

- A "Copy JSON" button next to the output that copies the raw JSON to clipboard.

## Requirements for the smoke test checklist

Write a numbered checklist I can follow to verify Phase 1 works end-to-end:

1. Prerequisites check (`.env` file, Go installed, dependencies fetched).
2. Start the server (`go run cmd/server/main.go`).
3. Verify the server starts (expected log output).
4. Open `http://localhost:8080` in the browser.
5. Verify the HTML page loads.
6. Click "Generate promo for sample repo".
7. Verify the loading indicator appears.
8. Verify JSON output appears (list the key fields to check).
9. Verify the JSON is valid (no markdown fences, no extra text).
10. Test error scenarios:
    - What happens if the agent is unreachable?
    - What happens if you send a GET instead of POST to `/api/generate`?

## Deliverables

1. **`static/index.html`** — full, working HTML file.
2. **Smoke test checklist** — as a markdown section I can paste into docs or follow directly.
3. **Any fixes** — if you spot issues in the Go code from previous prompts that would prevent the smoke test from passing, list them (but don't re-implement — I'll fix in a separate session).

## Constraints

- No external dependencies (no CDN links, no npm, no build step).
- The page must work when served by Go's `http.FileServer` from the `static/` directory.
- Keep it under 150 lines of HTML/CSS/JS total.
- The page should work in any modern browser (Chrome, Firefox, Edge).
