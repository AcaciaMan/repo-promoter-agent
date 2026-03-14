# Prompt: Faceted Search — Frontend Facet Sidebar and Filter Chips

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. This is **Phase 2, prompt 4 of 5** for faceted search and filtering.

**Prerequisites**: Prompts 66–68 are complete. The backend now:
- Returns `facets` in the search API response: `{ "results": [...], "count": N, "facets": { "tags": [{"value":"go","count":5},...], "target_channel": [{"value":"general","count":8},...] } }`
- Accepts filter query params: `?tag=go&tag=cli`, `?channel=twitter`, `?min_stars=100`
- These params are combined with the full-text query `q` (AND logic)

The app compiles and runs. The frontend in `static/index.html` does **not** yet use facets or send filter params.

## Current frontend search flow (`static/index.html`)

The search tab has:
- A text `<input id="search-q">` for the query
- A `<button id="search-btn">` to trigger search
- A `<div id="search-results">` for result cards

The `doSearch()` function:
```javascript
async function doSearch(){
    const q=searchQ.value.trim();
    searchStatus.textContent='Searching…';searchError.textContent='';searchResults.innerHTML='';searchBtn.disabled=true;
    try{
        const url='/api/search'+(q?'?q='+encodeURIComponent(q):'');
        const res=await fetch(url);
        // ... error handling ...
        const d=await res.json();
        if(!d.results||d.results.length===0){searchResults.innerHTML='<p style="color:#888">No results found.</p>';return}
        searchResults.innerHTML=d.results.map(renderCompact).join('');
    }catch(e){searchError.textContent='Error: '+e.message}
    finally{searchStatus.textContent='';searchBtn.disabled=false}
}
```

It ignores `d.facets` entirely and builds the URL without filter params.

## Your task

Update `static/index.html` to:
1. Track active filters in a JavaScript state object.
2. Render facet counts as clickable chips/buttons.
3. Show active filters as removable chips above results.
4. Include filter params in the search API URL.

## Requirements

### 1. Add filter state

Add a state object at the top of the search section script (near where `searchBtn`, `searchQ`, etc. are declared):

```javascript
// Active filters state
let activeFilters = { tags: [], channel: '', minStars: 0 };
```

### 2. Update `doSearch()` to include filters and render facets

Update the URL construction to include filter params:

```javascript
async function doSearch(){
    const q=searchQ.value.trim();
    searchStatus.textContent='Searching…';searchError.textContent='';searchResults.innerHTML='';searchBtn.disabled=true;
    try{
        const params=new URLSearchParams();
        if(q)params.set('q',q);
        activeFilters.tags.forEach(t=>params.append('tag',t));
        if(activeFilters.channel)params.set('channel',activeFilters.channel);
        if(activeFilters.minStars>0)params.set('min_stars',String(activeFilters.minStars));
        const url='/api/search'+(params.toString()?'?'+params.toString():'');
        const res=await fetch(url);
        if(res.status===429){const e=await res.json().catch(()=>({}));const secs=e.retry_after_seconds||60;throw new Error('Search rate limit reached. Please wait '+secs+' seconds.')}
        if(!res.ok){const e=await res.json().catch(()=>({}));throw new Error(e.error||res.statusText)}
        const d=await res.json();
        renderActiveFilters();
        renderFacets(d.facets);
        if(!d.results||d.results.length===0){searchResults.innerHTML='<p style="color:#888">No results found.</p>';return}
        searchResults.innerHTML=d.results.map(renderCompact).join('');
    }catch(e){searchError.textContent='Error: '+e.message}
    finally{searchStatus.textContent='';searchBtn.disabled=false}
}
```

### 3. Add HTML containers for facets and active filters

In the search panel section of the HTML, add two new `<div>` elements **between** the search input row and `<div id="search-results">`:

```html
<div id="active-filters" style="display:flex;flex-wrap:wrap;gap:.3rem;margin:.5rem 0"></div>
<div id="search-facets" style="margin:.5rem 0"></div>
```

The exact location is after the search-error div and before search-results:

```html
    <div id="search-status" class="status"></div>
    <div id="search-error" class="error"></div>
    <div id="active-filters" style="display:flex;flex-wrap:wrap;gap:.3rem;margin:.5rem 0"></div>
    <div id="search-facets" style="margin:.5rem 0"></div>
    <div id="search-results"></div>
```

### 4. Add CSS for facet styling

Add these styles to the existing `<style>` block:

```css
.facet-group{margin-bottom:.5rem}
.facet-group strong{font-size:.85rem;color:#555;margin-right:.5rem}
.facet-chip{display:inline-block;background:#e8f0fe;color:#1a73e8;border:1px solid #c2d7f5;border-radius:12px;padding:.15rem .6rem;font-size:.8rem;cursor:pointer;margin:.15rem .2rem;transition:background .15s}
.facet-chip:hover{background:#c2d7f5}
.facet-chip.active{background:#1a73e8;color:#fff;border-color:#1a73e8}
.filter-chip{display:inline-flex;align-items:center;gap:.3rem;background:#fef3c7;color:#92400e;border:1px solid #fcd34d;border-radius:12px;padding:.15rem .6rem;font-size:.8rem}
.filter-chip button{background:none;border:none;color:#92400e;cursor:pointer;font-size:.9rem;padding:0;line-height:1}
.filter-chip button:hover{color:#c00}
```

### 5. Add `renderFacets()` function

```javascript
function renderFacets(facets){
    const el=document.getElementById('search-facets');
    if(!facets||(!facets.tags&&!facets.target_channel)){el.innerHTML='';return}
    let h='';
    if(facets.tags&&facets.tags.length){
        h+='<div class="facet-group"><strong>Tags:</strong>';
        facets.tags.forEach(f=>{
            const isActive=activeFilters.tags.includes(f.value);
            h+='<span class="facet-chip'+(isActive?' active':'')+'" onclick="toggleTagFilter(\''+esc(f.value).replace(/'/g,"\\'")+'\')">'+esc(f.value)+' ('+f.count+')</span>';
        });
        h+='</div>';
    }
    if(facets.target_channel&&facets.target_channel.length){
        h+='<div class="facet-group"><strong>Channel:</strong>';
        facets.target_channel.forEach(f=>{
            const isActive=activeFilters.channel===f.value;
            h+='<span class="facet-chip'+(isActive?' active':'')+'" onclick="toggleChannelFilter(\''+esc(f.value).replace(/'/g,"\\'")+'\')">'+esc(f.value)+' ('+f.count+')</span>';
        });
        h+='</div>';
    }
    el.innerHTML=h;
}
```

### 6. Add `renderActiveFilters()` function

```javascript
function renderActiveFilters(){
    const el=document.getElementById('active-filters');
    let h='';
    activeFilters.tags.forEach(t=>{
        h+='<span class="filter-chip">tag: '+esc(t)+' <button onclick="removeTagFilter(\''+esc(t).replace(/'/g,"\\'")+'\')">×</button></span>';
    });
    if(activeFilters.channel){
        h+='<span class="filter-chip">channel: '+esc(activeFilters.channel)+' <button onclick="removeChannelFilter()">×</button></span>';
    }
    if(activeFilters.minStars>0){
        h+='<span class="filter-chip">stars ≥ '+activeFilters.minStars+' <button onclick="removeStarsFilter()">×</button></span>';
    }
    if(h){
        h+='<span class="filter-chip" style="background:#fee2e2;color:#991b1b;border-color:#fca5a5;cursor:pointer" onclick="clearAllFilters()">Clear all ×</span>';
    }
    el.innerHTML=h;
}
```

### 7. Add filter toggle/remove functions

```javascript
function toggleTagFilter(tag){
    const idx=activeFilters.tags.indexOf(tag);
    if(idx>=0)activeFilters.tags.splice(idx,1);
    else activeFilters.tags.push(tag);
    doSearch();
}
function toggleChannelFilter(ch){
    activeFilters.channel=activeFilters.channel===ch?'':ch;
    doSearch();
}
function removeTagFilter(tag){
    activeFilters.tags=activeFilters.tags.filter(t=>t!==tag);
    doSearch();
}
function removeChannelFilter(){
    activeFilters.channel='';
    doSearch();
}
function removeStarsFilter(){
    activeFilters.minStars=0;
    doSearch();
}
function clearAllFilters(){
    activeFilters={tags:[],channel:'',minStars:0};
    doSearch();
}
```

### 8. Implementation notes

- Clicking a facet chip toggles it on/off (adds/removes from `activeFilters`) and re-runs the search.
- Active facet chips have a highlighted style (`.active` class).
- Active filters are shown as yellow chips with × buttons above results.
- "Clear all" chip appears when any filters are active.
- The `esc()` function (already defined) is used to prevent XSS in facet values.
- When filters are active, facet counts in the response reflect the filtered subset (Solr recomputes facets after `fq` is applied).

## Verification

After applying the changes:

1. Run `go build ./...` — must compile (no Go changes in this prompt).
2. Open the app in a browser at `http://localhost:8080`.
3. Go to the Search tab — leave the query empty and click Search.
4. **Facet chips** should appear below the search bar showing tag names with counts and channel names with counts.
5. Click a tag chip — it should highlight, a yellow filter chip should appear, and results should filter to only that tag.
6. Click the × on the filter chip — filter should be removed and results restored.
7. Click "Clear all" — all filters should be removed.

## Files to modify

- `static/index.html` — add filter state, facet/filter rendering, filter toggle functions, CSS, HTML containers

## Files NOT to modify

- `internal/store/store.go` — done in prompts 66–67
- `internal/handler/search.go` — done in prompt 68
