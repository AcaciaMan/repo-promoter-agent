# Prompt: Manual Smoke Test and Bug Fix Pass

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. The application is functionally complete:

- **Go backend** at `cmd/server/main.go`
- **Agent client** at `internal/agent/client.go` — calls DigitalOcean Gradient AI
- **GitHub client** at `internal/github/client.go` — fetches real repo data
- **Storage** at `internal/store/store.go` — SQLite + FTS5
- **Handlers** at `internal/handler/generate.go` and `internal/handler/search.go`
- **Frontend** at `static/index.html`
- **Env** via `.env` with `AGENT_ENDPOINT`, `AGENT_ACCESS_KEY`, `PORT`, `DB_PATH`

## Your task

Conduct a **thorough code review and smoke test preparation**. You will NOT actually run the server, but you will:

1. Read all source files carefully.
2. Identify bugs, edge cases, and issues that would surface during manual testing.
3. Fix them.
4. Write a comprehensive smoke test checklist.

## Part 1: Code review — check for these specific issues

### Backend

1. **Compilation**: Does `go build ./...` succeed? Check for import issues, type mismatches, missing methods.

2. **Agent response parsing**: The agent returns JSON as a string inside `choices[0].message.content`. Verify:
   - The `stripMarkdownFences` regex works correctly for both ` ```json\n{...}\n``` ` and ` ```\n{...}\n``` `.
   - The fallback JSON extraction (find first `{`, last `}`) is safe.
   - What happens if the agent returns an empty string?

3. **GitHub URL parsing**: Does `parseGitHubURL` handle:
   - `https://github.com/owner/repo`
   - `https://github.com/owner/repo/`
   - `https://github.com/owner/repo/tree/main` (should still extract owner/repo)
   - `github.com/owner/repo` (no scheme)
   - Invalid URLs (empty string, random text, non-GitHub URLs)?

4. **README base64 decoding**: GitHub returns base64 with `\n` line breaks in the content field. Verify the decoding handles this. Check if `strings.ReplaceAll(rr.Content, "\n", "")` before `base64.StdEncoding.DecodeString` is correct (it should be — GitHub uses standard base64 with embedded newlines).

5. **SQLite FTS triggers**: Verify the trigger SQL is correct — especially that they fire on INSERT and the FTS table stays in sync.

6. **FTS query sanitization**: Does `sanitizeFTSQuery` handle:
   - Normal queries: `"go web framework"`
   - Empty queries
   - Queries with special chars: `"c++ library"`, `"node.js"`
   - Single characters: `"a"`

7. **`writeError` collision**: Both `generate.go` and `search.go` are in the `handler` package. If both define `writeError`, there's a compile error. Check if `writeError` is defined once or duplicated.

8. **Promotion JSON marshaling**: The `Promotion` struct has `[]string` fields that are stored as JSON text in SQLite. Verify the Save → List round-trip preserves the data correctly (marshal to JSON for insert, unmarshal on scan).

9. **Context propagation**: Verify all HTTP calls (agent, GitHub, SQLite) use `r.Context()` from the HTTP request, so cancellation works.

10. **HTTP client timeouts**: Agent client has 60s, GitHub client has 10s. Are these reasonable? The agent can be slow; the overall HTTP request to `/api/generate` has no server-side timeout. Should there be one?

### Frontend

11. **Fetch error handling**: What happens if the server is unreachable? Does the catch block show a useful message?

12. **JSON.parse failure**: If the server returns non-JSON (e.g., an HTML error page from a proxy), does the frontend handle it gracefully?

13. **XSS**: Verify all user-generated content (repo names, headlines, summaries, tweets, etc.) goes through `esc()` before being inserted into HTML.

14. **Created_at parsing**: The backend returns `created_at` as a Go `time.Time` marshaled to JSON. What format is it? Does `new Date(...)` parse it correctly in JS?

## Part 2: Fix all issues found

For each issue found, provide the fix:
- Modified Go code (show the full updated file if changes are significant, or just the diff if minor).
- Modified HTML if needed.

## Part 3: Smoke test checklist

Write a step-by-step manual smoke test I can follow. Format as a numbered markdown checklist.

### Prerequisites
- List exact commands to run before starting

### Test scenarios

**A. Basic startup**
- Start server, verify log output
- Open browser to `http://localhost:8080`
- Verify page loads with Generate and Search sections

**B. Generate with hardcoded sample (no URL)**
- Leave URL empty, click Generate
- Verify response renders with all sections (headline, summary, benefits, tweets, LinkedIn, CTA, tags)
- Verify "Show raw JSON" toggle works
- Verify copy buttons work on tweets and LinkedIn post
- Verify result has `id` and `created_at` (stored in DB)

**C. Generate with real GitHub URL**
- Enter a known public repo URL (e.g., `https://github.com/golang/go`)
- Click Generate
- Verify real repo data appears (correct repo name, actual star count referenced)
- Try an invalid URL (e.g., `https://github.com/nonexistent/repo-that-does-not-exist`)
- Verify error message is shown

**D. Search & Browse**
- After generating 2+ promotions, search for a keyword
- Verify results appear as compact cards
- Click a card to expand
- Verify expanded content matches what was generated
- Search for empty string — verify recent promotions are listed
- Search for nonsense — verify "No results found"

**E. Error scenarios**
- Stop the server, click Generate — verify error shown
- Send GET to `/api/generate` (via browser URL bar) — verify 405
- Send POST to `/api/search` — verify 405

**F. Edge cases**
- Generate for same repo twice — verify both stored (two rows)
- Generate with different channels (twitter vs linkedin) — verify tone differences
- Very long repo URL with extra path segments

## Deliverables

1. **List of issues found** — numbered, with severity (bug/warning/nit).
2. **Fixes** — code changes for each issue.
3. **Smoke test checklist** — ready to follow, in markdown format.
4. **Verification** — `go build ./...` must succeed after all fixes.

## Constraints

- Don't refactor working code unless there's a bug.
- Don't add new features.
- Focus on correctness and robustness, not style.
- If you find no issues in a category, say "Verified OK" and move on.
