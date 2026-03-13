# Prompt: Refine Analysis Agent System Prompt Based on Observed Outputs

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. Phases 0–5 are complete: the Analysis Agent is wired into the generate flow, its output is stored in SQLite, and the frontend renders it in a "Why this repo?" panel.

Now I need to review real analysis outputs across several repos and refine the prompt embedded in `internal/agent/analysis.go` to improve quality.

The full intent document is at `docs/intent-for-analysis-agent.md` — read it for high-level context.
The Gradient model instructions are at `docs/analysis-agent-model-instructions.md`.

## Current state

### Embedded prompt template in `internal/agent/analysis.go`

The `analysisPromptTemplate` const contains the user-message prompt template. It includes:
- Instructions to output only JSON
- Rules about basing statements on input, avoiding buzzwords
- Guidance for interpreting traffic metrics at different levels
- Guidance for sparse repos (empty README, few stars)
- Guidance for `risk_or_limitations` (say "none clearly indicated" for mature repos)

### Model instructions at `docs/analysis-agent-model-instructions.md`

The system prompt for the Gradient agent, defining the agent's role, input/output schemas, and style constraints.

## Your task

### Step 1 — Generate analysis for several test repos

Start the server with both agents configured. Generate promotions for these repos to see real analysis outputs:

```bash
# Repo 1: A well-established repo with README, topics, and stars
curl -s -X POST http://localhost:8080/api/generate \
  -H "Content-Type: application/json" \
  -d '{"repo_url":"https://github.com/AcaciaMan/acacia-log","target_channel":"twitter"}' | jq '.analysis'

# Repo 2: Another AcaciaMan repo (to compare consistency)
curl -s -X POST http://localhost:8080/api/generate \
  -H "Content-Type: application/json" \
  -d '{"repo_url":"https://github.com/AcaciaMan/village-square","target_channel":"general"}' | jq '.analysis'

# Repo 3: Default fallback (no URL — uses hardcoded village-square data)
curl -s -X POST http://localhost:8080/api/generate \
  -H "Content-Type: application/json" \
  -d '{"target_channel":"linkedin"}' | jq '.analysis'
```

If you have access to other test repos, try a few more — especially:
- A repo with minimal README or no topics
- A repo with many stars (use a popular public repo)

### Step 2 — Evaluate outputs against quality criteria

For each output, check:

1. **Verbosity**: Are items too long (>2 sentences)? The prompt says 1–2 short sentences per item.
2. **Buzzword usage**: Does the output use generic terms like "revolutionary", "cutting-edge", "seamlessly integrates"? These should be eliminated.
3. **Invented features**: Does it mention capabilities not present in the input data? This is the most critical failure mode.
4. **Sparse repos**: When README is short or empty, does the output acknowledge limited information, or does it pad with generic statements?
5. **Social proof calibration**: Do low-star repos get described accurately ("early-stage") vs. inflated ("growing community")?
6. **Risk inflation**: Are risks fabricated when none are obvious? The output should say "none clearly indicated" for mature repos, not invent generic risks like "limited documentation" when docs exist.
7. **Array lengths**: Are there 2–4 items per array field, as instructed?
8. **Positioning angles**: Are they actionable and specific, or generic platitudes?

### Step 3 — Refine the prompt template

Based on observed issues, edit `analysisPromptTemplate` in `internal/agent/analysis.go`. Common refinements:

**If outputs are too verbose:**
Add/strengthen:
```
- Keep each string item to ONE sentence, maximum 20 words. Brevity is a requirement.
```

**If buzzwords persist:**
Add:
```
- FORBIDDEN words: revolutionary, cutting-edge, seamlessly, robust, powerful, state-of-the-art, game-changing, innovative, next-generation, world-class. Use plain language instead.
```

**If sparse repos get padded:**
Strengthen the existing rule:
```
- If readme_text is empty or very short, acknowledge limited information explicitly. Output fewer items (minimum 1 per array field) rather than generating generic filler. It is better to say "insufficient data to determine key features" than to guess.
```

**If social proof is inflated for low-traction repos:**
Make thresholds clearer:
```
- For social_proof_signals, be brutally honest:
  - 0–5 stars: "very early stage, minimal public adoption"
  - 5–20 stars: "small user base, early traction"
  - 20–100 stars: "modest community interest"
  - 100+: describe proportionally
  - If views/clones are zero or absent, say "no recent traffic data available"
```

**If risks are fabricated:**
```
- For risk_or_limitations: only list risks that are clearly evidenced in the input. An absent README is a risk. An absent test suite is NOT a risk unless explicitly noted. When in doubt, output ["none clearly indicated from available data"].
```

### Step 4 — Optionally refine the Gradient model instructions

If the embedded prompt refinements aren't sufficient, also update `docs/analysis-agent-model-instructions.md` and re-apply the system prompt on the Gradient agent dashboard. The model instructions are the system prompt; the embedded template is the per-request user message. Both can be tuned.

Key model instruction refinements:
- Add the forbidden words list to the system prompt as well.
- Emphasize the "data-grounded only" requirement more strongly.
- Add: "When the input is sparse, produce shorter output. Do not compensate for missing data with assumptions."

### Step 5 — Re-test with the refined prompt

After editing the template, rebuild and restart:

```bash
go build ./... && go run ./cmd/server
```

Re-run the same curl commands from Step 1 and compare outputs. Check that:
- Outputs are shorter/more focused.
- Buzzwords are removed.
- Sparse repos get honest, brief analysis.
- Social proof is calibrated to actual metrics.

### Step 6 — Update the analysis-agent-model-instructions doc

If you changed the embedded prompt template, ensure `docs/analysis-agent-model-instructions.md` stays consistent. The model instructions doc should reflect the latest version of the system prompt applied on Gradient.

## What NOT to change

- Do not modify any Go types, structs, or function signatures.
- Do not modify the frontend.
- Do not modify the store or handler logic.
- Only change content strings: `analysisPromptTemplate`, `analysisOutputSchema` in `analysis.go`, and optionally `docs/analysis-agent-model-instructions.md`.

## Deliverables

- Refined `analysisPromptTemplate` in `internal/agent/analysis.go`.
- Optionally refined `docs/analysis-agent-model-instructions.md`.
- Brief notes on what issues were found and what was changed (as comments in the code or as output to this session).
