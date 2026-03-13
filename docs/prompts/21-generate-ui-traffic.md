# Prompt: Show Traffic Metrics on the Generate Result Card

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. The app generates promotional content for GitHub repos using an AI agent, stores results in SQLite, and serves a single-page HTML frontend.

I recently completed **Phase 1** (backend plumbing) which added GitHub traffic metrics (views & clones) for AcaciaMan repositories. The API already returns these fields in every promotion response:

```json
{
  "views_14d_total": 42,
  "views_14d_unique": 15,
  "clones_14d_total": 8,
  "clones_14d_unique": 5,
  ...other promotion fields...
}
```

This is **Phase 2, Step 1**: make the generate result card display traffic metrics.

The full intent document is at `docs/intent-for-views-clones.md`.

## Current frontend (`static/index.html`)

The HTML is a single file with inline CSS and JS. Key rendering function for the generate result:

```javascript
function renderFull(d){
    let h='<div class="card"><h3>'+esc(d.headline)+'</h3><p>'+esc(d.summary)+'</p>';
    if(d.key_benefits&&d.key_benefits.length){h+='<strong>Key Benefits</strong><ul class="benefits">';d.key_benefits.forEach(b=>{h+='<li>'+esc(b)+'</li>'});h+='</ul>'}
    if(d.twitter_posts&&d.twitter_posts.length){h+='<strong>Twitter Posts</strong>';d.twitter_posts.forEach(t=>{h+='<div class="tweet"><span>'+esc(t)+'</span>'+tweetCopyBtn()+'</div>'})}
    if(d.linkedin_post){h+='<strong>LinkedIn Post</strong><div class="linkedin-block">'+esc(d.linkedin_post)+'</div>'+liCopyBtn()}
    if(d.call_to_action){h+='<div class="cta">'+esc(d.call_to_action)+'</div>'}
    if(d.tags&&d.tags.length){h+='<div class="pills">';d.tags.forEach(t=>{h+='<span class="pill">'+esc(t)+'</span>'});h+='</div>'}
    if(d.id){h+='<div class="meta">ID: '+d.id+' · Saved: '+esc(d.created_at||'')+'</div>'}
    h+='<br><button class="toggle" onclick="toggleRaw()">Show raw JSON</button><pre class="hidden" id="raw-json">'+esc(JSON.stringify(d,null,2))+'</pre>';
    h+='</div>';
    return h;
}
```

Existing CSS classes used: `.card`, `.meta`, `.pills`, `.pill`, `.tweet`, `.linkedin-block`, `.cta`, `.benefits`.

## Your task

Add a **"Traffic (last 14 days)"** section to the `renderFull` function, shown between the tags (`.pills`) and the metadata line (`.meta`).

### Requirements

1. **Only show when metrics are present.** Check if at least one of `views_14d_total`, `views_14d_unique`, `clones_14d_total`, `clones_14d_unique` is greater than zero. If all are zero or undefined, don't render the section at all.

2. **Render a small metrics block** with this structure:

   ```
   📊 Traffic (last 14 days)
   👁 Views: 42 total / 15 unique
   📦 Clones: 8 total / 5 unique
   ```

3. **CSS styling:** Add a new `.traffic-block` class (inline in the `<style>` block):

   ```css
   .traffic-block{background:#f0f7ff;border:1px solid #c8e1ff;border-radius:4px;padding:.5rem .75rem;margin:.5rem 0;font-size:.9rem}
   .traffic-block strong{display:block;margin-bottom:.25rem}
   .traffic-block .metric{display:inline-block;margin-right:1rem;color:#333}
   ```

4. **Helper function.** Create a small helper `renderTraffic(d)` that returns the HTML string (or empty string if no metrics):

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

5. **Insert into `renderFull`** — call `renderTraffic(d)` after the tags pills and before the meta line:

   ```javascript
   // after: if(d.tags&&d.tags.length){...}
   h+=renderTraffic(d);
   // before: if(d.id){...}
   ```

## What NOT to do

- Do NOT modify any Go code (handler, store, agent, github client).
- Do NOT change the search results rendering yet (that's the next prompt).
- Do NOT add any new API calls. The data is already in the response.
- Do NOT change any existing HTML structure or JS logic outside of the generate result rendering.

## Verification

After implementation:

1. Open `http://localhost:8080` in a browser.
2. Generate for an AcaciaMan repo (e.g., `https://github.com/AcaciaMan/village-square`).
   - The result card should show a blue-tinted traffic block below the tags, showing views and clones.
3. Generate for a non-AcaciaMan repo (e.g., `https://github.com/golang/go`).
   - The result card should NOT show a traffic block (all metrics are 0).
4. The rest of the card (headline, summary, tweets, LinkedIn, CTA, tags, raw JSON) should render identically to before.
