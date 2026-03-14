# Prompt: Highlighting & Sort — Frontend Integration

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. This is **Phase 3, prompt 3 of 4** for search highlighting & sort options.

**Prerequisites**: Prompts 71–72 are complete. The backend now:
- Returns `highlights` in the search API response: `{ "results": [...], "facets": {...}, "highlights": { "https://github.com/owner/repo": { "headline": "Fast <mark>CLI</mark> Framework", "summary": "A <mark>CLI</mark> tool..." } } }`
- Accepts `?sort=relevance|newest|stars|views` query parameter
- Highlights are only returned for full-text searches (not for browse/list)

## Current frontend search flow (`static/index.html`)

The `doSearch()` function fetches from `/api/search`, gets `d.results` and `d.facets`, renders facets and result cards. It does **not** use `d.highlights` or send a `sort` parameter.

The `renderCompact(p)` function renders cards using `esc(p.headline)` and `esc(p.summary)` — which escapes all HTML, so even if highlights were present in the data, `<mark>` tags would be escaped and displayed as literal text.

## Your task

Update `static/index.html` to:
1. Add a sort dropdown above results.
2. Send the selected sort option in the search API URL.
3. Store highlights from the API response and use them to render highlighted text in search result cards.

## Requirements

### 1. Add sort state variable

Near the existing `activeFilters` state:

```javascript
let currentSort = 'relevance';
let currentHighlights = {};
```

### 2. Add sort dropdown HTML

Add a sort control between `<div id="search-facets">` and `<div id="search-results">`. The exact location in the HTML:

```html
    <div id="search-facets" style="margin:.5rem 0"></div>
    <div id="search-sort" style="margin:.5rem 0;display:flex;align-items:center;gap:.5rem">
        <label for="sort-select" style="font-size:.85rem;color:#555;font-weight:600;margin:0">Sort by:</label>
        <select id="sort-select" style="padding:.25rem .5rem;font-size:.85rem;border:1px solid #ccc;border-radius:4px" onchange="currentSort=this.value;doSearch()">
            <option value="relevance">Relevance</option>
            <option value="newest">Newest</option>
            <option value="stars">Most Stars</option>
            <option value="views">Most Views</option>
        </select>
    </div>
    <div id="search-results"></div>
```

### 3. Update `doSearch()` to send sort and store highlights

Update the URL construction to include the sort parameter and store highlights from the response:

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
        if(currentSort&&currentSort!=='relevance')params.set('sort',currentSort);
        const url='/api/search'+(params.toString()?'?'+params.toString():'');
        const res=await fetch(url);
        if(res.status===429){const e=await res.json().catch(()=>({}));const secs=e.retry_after_seconds||60;throw new Error('Search rate limit reached. Please wait '+secs+' seconds.')}
        if(!res.ok){const e=await res.json().catch(()=>({}));throw new Error(e.error||res.statusText)}
        const d=await res.json();
        currentHighlights=d.highlights||{};
        renderActiveFilters();
        renderFacets(d.facets);
        if(!d.results||d.results.length===0){searchResults.innerHTML='<p style="color:#888">No results found.</p>';return}
        searchResults.innerHTML=d.results.map(renderCompact).join('');
    }catch(e){searchError.textContent='Error: '+e.message}
    finally{searchStatus.textContent='';searchBtn.disabled=false}
}
```

Key changes:
- Added `if(currentSort&&currentSort!=='relevance')params.set('sort',currentSort)` — sends sort only if non-default to keep URLs clean.
- Added `currentHighlights=d.highlights||{}` — stores highlights for use in rendering.

### 4. Add a safe highlight rendering function

Solr highlights contain only `<mark>` and `</mark>` tags. We need to render these as HTML, but must sanitize to prevent XSS if the content somehow contains other tags.

```javascript
function safeHighlight(html){
    // Allow only <mark> and </mark> tags; escape everything else
    return html
        .replace(/<mark>/g, '\x00MARK_OPEN\x00')
        .replace(/<\/mark>/g, '\x00MARK_CLOSE\x00')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/\x00MARK_OPEN\x00/g, '<mark>')
        .replace(/\x00MARK_CLOSE\x00/g, '</mark>');
}
```

This approach:
1. Temporarily replaces `<mark>` / `</mark>` with null-byte delimited placeholders.
2. Escapes all remaining `<` and `>` (prevents any script/image/iframe injection).
3. Restores the `<mark>` tags.

### 5. Add a helper to get highlighted or plain text for a field

```javascript
function hlField(repoUrl, field, plainValue){
    const docHl = currentHighlights[repoUrl];
    if(docHl && docHl[field]){
        return safeHighlight(docHl[field]);
    }
    return esc(plainValue);
}
```

### 6. Update `renderCompact` to use highlighted text

Update the `renderCompact(p)` function. Replace the headline and first-tweet lines to use `hlField` for fields that have highlights:

Current:
```javascript
+'<p>'+esc(p.headline)+'</p>'
```

New:
```javascript
+'<p>'+hlField(p.repo_url,'headline',p.headline)+'</p>'
```

**Important**: Only apply highlighting to fields listed in `hl.fl` from the backend. For `renderCompact`, the relevant fields are:
- `headline` — the one-liner pitch shown in the compact card

The repo name link and other metadata should remain plain-escaped text (not highlighted).

### 7. Update `renderExpandedBody` to use highlighted text

In the expanded card view, use highlights for `summary`:

Current:
```javascript
let h='<p>'+esc(d.summary)+'</p>';
```

New:
```javascript
let h='<p>'+hlField(d.repo_url,'summary',d.summary)+'</p>';
```

### 8. Add CSS for `<mark>` highlight styling

Add to the existing `<style>` block:

```css
mark{background:#fff3b0;color:inherit;padding:.05rem .15rem;border-radius:2px}
```

This gives highlighted terms a soft yellow background that feels natural and is distinct from other styling.

### 9. Sync sort dropdown with state

When the page loads and `doSearch()` runs, ensure the dropdown reflects the current sort value. Add after the sort-select element is available:

```javascript
document.getElementById('sort-select').value=currentSort;
```

Place this near the existing `searchBtn.onclick=doSearch;` line.

## Implementation notes

- Highlights are **only** present when the user performs a text search (`q` is non-empty). When browsing (empty query), `currentHighlights` will be `{}` and `hlField` will fall through to `esc()`.
- The sort dropdown always shows all options. When browsing without a query, "Relevance" behaves the same as "Newest" (both sort by `created_at desc` because there are no scores).
- The `safeHighlight` function is critical for security — it ensures no HTML other than `<mark>` tags can be injected.

## Verification

After applying the changes:

1. Run `go build ./...` — no Go changes, should still compile.
2. Open `http://localhost:8080` in a browser.
3. Go to Search tab, search for a term that exists in indexed data (e.g., a repo name or tag).
4. **Verify**: Matched terms in headline and summary should have a yellow background highlight.
5. **Verify**: The sort dropdown appears above results.
6. Change sort to "Most Stars" — results should reorder by star count.
7. Change sort to "Newest" — results should reorder by creation date.
8. Clear the search query and browse — highlights should be absent (plain text).

## Files to modify

- `static/index.html` — add sort dropdown, highlight rendering, `safeHighlight`/`hlField` functions, update `doSearch`/`renderCompact`/`renderExpandedBody`, add CSS

## Files NOT to modify

- `internal/store/store.go` — done in prompt 71
- `internal/handler/search.go` — done in prompt 72
