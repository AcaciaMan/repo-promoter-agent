# Prompt: Add Copy Buttons and Tooltips to Analysis Panel

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. Phases 0–5 are complete and Phase 6 prompt refinement has been done (prompt 42). Now I need to add UX polish to the analysis panel: copy buttons for text blocks (matching the existing promo content UX) and tooltips explaining analysis fields.

The full intent document is at `docs/intent-for-analysis-agent.md` — read it for high-level context.

## Current state

### `static/index.html` — analysis panel rendering

The `renderAnalysis()` function renders a green-bordered "Why this repo?" panel with 7 fields. Each field has a `<strong>` label and either a `<p>` (for value proposition) or `<ul>` (for lists).

```javascript
function renderAnalysis(analysis){
    if(!analysis)return '';
    let h='<div class="analysis-panel">';
    h+='<h4>🔍 Why This Repo?</h4>';
    h+='<div class="analysis-label">AI-generated analysis</div>';
    if(analysis.primary_value_proposition){
        h+='<div class="analysis-field"><strong>Value Proposition</strong><p>'+esc(analysis.primary_value_proposition)+'</p></div>';
    }
    // ... similar for each array field ...
    h+='</div>';
    return h;
}
```

### Existing copy button pattern

The codebase already has copy buttons for tweets and LinkedIn posts:

```javascript
function copyText(text,btn){
    navigator.clipboard.writeText(text).then(()=>{
        btn.textContent='Copied!';
        setTimeout(()=>{btn.textContent='Copy'},1500)
    })
}

// Tweet copy button (inline, next to content):
function tweetCopyBtn(){
    return '<button class="btn-sm" aria-label="Copy tweet" onclick="event.stopPropagation();copyText(this.previousElementSibling.textContent,this)">Copy</button>';
}
```

CSS for `.btn-sm`:
```css
.btn-sm{font-size:.75rem;padding:.25rem .5rem;background:#555;border:none}
.btn-sm:hover{background:#777}
```

## Your task

### 1. Add a "Copy" button to the Value Proposition field

The primary value proposition is a single sentence — useful for pasting into pitches, emails, or social posts. Add a small copy button next to it, matching the existing tweet copy UI.

Update the value proposition rendering in `renderAnalysis()`:

```javascript
if(analysis.primary_value_proposition){
    h+='<div class="analysis-field"><strong>Value Proposition</strong>'
      +'<div class="analysis-copyable"><span>'+esc(analysis.primary_value_proposition)+'</span>'
      +'<button class="btn-sm" aria-label="Copy value proposition" onclick="event.stopPropagation();copyText(this.previousElementSibling.textContent,this)">Copy</button>'
      +'</div></div>';
}
```

### 2. Add a "Copy All" button for the entire analysis panel

Add a button at the bottom of the analysis panel that copies all analysis text as formatted plain text. This is useful when the user wants to paste the full analysis into a document or chat.

At the end of `renderAnalysis()`, before the closing `</div>`:

```javascript
h+='<div style="text-align:right;margin-top:.5rem">'
  +'<button class="btn-sm" aria-label="Copy all analysis" onclick="event.stopPropagation();copyAnalysis(this)">Copy All</button>'
  +'</div>';
```

Add a new helper function:

```javascript
function copyAnalysis(btn){
    const panel=btn.closest('.analysis-panel');
    if(!panel)return;
    const fields=panel.querySelectorAll('.analysis-field');
    let text='';
    fields.forEach(f=>{
        const label=f.querySelector('strong');
        const items=f.querySelectorAll('li');
        const p=f.querySelector('p');
        if(label)text+=label.textContent+':\n';
        if(p)text+=p.textContent+'\n';
        if(items.length){items.forEach(li=>{text+='- '+li.textContent+'\n'})}
        text+='\n';
    });
    navigator.clipboard.writeText(text.trim()).then(()=>{
        btn.textContent='Copied!';
        setTimeout(()=>{btn.textContent='Copy All'},1500);
    });
}
```

### 3. Add tooltips to analysis field labels

Add `title` attributes to the `<strong>` labels explaining what each field represents. This helps users understand the analysis without needing documentation.

Update each field's `<strong>` tag in `renderAnalysis()`:

| Field | Tooltip text |
|-------|-------------|
| Value Proposition | `"The core problem this repo solves, in one sentence"` |
| Ideal Audience | `"Who would benefit most from using this repo"` |
| Key Features | `"Main capabilities framed as user benefits"` |
| Differentiators | `"What sets this repo apart from alternatives"` |
| Risks & Limitations | `"Potential concerns or gaps to be aware of"` |
| Social Proof | `"Traction signals based on stars, traffic, and community"` |
| Positioning Angles | `"Suggested ways to pitch this repo to different audiences"` |

Example for the value proposition field:

```javascript
h+='<div class="analysis-field"><strong title="The core problem this repo solves, in one sentence">Value Proposition</strong>...';
```

Apply the same pattern to all 7 fields.

### 4. Add CSS for the copyable layout

Add a style for the value proposition copy button layout:

```css
.analysis-copyable {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: .5rem;
    background: #fff;
    border: 1px solid #d1fae5;
    border-radius: 4px;
    padding: .4rem .6rem;
    margin-top: .15rem;
    font-size: .9rem;
}

.analysis-copyable span {
    flex: 1;
}
```

### 5. Add tooltip cursor for labeled fields

Add a CSS rule so users know they can hover over labels:

```css
.analysis-field strong[title] {
    cursor: help;
    border-bottom: 1px dotted #999;
}
```

The dotted underline is a standard convention indicating a tooltip is available.

## What NOT to change

- Do not modify any backend code.
- Do not modify the existing promo content rendering (`renderFull`, tweet/LinkedIn copy buttons).
- Do not modify the search card rendering (`renderCompact`, `renderExpandedBody`) — the analysis panel in expanded cards will inherit these changes automatically since it calls the same `renderAnalysis()` function.
- Do not modify `renderAnalysisCompact()` (the search card snippet) — it's a one-liner, no copy button needed.
- Do not modify the loading UX (prompt 39).

## Verification

1. Open `http://localhost:8080` with both agents configured.
2. Generate a promotion for a repo.
3. In the "Why this repo?" panel:
   - **Value Proposition** should have a "Copy" button on the right. Click it — verify it copies the text and shows "Copied!" briefly.
   - **"Copy All" button** at the bottom of the panel. Click it — verify it copies all analysis as formatted plain text with field labels and bullet points.
   - **Hover over field labels** (e.g., "Value Proposition", "Ideal Audience") — a tooltip should appear explaining the field. Labels should have a dotted underline.
4. In the Search view, click a card to expand it — the analysis panel should also have copy buttons and tooltips (since it reuses `renderAnalysis()`).
5. Test with analysis disabled — no panel, no errors.
6. Open browser DevTools Console — no JavaScript errors.
