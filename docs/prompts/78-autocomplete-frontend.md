# Prompt: Frontend Autocomplete Dropdown with Debounced Input

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. The app stores AI-generated promotional content in **Apache Solr 10** and exposes full-text search.

This is **Phase 4, prompt 4 of 5**. Prompts 75–77 added indexed analysis fields, updated Save/Search, configured the Solr Suggester, and wired a `GET /api/suggest?q=prefix` endpoint that returns `[{"term":"...","weight":N}, ...]`.

This prompt adds a dropdown autocomplete UI to the search input.

## Current state

### Search input section in `static/index.html`

```html
<div class="tab-content" id="panel-search">
<section class="search-section">
    <h2>Search &amp; Browse</h2>
    <div style="display:flex;gap:.5rem">
        <input type="text" id="search-q" placeholder="Search promotions… (leave empty to browse recent)" aria-label="Search promotions">
        <button id="search-btn" aria-label="Search promotions">Search</button>
    </div>
    <div id="search-status" class="status"></div>
    ...
```

### JavaScript — search input binding

```js
const searchQ=document.getElementById('search-q');
searchBtn.onclick=doSearch;
searchQ.onkeydown=e=>{if(e.key==='Enter')doSearch()};
```

### API contract — `GET /api/suggest?q=prefix`

Returns a JSON array:
```json
[
  {"term": "Lightning Fast CLI Framework for Go Developers", "weight": 500},
  {"term": "Python ML Pipeline Helper", "weight": 120}
]
```

## Your task

Add autocomplete functionality to the search input in `static/index.html`:
1. CSS for the suggestion dropdown
2. HTML container for suggestions
3. Debounced input handler calling `/api/suggest`
4. Keyboard navigation (arrow keys, Enter, Escape)
5. Mouse click selection

## Requirements

### 1. Add CSS for the autocomplete dropdown

Add the following styles to the `<style>` block, after the existing `.compact-card` rule:

```css
.autocomplete-wrap{position:relative}
.autocomplete-list{position:absolute;top:100%;left:0;right:0;background:#fff;border:1px solid #ccc;border-top:none;border-radius:0 0 6px 6px;max-height:240px;overflow-y:auto;z-index:100;display:none;box-shadow:0 4px 12px rgba(0,0,0,.1)}
.autocomplete-list.visible{display:block}
.autocomplete-item{padding:.5rem .75rem;cursor:pointer;font-size:.9rem;border-bottom:1px solid #f0f0f0}
.autocomplete-item:last-child{border-bottom:none}
.autocomplete-item:hover,.autocomplete-item.active{background:#f0f7ff;color:#0969da}
.autocomplete-item .ac-weight{float:right;font-size:.75rem;color:#888}
```

### 2. Wrap the search input and add the dropdown container

Replace the search input `<div>`:

```html
<div style="display:flex;gap:.5rem">
    <input type="text" id="search-q" placeholder="Search promotions… (leave empty to browse recent)" aria-label="Search promotions">
    <button id="search-btn" aria-label="Search promotions">Search</button>
</div>
```

With:

```html
<div style="display:flex;gap:.5rem">
    <div class="autocomplete-wrap" style="flex:1">
        <input type="text" id="search-q" placeholder="Search promotions… (leave empty to browse recent)" aria-label="Search promotions" autocomplete="off">
        <div id="suggest-list" class="autocomplete-list" role="listbox"></div>
    </div>
    <button id="search-btn" aria-label="Search promotions">Search</button>
</div>
```

### 3. Add JavaScript for autocomplete behavior

Add the following code in the `<script>` block, right after the `searchQ.onkeydown=...` line and before the `async function doSearch()` definition:

```js
/* ---- autocomplete / suggest ---- */
let suggestTimer=null;
let suggestIdx=-1;
const suggestList=document.getElementById('suggest-list');

searchQ.addEventListener('input',function(){
    clearTimeout(suggestTimer);
    const v=this.value.trim();
    if(v.length<2){hideSuggestions();return}
    suggestTimer=setTimeout(()=>fetchSuggestions(v),300);
});

searchQ.addEventListener('keydown',function(e){
    if(!suggestList.classList.contains('visible'))return;
    const items=suggestList.querySelectorAll('.autocomplete-item');
    if(e.key==='ArrowDown'){e.preventDefault();suggestIdx=Math.min(suggestIdx+1,items.length-1);highlightSuggest(items)}
    else if(e.key==='ArrowUp'){e.preventDefault();suggestIdx=Math.max(suggestIdx-1,-1);highlightSuggest(items)}
    else if(e.key==='Enter'&&suggestIdx>=0){e.preventDefault();selectSuggestion(items[suggestIdx].dataset.term)}
    else if(e.key==='Escape'){hideSuggestions()}
});

document.addEventListener('click',function(e){
    if(!e.target.closest('.autocomplete-wrap'))hideSuggestions();
});

async function fetchSuggestions(prefix){
    try{
        const res=await fetch('/api/suggest?q='+encodeURIComponent(prefix));
        if(!res.ok)return;
        const data=await res.json();
        if(!data||!data.length){hideSuggestions();return}
        suggestIdx=-1;
        suggestList.innerHTML=data.map((s,i)=>{
            const termEsc=esc(s.term);
            return '<div class="autocomplete-item" role="option" data-term="'+termEsc.replace(/"/g,'&quot;')+'" onclick="selectSuggestion(this.dataset.term)">'+termEsc+'<span class="ac-weight">⭐ '+s.weight+'</span></div>';
        }).join('');
        suggestList.classList.add('visible');
    }catch(e){hideSuggestions()}
}

function highlightSuggest(items){
    items.forEach((el,i)=>{el.classList.toggle('active',i===suggestIdx)});
    if(suggestIdx>=0&&items[suggestIdx])items[suggestIdx].scrollIntoView({block:'nearest'});
}

function selectSuggestion(term){
    searchQ.value=term;
    hideSuggestions();
    doSearch();
}

function hideSuggestions(){
    suggestList.classList.remove('visible');
    suggestList.innerHTML='';
    suggestIdx=-1;
}
```

### 4. Update the existing `searchQ.onkeydown` handler

The existing `searchQ.onkeydown` that fires `doSearch` on Enter should be replaced because the new `keydown` listener handles Enter when suggestions are visible, and we still want Enter to search when suggestions are hidden.

Replace:

```js
searchQ.onkeydown=e=>{if(e.key==='Enter')doSearch()};
```

With:

```js
searchQ.onkeydown=e=>{if(e.key==='Enter'&&!suggestList.classList.contains('visible'))doSearch()};
```

This ensures Enter searches only when the suggestion dropdown is hidden. When visible, the autocomplete `keydown` handler takes over.

## What NOT to do

- Do **NOT** modify any Go files — backend is complete from prompt 77
- Do **NOT** add external libraries (jQuery UI, etc.) — plain JS only
- Do **NOT** call `/api/suggest` for inputs shorter than 2 characters — too many useless results
- Do **NOT** add a separate debounce utility function — inline `setTimeout`/`clearTimeout` is sufficient

## Verification

After making changes:
1. Open `http://localhost:8080` in a browser
2. Switch to the Search tab
3. Type 2+ characters in the search input — a dropdown should appear after ~300ms (if matching suggestions exist in Solr)
4. Press Arrow Down/Up to navigate — highlighted item should change
5. Press Enter on a highlighted item — search input should be filled and search should execute
6. Press Escape — dropdown should close
7. Click outside the input — dropdown should close
8. Click a suggestion — search input should be filled and search should execute
