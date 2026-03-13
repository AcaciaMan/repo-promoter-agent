# Prompt: Phase 2 Smoke Test — Verify Traffic Metrics in UI

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. I just finished **Phase 2** which added traffic metrics display to the frontend:

- Prompt 21: Traffic block on the generate result card.
- Prompt 22: Compact traffic summary + full block on search result cards.

The full intent document is at `docs/intent-for-views-clones.md`.

## Your task

Verify that Phase 2 is working correctly. This is a **test and fix** prompt.

### Step 1: Compile and start

```bash
go build ./...
go run cmd/server/main.go
```

Make sure the server starts with no errors. The `GITHUB_TOKEN` should be set in `.env` or environment.

### Step 2: Generate — AcaciaMan repo (traffic available)

1. Open `http://localhost:8080` in a browser.
2. Enter `https://github.com/AcaciaMan/village-square` (or another AcaciaMan repo) in the URL field.
3. Click **Generate**.

**Verify:**
- The result card shows the normal content (headline, summary, tweets, LinkedIn, CTA, tags).
- Below the tags, there is a blue-tinted **"📊 Traffic (last 14 days)"** block showing:
  - `👁 Views: X total / Y unique`
  - `📦 Clones: X total / Y unique`
- The numbers may be small (even zero is okay if the repo has no recent traffic) — the point is the block renders correctly.
- The meta line (ID + date) appears below the traffic block.
- The raw JSON toggle still works and shows the traffic fields in the JSON.

### Step 3: Generate — non-AcaciaMan repo (no traffic)

1. Enter `https://github.com/golang/go` in the URL field.
2. Click **Generate**.

**Verify:**
- The result card shows normal content.
- There is **no** traffic block visible (all metrics are 0).
- Everything else renders the same as before.

### Step 4: Search — compact cards with traffic

1. Switch to the Search section.
2. Search for a term that matches an AcaciaMan promotion (e.g., "village" or leave empty to browse recent).

**Verify:**
- Compact search cards for AcaciaMan repos show a one-line blue text: `📊 Views: X · Clones: Y` between the tags and the date.
- Compact cards for non-AcaciaMan repos do NOT show this line.

### Step 5: Search — expanded cards with traffic

1. Click on an AcaciaMan repo card to expand it.

**Verify:**
- The expanded section shows the full **"📊 Traffic (last 14 days)"** block at the bottom (after CTA), with total/unique breakdown for both views and clones.
- Click on a non-AcaciaMan card — no traffic block in the expanded section.

### Step 6: Backward compatibility

1. If there are old promotions in the database from before Phase 1 (with NULL/0 traffic values):
   - They should render cleanly in both compact and expanded views.
   - No traffic info should show (since values are 0).
   - No JavaScript errors in the browser console.

### Step 7: No token scenario

1. Remove `GITHUB_TOKEN` from the environment.
2. Restart the server.
3. Generate for an AcaciaMan repo.

**Verify:**
- The generate result card does NOT show a traffic block (metrics are 0 because no token was available to fetch them).
- No errors in the UI.

## Common issues to fix

- **Traffic block shows "undefined":** The JavaScript checks for `d.views_14d_total` etc. If the field is missing from older records, it may be `undefined`. Fix with `(d.views_14d_total||0)` in the rendering.
- **Traffic block appears when all zeros:** The check `!d.views_14d_total && !d.views_14d_unique && ...` should correctly return false for all-zero, but double-check edge cases.
- **Layout broken:** If the traffic block pushes other elements out of alignment, adjust margins/padding in the `.traffic-block` CSS.
- **Compact metric line wraps badly:** If the one-line summary is too long on mobile, check it still looks acceptable.

## Deliverable

After this prompt, Phase 2 is **complete and verified**:

- Generate result cards show traffic metrics for AcaciaMan repos.
- Search result cards show compact + expanded traffic metrics.
- Everything degrades gracefully when metrics are unavailable.
- No JavaScript errors in the browser console.
