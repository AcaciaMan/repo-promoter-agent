# Prompt: Show Traffic Metrics on Search Result Cards

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. The app generates promotional content for GitHub repos. I'm working on **Phase 2** — exposing traffic metrics in the UI.

The previous prompt (21) added a traffic metrics block to the **generate** result card. Now I need to add the same metrics to **search** result cards.

The full intent document is at `docs/intent-for-views-clones.md`.

## Current frontend (`static/index.html`)

### Existing `renderTraffic` helper (added in prompt 21)

```javascript
function renderTraffic(d){
    if(!d.views_14d_total && !d.views_14d_unique && !d.clones_14d_total && !d.clones_14d_unique) return '';
    return '<div class="traffic-block">'
        +'<strong>📊 Traffic (last 14 days)</strong>'
        +'<span class="metric">👁 Views: '+d.views_14d_total+' total / '+d.views_14d_unique+' unique</span>'
        +'<span class="metric">📦 Clones: '+d.clones_14d_total+' total / '+d.clones_14d_unique+' unique</span>'
        +'</div>';
}
```

### Existing CSS (added in prompt 21)

```css
.traffic-block{background:#f0f7ff;border:1px solid #c8e1ff;border-radius:4px;padding:.5rem .75rem;margin:.5rem 0;font-size:.9rem}
.traffic-block strong{display:block;margin-bottom:.25rem}
.traffic-block .metric{display:inline-block;margin-right:1rem;color:#333}
```

### Current `renderCompact` function (search result cards)

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
        +'<div class="meta">'+esc(date)+'</div>'
        +'<div class="expand hidden" style="margin-top:.75rem">'+renderExpandedBody(p)+'</div>'
        +'</div>';
}
```

### Current `renderExpandedBody` function (expanded details in search cards)

```javascript
function renderExpandedBody(d){
    let h='<p>'+esc(d.summary)+'</p>';
    if(d.key_benefits&&d.key_benefits.length){h+='<strong>Key Benefits</strong><ul class="benefits">';d.key_benefits.forEach(b=>{h+='<li>'+esc(b)+'</li>'});h+='</ul>'}
    if(d.twitter_posts&&d.twitter_posts.length){h+='<strong>Twitter Posts</strong>';d.twitter_posts.forEach(t=>{h+='<div class="tweet"><span>'+esc(t)+'</span>'+tweetCopyBtn()+'</div>'})}
    if(d.linkedin_post){h+='<strong>LinkedIn Post</strong><div class="linkedin-block">'+esc(d.linkedin_post)+'</div>'+liCopyBtn()}
    if(d.call_to_action){h+='<div class="cta">'+esc(d.call_to_action)+'</div>'}
    return h;
}
```

## Your task

Add traffic metrics to search result cards in two places.

### 1. Compact view — inline metrics summary

Add a compact one-line metrics summary to `renderCompact`, shown between the tags (`.pills`) and the date (`.meta`). Use a new helper:

```javascript
function renderTrafficCompact(d){
    if(!d.views_14d_total && !d.views_14d_unique && !d.clones_14d_total && !d.clones_14d_unique) return '';
    return '<div class="meta" style="color:#0969da">📊 Views: '+d.views_14d_total+' · Clones: '+d.clones_14d_total+'</div>';
}
```

This shows just the total views and total clones — enough to scan at a glance without cluttering the compact card. Insert it in `renderCompact` between the tags and the date line.

### 2. Expanded view — full traffic block

In `renderExpandedBody`, append the full `renderTraffic(d)` block at the end (after the CTA), so users see the complete metrics when they click to expand a card.

## What NOT to do

- Do NOT modify any Go code (handler, store, agent, github client).
- Do NOT change the generate result card (that was done in prompt 21).
- Do NOT add any new API calls.
- Do NOT change any CSS — reuse the `.traffic-block` and `.metric` classes already added in prompt 21, and the existing `.meta` class for the compact summary.

## Verification

After implementation:

1. Open `http://localhost:8080` in a browser.
2. If there are stored promotions for AcaciaMan repos, the search results should show:
   - In the compact card: a one-line summary ("📊 Views: 42 · Clones: 8") in blue text between tags and date.
   - In the expanded card (click to expand): the full traffic block with total/unique breakdown.
3. For non-AcaciaMan repos (all metrics 0), neither compact nor expanded view shows any traffic info.
4. Old promotions stored before Phase 1 (with no traffic columns or all zeros) should render cleanly — no broken display.
