# Prompt: Analysis in Search Cards + Phase 5 Smoke Test

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. In the previous prompts (39–40), I added a multi-step loading indicator and a "Why this repo?" analysis panel on the Generate view. Now I need to show compact analysis snippets in the Search/Browse cards and verify the full Phase 5 frontend experience.

The full intent document is at `docs/intent-for-analysis-agent.md` — read it for high-level context.

## Current state

### `/api/search` response shape (after Phase 4)

Each promotion in the `results` array includes an `analysis` field — either a JSON object or `null`.

### `static/index.html` — search card rendering

**`renderCompact(p)` — the search card:**
```javascript
function renderCompact(p){
    const firstTweet=p.twitter_posts&&p.twitter_posts.length?p.twitter_posts[0]:'';
    const tags=(p.tags||[]).map(t=>'<span class="pill">'+esc(t)+'</span>').join('');
    const date=fmtDate(p.created_at);
    return '<div class="card compact-card" onclick="this.querySelector(\'.expand\').classList.toggle(\'hidden\')">'
        +'<h3><a href="'+safeHref(p.repo_url)+'" target="_blank" rel="noopener" onclick="event.stopPropagation()">'+esc(p.repo_name)+'</a></h3>'
        +'<p>'+esc(p.headline)+'</p>'
        +(firstTweet?'<p class="meta" style="color:#333">\uD83D\uDC26 '+esc(firstTweet)+'</p>':'')
        +'<div class="pills">'+tags+'</div>'
        +renderTrafficCompact(p)
        +'<div class="meta">'+esc(date)+'</div>'
        +'<div class="expand hidden" style="margin-top:.75rem">'+renderExpandedBody(p)+'</div>'
        +'</div>';
}
```

**`renderExpandedBody(d)` — shown when a card is clicked:**
```javascript
function renderExpandedBody(d){
    let h='<p>'+esc(d.summary)+'</p>';
    if(d.key_benefits&&d.key_benefits.length){h+='<strong>Key Benefits</strong><ul class="benefits">';d.key_benefits.forEach(b=>{h+='<li>'+esc(b)+'</li>'});h+='</ul>'}
    if(d.twitter_posts&&d.twitter_posts.length){h+='<strong>Twitter Posts</strong>';d.twitter_posts.forEach(t=>{h+='<div class="tweet"><span>'+esc(t)+'</span>'+tweetCopyBtn()+'</div>'})}
    if(d.linkedin_post){h+='<strong>LinkedIn Post</strong><div class="linkedin-block">'+esc(d.linkedin_post)+'</div>'+liCopyBtn()}
    if(d.call_to_action){h+='<div class="cta">'+esc(d.call_to_action)+'</div>'}
    h+=renderTraffic(d);
    return h;
}
```

### `renderAnalysis()` (from prompt 40)

Already exists — renders the full "Why this repo?" panel. Can be reused in the expanded card body.

## Your task

### 1. Add compact analysis snippet to search cards

In `renderCompact()`, add a one-line analysis snippet **below the headline and above the first tweet**. This should show the `primary_value_proposition` as a subtitle when analysis is available.

Add a new helper function:

```javascript
function renderAnalysisCompact(analysis) {
    if (!analysis || !analysis.primary_value_proposition) return '';
    return '<p class="analysis-snippet">💡 ' + esc(analysis.primary_value_proposition) + '</p>';
}
```

Insert it in `renderCompact()` right after the headline `<p>`:

```javascript
return '<div class="card compact-card" ...>'
    +'<h3>...</h3>'
    +'<p>'+esc(p.headline)+'</p>'
    +renderAnalysisCompact(p.analysis)       // ← NEW
    +(firstTweet?'<p class="meta" ...>':'')
    ...
```

### 2. Add CSS for the analysis snippet

Add to the `<style>` block:

```css
.analysis-snippet {
    color: #166534;
    font-size: .85rem;
    margin: .2rem 0 .4rem;
    padding-left: .25rem;
    border-left: 2px solid #86efac;
}
```

This gives a subtle green left-border to visually distinguish the analysis line from the promotional headline.

