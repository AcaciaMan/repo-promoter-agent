# Prompt: Result Grouping by Repo URL

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. The app stores AI-generated promotional content in **Apache Solr 10** and exposes full-text search.

This is **Phase 5, prompt 2 of 5**. Prompt 80 added spellcheck / "Did you mean?". This prompt adds result grouping — when multiple promotions exist for the same repo (e.g. one for Twitter, one for LinkedIn), they're clustered together under a single card instead of shown as separate results.

## Current state

### `SearchResult` in `internal/store/store.go`

```go
type SearchResult struct {
    Results    []Promotion                  `json:"results"`
    Facets     map[string][]Facet           `json:"facets,omitempty"`
    Highlights map[string]map[string]string `json:"highlights,omitempty"`
    Collation  string                       `json:"collation,omitempty"`
}
```

### `Store.Search()` — returns flat list

```go
docs, err := parseSolrDocs(body)
// ...
return SearchResult{Results: docs, Facets: facets, Highlights: highlights, Collation: collation}, nil
```

### `searchResponse` in `internal/handler/search.go`

```go
type searchResponse struct {
    Results    []store.Promotion            `json:"results"`
    Count      int                          `json:"count"`
    Facets     map[string][]store.Facet     `json:"facets,omitempty"`
    Highlights map[string]map[string]string `json:"highlights,omitempty"`
    Collation  string                       `json:"collation,omitempty"`
}
```

### Frontend `renderCompact(p)` — renders each Promotion as a card

Each result is rendered individually. There's no concept of grouping.

### Document model

Each `Promotion` has a `repo_url` and a `target_channel`. A single repo can produce multiple promotions (one per channel: general, twitter, linkedin). All promotions for the same repo share the same `repo_url` but have different `target_channel` values and different generated content.

## Your task

Instead of using Solr's group query (which changes the response format significantly and complicates faceting/highlighting), implement **client-side grouping** in the frontend. This is simpler, keeps the backend stable, and works well for our use case where result sets are small (max 100).

Modify only `static/index.html` to:
1. Group results by `repo_url` after receiving the API response
2. Show the first promotion as the main card
3. Add channel tabs within expanded cards to switch between promotions for the same repo

## Requirements

### 1. Add CSS for channel tabs

Add to the `<style>` block, after the `.did-you-mean` rules:

```css
.channel-tabs{display:flex;gap:0;margin:.75rem 0 .5rem;border-bottom:2px solid #e1e4e8}
.channel-tab{padding:.35rem .75rem;font-size:.8rem;font-weight:600;cursor:pointer;border:none;background:none;color:#666;border-bottom:2px solid transparent;margin-bottom:-2px;transition:color .15s,border-color .15s}
.channel-tab:hover{color:#333}
.channel-tab.active{color:#0969da;border-bottom-color:#0969da}
.channel-tab .ch-icon{margin-right:.3rem}
.group-count{font-size:.75rem;color:#888;font-weight:400;margin-left:.5rem}
```

### 2. Add grouping logic and update rendering

Add a `groupByRepo` function in the `<script>` block, before the `renderCompact` function:

```js
function groupByRepo(results){
    const map=new Map();
    results.forEach(p=>{
        const key=p.repo_url;
        if(!map.has(key))map.set(key,[]);
        map.get(key).push(p);
    });
    return Array.from(map.values());
}

function channelIcon(ch){
    if(ch==='twitter')return '🐦';
    if(ch==='linkedin')return '💼';
    return '📢';
}
```

### 3. Update `doSearch()` to render grouped results

In `doSearch()`, replace the line:

```js
searchResults.innerHTML=d.results.map(renderCompact).join('');
```

With:

```js
const groups=groupByRepo(d.results);
searchResults.innerHTML=groups.map(renderGroup).join('');
```

### 4. Add `renderGroup` function

Add after the `channelIcon` function:

```js
function renderGroup(group){
    const p=group[0]; // primary promotion (highest relevance)
    const firstTweet=p.twitter_posts&&p.twitter_posts.length?p.twitter_posts[0]:'';
    const tags=(p.tags||[]).map(t=>'<span class="pill">'+esc(t)+'</span>').join('');
    const date=fmtDate(p.created_at);
    const groupCount=group.length>1?'<span class="group-count">'+group.length+' channels</span>':'';
    return '<div class="card compact-card" onclick="this.querySelector(\'.expand\').classList.toggle(\'hidden\')">'
        +'<h3><a href="'+safeHref(p.repo_url)+'" target="_blank" rel="noopener" onclick="event.stopPropagation()">'+esc(p.repo_name)+'</a>'+groupCount+'</h3>'
        +'<div class="meta" style="user-select:all;cursor:text" onclick="event.stopPropagation()">'+esc(p.repo_url)+'</div>'
        +'<p>'+hlField(p.repo_url,'headline',p.headline)+'</p>'
        +renderAnalysisCompact(p.analysis)
        +(firstTweet?'<p class="meta" style="color:#333">\uD83D\uDC26 '+esc(firstTweet)+'</p>':'')
        +'<div class="pills">'+tags+'</div>'
        +renderStatsCompact(p)
        +renderTrafficCompact(p)
        +'<div class="meta">'+esc(date)+'</div>'
        +'<div class="expand hidden" style="margin-top:.75rem" onclick="event.stopPropagation()">'
        +renderGroupedExpanded(group)
        +'</div>'
        +'</div>';
}
```

### 5. Add `renderGroupedExpanded` function

Add after `renderGroup`:

```js
function renderGroupedExpanded(group){
    if(group.length===1)return renderExpandedBody(group[0]);

    const uid='grp-'+Math.random().toString(36).slice(2,8);
    let tabs='<div class="channel-tabs">';
    group.forEach((p,i)=>{
        tabs+='<button class="channel-tab'+(i===0?' active':'')+'" onclick="event.stopPropagation();switchChannelTab(\''+uid+'\','+i+',this)">'
            +'<span class="ch-icon">'+channelIcon(p.target_channel)+'</span>'
            +esc(p.target_channel||'general')
            +'</button>';
    });
    tabs+='</div>';

    let panels='';
    group.forEach((p,i)=>{
        panels+='<div class="channel-panel-'+uid+'" style="'+(i>0?'display:none':'')+'">'
            +renderExpandedBody(p)
            +'</div>';
    });

    return tabs+panels;
}

function switchChannelTab(uid, idx, btn){
    btn.closest('.channel-tabs').querySelectorAll('.channel-tab').forEach((t,i)=>t.classList.toggle('active',i===idx));
    const panels=btn.closest('.expand').querySelectorAll('.channel-panel-'+uid);
    panels.forEach((p,i)=>p.style.display=i===idx?'':'none');
}
```

## What NOT to do

- Do **NOT** modify `internal/store/store.go` — no backend changes needed
- Do **NOT** modify `internal/handler/search.go` — the API response stays flat
- Do **NOT** use Solr's `group=true` parameter — client-side grouping is simpler and preserves facets/highlights
- Do **NOT** remove the existing `renderCompact` function — it's still used by `renderGroup` as the base

## Verification

```powershell
# No Go changes, just frontend — verify build still passes
go build ./...
```

After starting the server, to test grouping you need multiple promotions for the same repo with different channels. Use the Generate feature:
1. Generate a promotion for a repo with channel "twitter"
2. Generate another promotion for the same repo with channel "linkedin"
3. Generate a third for the same repo with channel "general"
4. Search or browse — the three should appear as a single card with "3 channels" badge
5. Expand the card — channel tabs should appear (🐦 twitter | 💼 linkedin | 📢 general)
6. Click each tab — content should switch to show that channel's promotion
7. A repo with only one promotion should render normally (no tabs, no group count)
