# Prompt: "Why This Repo?" Analysis Panel on Generate View

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. After Phase 3, the `/api/generate` response now includes an `analysis` field (a nested JSON object, or `null` when analysis is unavailable). In the previous prompt (39), I added a multi-step loading indicator.

Now I need to render the analysis data as a **"Why this repo?"** panel below the promotional content in the Generate view.

The full intent document is at `docs/intent-for-analysis-agent.md` — read it for high-level context.

## Current state

### API response shape (after Phase 3)

```json
{
  "id": 1,
  "repo_url": "https://github.com/owner/repo",
  "repo_name": "repo-name",
  "headline": "...",
  "summary": "...",
  "key_benefits": ["..."],
  "tags": ["..."],
  "twitter_posts": ["..."],
  "linkedin_post": "...",
  "call_to_action": "...",
  "target_channel": "twitter",
  "target_audience": "...",
  "views_14d_total": 100,
  "views_14d_unique": 50,
  "clones_14d_total": 20,
  "clones_14d_unique": 10,
  "created_at": "...",
  "analysis": {
    "repo_url": "...",
    "repo_name": "...",
    "primary_value_proposition": "One sentence about what this repo does.",
    "ideal_audience": ["Segment 1", "Segment 2"],
    "key_features": ["Feature benefit 1", "Feature benefit 2"],
    "differentiators": ["What makes it special"],
    "risk_or_limitations": ["Early stage", "Limited docs"],
    "social_proof_signals": ["Modest traction with steady clones"],
    "recommended_positioning_angle": ["Time-saver for busy devs"]
  }
}
```

When analysis is not available: `"analysis": null`.

### `static/index.html` — current `renderFull()` function

```javascript
function renderFull(d){
    let h='<div class="card"><h3>'+esc(d.headline)+'</h3><p>'+esc(d.summary)+'</p>';
    if(d.key_benefits&&d.key_benefits.length){h+='<strong>Key Benefits</strong><ul class="benefits">';d.key_benefits.forEach(b=>{h+='<li>'+esc(b)+'</li>'});h+='</ul>'}
    if(d.twitter_posts&&d.twitter_posts.length){h+='<strong>Twitter Posts</strong>';d.twitter_posts.forEach(t=>{h+='<div class="tweet"><span>'+esc(t)+'</span>'+tweetCopyBtn()+'</div>'})}
    if(d.linkedin_post){h+='<strong>LinkedIn Post</strong><div class="linkedin-block">'+esc(d.linkedin_post)+'</div>'+liCopyBtn()}
    if(d.call_to_action){h+='<div class="cta">'+esc(d.call_to_action)+'</div>'}
    if(d.tags&&d.tags.length){h+='<div class="pills">';d.tags.forEach(t=>{h+='<span class="pill">'+esc(t)+'</span>'});h+='</div>'}
    h+=renderTraffic(d);
    if(d.id){h+='<div class="meta">ID: '+d.id+' · Saved: '+esc(d.created_at||'')+'</div>'}
    h+='<br><button class="toggle" onclick="toggleRaw()">Show raw JSON</button><pre class="hidden" id="raw-json">'+esc(JSON.stringify(d,null,2))+'</pre>';
    h+='</div>';
    return h;
}
```

## Your task

### 1. Add a `renderAnalysis()` function

Create a new function that renders the analysis panel. Call it from `renderFull()`.