### 3. Add full analysis panel to expanded card body

In `renderExpandedBody()`, add the full analysis panel at the end (after traffic). Reuse the existing `renderAnalysis()` from prompt 40:

Add this line at the end of `renderExpandedBody()`, before `return h;`:

```javascript
h += renderAnalysis(d.analysis);
```

This way, when a user clicks to expand a search card, they see the full "Why this repo?" panel (same as on the Generate view).

### 4. Graceful degradation for cards without analysis

Both `renderAnalysisCompact()` and `renderAnalysis()` already return empty strings when `analysis` is null/undefined. Legacy promotions (without analysis) will render exactly as before — just the headline, tweet, tags, and date. No empty boxes, no errors.

## What NOT to change

- Do not modify any backend code.
- Do not modify the Generate view or `renderFull()` — that was handled in prompt 40.
- Do not modify the loading UX — that was handled in prompt 39.
- Do not change the card click-to-expand behavior.
- Do not add analysis to the FTS search index.

## Verification — Full Phase 5 smoke test

After making the search card changes, run through this comprehensive test of the entire Phase 5 frontend:

### Test 1 — Loading UX (from prompt 39)

1. Open `http://localhost:8080` with both agents configured.
2. Paste a repo URL (e.g., `https://github.com/AcaciaMan/acacia-log`) and click Generate.
3. **Watch the status text** — it should progress through:
   - "🔍 Analyzing repository…"
   - "🔍 Still analyzing…" (after ~8s)
   - "✍️ Generating promotion…" (after ~15s)
   - "⏳ Almost there…" (after ~25s, if still waiting)
4. When the response arrives, status text clears immediately.
5. If it pulses/animates, that's a bonus.

### Test 2 — "Why this repo?" panel on Generate view (from prompt 40)

1. After Test 1 completes, scroll down to see the result.
2. Below the promotional content (tweets, LinkedIn post, CTA, traffic), there should be a **green-bordered "Why this repo?" panel** containing:
   - Value Proposition
   - Ideal Audience (list)
   - Key Features (list)
   - Differentiators
   - Risks & Limitations
   - Social Proof
   - Positioning Angles
   - "AI-generated analysis" label
3. Click "Show raw JSON" and confirm the `analysis` object is present.

### Test 3 — Search cards with analysis snippets

1. In the Search & Browse section, the recently generated promotion should appear.
2. The card should show:
   - Repo name (link), headline, **green analysis snippet** (💡 value proposition), first tweet, tags, date.
3. Click the card to expand it — the full "Why this repo?" panel should appear in the expanded view.

### Test 4 — Legacy cards without analysis

1. If the DB has older promotions without analysis:
   - Their cards should render normally — no green snippet, no empty space.
   - Clicking to expand should show the normal body — no empty analysis panel.
2. If all promotions have analysis, generate one with analysis disabled (unset env vars) and verify.

### Test 5 — Analysis disabled entirely

1. Stop the server.
2. Unset `ANALYSIS_AGENT_ENDPOINT` and `ANALYSIS_AGENT_ACCESS_KEY`.
3. Start the server.
4. Generate a promotion — the loading UX should still work (progress steps appear), and when complete:
   - No "Why this repo?" panel (analysis is null).
   - The rest of the promotion renders normally.
5. Browse/search — cards should render without analysis snippets.

### Test 6 — No JavaScript errors

1. Open browser DevTools Console (F12).
2. Generate a promotion and browse search results.
3. Confirm **no JS errors** in the console — especially no errors about accessing properties of null/undefined on the `analysis` object.

## Fix any issues found

Common issues:
- **Null access:** `analysis.primary_value_proposition` when `analysis` is null → fix with the null guard in `renderAnalysisCompact()`.
- **Missing function:** `renderAnalysis` not defined yet → confirm prompt 40 was implemented first.
- **CSS conflicts:** Green border clashing with card border → adjust margins/padding.
- **Expanded view missing analysis:** Forgot to add `renderAnalysis(d.analysis)` to `renderExpandedBody()`.

Fix any issues and re-test.
