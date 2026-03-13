# Prompt: Enhanced Loading UX for Two-Agent Generate Flow

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. After Phase 3 (prompt 36), the `/api/generate` endpoint now calls **two** sequential AI agents: the Analysis Agent (~5–15s) and then the Promotion Agent (~10–15s). Total wait time is roughly 15–30 seconds.

The current frontend shows only a static "Generating…" message during this entire wait. This needs to be improved so the user understands work is happening and doesn't think the page is stuck.

The full intent document is at `docs/intent-for-analysis-agent.md` — read it for high-level context.

## Current state

### `static/index.html`

The frontend is a single-page HTML file (~203 lines) with inline CSS and JS. Key pieces:

**Generate button handler (`genBtn.onclick`):**
```javascript
genBtn.onclick=async()=>{
    genStatus.textContent='Generating…';
    genInfo.textContent='';genError.textContent='';genResult.classList.add('hidden');genResult.innerHTML='';genBtn.disabled=true;
    try{
        const body={target_channel:..., target_audience:...};
        const url=...;
        if(url)body.repo_url=url;
        const res=await fetch('/api/generate',{method:'POST',...});
        if(!res.ok){...throw...}
        const d=await res.json();
        genResult.innerHTML=renderFull(d);
        genResult.classList.remove('hidden');
        ...
    }catch(e){genError.textContent='Error: '+e.message}
    finally{genStatus.textContent='';genBtn.disabled=false}
};
```

**Status element:**
```html
<div id="gen-status" class="status"></div>
```

**CSS for status:**
```css
.status{color:#555;font-style:italic;margin:.5rem 0}
```

### The problem

The user sees "Generating…" for 15–30 seconds with no feedback about what's happening. Since the backend handles both agents sequentially in a single POST request, the frontend can't get real-time progress from the server. But we can simulate a multi-step progress indicator client-side.

## Your task

### 1. Add a multi-step progress indicator

Replace the static "Generating…" text with a timed progress sequence that gives the user a sense of forward motion. Use a simple interval-based approach:

**Progression (client-side, approximate):**
- 0s: "Analyzing repository…" (Analysis Agent is running)
- 8s: "Still analyzing…" (in case analysis is slow)
- 15s: "Generating promotion…" (Promotion Agent should be running by now)
- 25s: "Almost there…" (in case promotion agent is slow)

**Implementation approach:**

```javascript
let progressInterval = null;

function startProgress() {
    const steps = [
        { delay: 0, text: '🔍 Analyzing repository…' },
        { delay: 8000, text: '🔍 Still analyzing…' },
        { delay: 15000, text: '✍️ Generating promotion…' },
        { delay: 25000, text: '⏳ Almost there…' },
    ];
    let stepIndex = 0;
    genStatus.textContent = steps[0].text;

    progressInterval = setInterval(() => {
        stepIndex++;
        if (stepIndex < steps.length) {
            genStatus.textContent = steps[stepIndex].text;
        }
    }, /* use step deltas, not fixed interval */);
}

function stopProgress() {
    if (progressInterval) {
        clearInterval(progressInterval);
        progressInterval = null;
    }
    genStatus.textContent = '';
}
```

A cleaner approach is to use `setTimeout` for each step rather than `setInterval`, storing the timeout IDs so they can be cleared:

```javascript
let progressTimers = [];

function startProgress() {
    const steps = [
        { delay: 0, text: '🔍 Analyzing repository…' },
        { delay: 8000, text: '🔍 Still analyzing…' },
        { delay: 15000, text: '✍️ Generating promotion…' },
        { delay: 25000, text: '⏳ Almost there…' },
    ];
    steps.forEach(step => {
        const t = setTimeout(() => {
            genStatus.textContent = step.text;
        }, step.delay);
        progressTimers.push(t);
    });
}

function stopProgress() {
    progressTimers.forEach(clearTimeout);
    progressTimers = [];
    genStatus.textContent = '';
}
```

### 2. Update the generate button handler

Replace the old progress logic in `genBtn.onclick`:

**Before:**
```javascript
genStatus.textContent='Generating…';
```

**After:**
```javascript
startProgress();
```

**Before (in finally block):**
```javascript
genStatus.textContent='';
```

**After:**
```javascript
stopProgress();
```

### 3. Optional: add a subtle CSS animation

Add a pulsing dot or animation to the status text to make it feel more alive:

```css
.status {
    color: #555;
    font-style: italic;
    margin: .5rem 0;
}

@keyframes pulse {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.5; }
}

.status:not(:empty) {
    animation: pulse 1.5s ease-in-out infinite;
}
```

This makes any status text gently pulse, indicating ongoing activity.

## What NOT to change

- Do not modify any backend code.
- Do not modify the search section.
- Do not add the "Why this repo?" panel yet — that's the next prompt.
- Do not change `renderFull()`, `renderCompact()`, or any rendering functions.
- Keep the existing error handling and info note logic intact.

## Verification

1. Open `http://localhost:8080`.
2. Paste a repo URL and click Generate.
3. Watch the status text — it should progress through the steps over ~25s.
4. When the response arrives, the status text should clear immediately.
5. If generation fails quickly (bad URL, network error), the status should also clear.
6. Click Generate again — the progress should restart cleanly from step 1.
7. Test with no URL (default repo) — progress should work the same way.