```javascript
function renderAnalysis(analysis) {
    if (!analysis) return '';

    let h = '<div class="analysis-panel">';
    h += '<h4>🔍 Why This Repo?</h4>';
    h += '<div class="analysis-label">AI-generated analysis</div>';

    if (analysis.primary_value_proposition) {
        h += '<div class="analysis-field">';
        h += '<strong>Value Proposition</strong>';
        h += '<p>' + esc(analysis.primary_value_proposition) + '</p>';
        h += '</div>';
    }

    if (analysis.ideal_audience && analysis.ideal_audience.length) {
        h += '<div class="analysis-field">';
        h += '<strong>Ideal Audience</strong><ul>';
        analysis.ideal_audience.forEach(a => { h += '<li>' + esc(a) + '</li>'; });
        h += '</ul></div>';
    }

    if (analysis.key_features && analysis.key_features.length) {
        h += '<div class="analysis-field">';
        h += '<strong>Key Features</strong><ul>';
        analysis.key_features.forEach(f => { h += '<li>' + esc(f) + '</li>'; });
        h += '</ul></div>';
    }

    if (analysis.differentiators && analysis.differentiators.length) {
        h += '<div class="analysis-field">';
        h += '<strong>Differentiators</strong><ul>';
        analysis.differentiators.forEach(d => { h += '<li>' + esc(d) + '</li>'; });
        h += '</ul></div>';
    }

    if (analysis.risk_or_limitations && analysis.risk_or_limitations.length) {
        h += '<div class="analysis-field">';
        h += '<strong>Risks & Limitations</strong><ul>';
        analysis.risk_or_limitations.forEach(r => { h += '<li>' + esc(r) + '</li>'; });
        h += '</ul></div>';
    }

    if (analysis.social_proof_signals && analysis.social_proof_signals.length) {
        h += '<div class="analysis-field">';
        h += '<strong>Social Proof</strong><ul>';
        analysis.social_proof_signals.forEach(s => { h += '<li>' + esc(s) + '</li>'; });
        h += '</ul></div>';
    }

    if (analysis.recommended_positioning_angle && analysis.recommended_positioning_angle.length) {
        h += '<div class="analysis-field">';
        h += '<strong>Positioning Angles</strong><ul>';
        analysis.recommended_positioning_angle.forEach(a => { h += '<li>' + esc(a) + '</li>'; });
        h += '</ul></div>';
    }

    h += '</div>';
    return h;
}
```

### 2. Call `renderAnalysis()` from `renderFull()`

In `renderFull()`, insert the analysis panel **after** the traffic block and **before** the metadata/raw-JSON section. Add this line after `h+=renderTraffic(d);`:

```javascript
h += renderAnalysis(d.analysis);
```

### 3. Add CSS for the analysis panel

Add these styles to the `<style>` block:

```css
.analysis-panel {
    background: #f0fdf4;
    border: 1px solid #86efac;
    border-left: 4px solid #22c55e;
    border-radius: 6px;
    padding: 1rem;
    margin: .75rem 0;
}

.analysis-panel h4 {
    margin: 0 0 .25rem;
    font-size: 1rem;
    color: #166534;
}

.analysis-label {
    font-size: .75rem;
    color: #888;
    margin-bottom: .75rem;
    font-style: italic;
}

.analysis-field {
    margin-bottom: .5rem;
}

.analysis-field strong {
    display: block;
    font-size: .85rem;
    color: #333;
    margin-bottom: .15rem;
}

.analysis-field p {
    margin: .2rem 0;
    font-size: .9rem;
}

.analysis-field ul {
    margin: .15rem 0 0 1.2rem;
    padding: 0;
    font-size: .9rem;
}

.analysis-field li {
    margin: .15rem 0;
}
```

The green-tinted background distinguishes the analysis panel from the promotional content (which uses the standard card styling).

### 4. Handle `analysis: null` gracefully

The `renderAnalysis()` function already returns an empty string when `analysis` is falsy (null, undefined). This means:
- When analysis is present → green "Why this repo?" panel appears.
- When analysis is null → nothing renders, no empty box, no error.

No additional null checks needed in `renderFull()`.

## What NOT to change

- Do not modify any backend code.
- Do not modify the search view (`renderCompact()`, `renderExpandedBody()`) — that's the next prompt.
- Do not modify the loading UX (prompt 39).
- Do not change `renderTraffic()`, `toggleRaw()`, or any existing rendering logic.
- Keep the existing card structure intact — the analysis panel is an addition, not a replacement.

## Verification

1. Open `http://localhost:8080` with both agents configured.
2. Paste a repo URL and click Generate.
3. After generation completes, verify:
   - The promotional content card appears as before.
   - Below the traffic block, a green-bordered "Why this repo?" panel appears.
   - All analysis fields render with their labels: Value Proposition, Ideal Audience, Key Features, Differentiators, Risks & Limitations, Social Proof, Positioning Angles.
   - The "AI-generated analysis" label is visible.
4. Test with analysis disabled (stop server, unset `ANALYSIS_AGENT_ENDPOINT`, restart):
   - Generate a promotion.
   - The "Why this repo?" panel should NOT appear (no empty box, no error).
5. Click "Show raw JSON" — confirm the full response including `analysis` is visible.
