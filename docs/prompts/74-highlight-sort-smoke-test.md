# Prompt: Highlighting & Sort — End-to-End Smoke Test

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. This is **Phase 3, prompt 4 of 4** for search highlighting & sort options.

**Prerequisites**: Prompts 71–73 are complete. The full stack now supports:
- **Backend**: `Search` requests Solr highlighting (`hl=true`, `hl.fl=headline,summary,...`, `hl.method=unified`); `parseHighlights` extracts snippets; `SearchResult.Highlights` returned in API response; `SearchOptions.Sort` maps to Solr sort via `solrSort` helper
- **Handler**: Passes `?sort=` param through; returns `highlights` in JSON response
- **Frontend**: Sort dropdown sends `?sort=` in API URL; `safeHighlight`/`hlField` render `<mark>` tags in result cards; CSS styles `mark` with yellow background

## Your task

Run an end-to-end smoke test to verify highlighting and sort options work correctly.

## Prerequisites check

```powershell
# 1. Verify Solr is running
try { $r = Invoke-WebRequest -Uri "http://localhost:8983/solr/promotions/admin/ping" -UseBasicParsing -TimeoutSec 5; "Solr OK: $($r.StatusCode)" } catch { "Solr not running — start it first" }

# 2. Build the app
go build ./...

# 3. Run tests
go test ./internal/store/... -tags integration
```

## Test plan

### Step 1 — Index test documents

```powershell
$docs = @(
    @{
        id = "https://github.com/test/hl-go-cli"
        repo_url = "https://github.com/test/hl-go-cli"
        repo_name = "hl-go-cli"
        headline = "Lightning Fast CLI Framework for Go Developers"
        summary = "Build production-ready CLI applications in Go with minimal boilerplate. This framework provides argument parsing, help generation, and plugin support out of the box."
        tags = @("go", "cli", "framework")
        target_channel = "general"
        created_at = "2026-03-14T10:00:00Z"
        stars = 500; forks = 50; watchers = 40
        views_14d_total = 1200; views_14d_unique = 600; clones_14d_total = 80; clones_14d_unique = 40
        key_benefits = @("zero boilerplate", "plugin system"); twitter_posts = @("Build CLI apps fast!"); linkedin_post = "CLI framework for Go."; call_to_action = "Star us on GitHub."; target_audience = "Go developers building CLI tools"; readme = "Readme content."
    },
    @{
        id = "https://github.com/test/hl-python-ml"
        repo_url = "https://github.com/test/hl-python-ml"
        repo_name = "hl-python-ml"
        headline = "Python Machine Learning Toolkit"
        summary = "Comprehensive machine learning utilities for Python developers. Includes data preprocessing, model training, evaluation metrics, and a CLI for quick experimentation."
        tags = @("python", "ml", "data-science", "cli")
        target_channel = "linkedin"
        created_at = "2026-03-14T09:00:00Z"
        stars = 2000; forks = 300; watchers = 200
        views_14d_total = 5000; views_14d_unique = 2500; clones_14d_total = 300; clones_14d_unique = 150
        key_benefits = @("all-in-one ML", "CLI experimentation"); twitter_posts = @("ML made simple!"); linkedin_post = "Python ML toolkit."; call_to_action = "Try it today."; target_audience = "data scientists and ML engineers"; readme = "Readme content."
    },
    @{
        id = "https://github.com/test/hl-js-react"
        repo_url = "https://github.com/test/hl-js-react"
        repo_name = "hl-js-react"
        headline = "React Component Library with TypeScript"
        summary = "Beautiful and accessible React components for building modern web applications. Includes buttons, forms, modals, tables, and a CLI scaffolding tool."
        tags = @("javascript", "react", "typescript", "frontend")
        target_channel = "twitter"
        created_at = "2026-03-14T08:00:00Z"
        stars = 800; forks = 120; watchers = 90
        views_14d_total = 3000; views_14d_unique = 1500; clones_14d_total = 200; clones_14d_unique = 100
        key_benefits = @("accessible", "TypeScript native"); twitter_posts = @("Beautiful React components!"); linkedin_post = "React component library."; call_to_action = "Contribute today."; target_audience = "frontend developers using React"; readme = "Readme content."
    },
    @{
        id = "https://github.com/test/hl-go-testing"
        repo_url = "https://github.com/test/hl-go-testing"
        repo_name = "hl-go-testing"
        headline = "Go Testing Utilities for CLI Applications"
        summary = "Helper functions and test fixtures for testing Go CLI applications. Capture stdout, simulate user input, and validate command output."
        tags = @("go", "testing", "cli")
        target_channel = "general"
        created_at = "2026-03-14T07:00:00Z"
        stars = 150; forks = 20; watchers = 15
        views_14d_total = 400; views_14d_unique = 200; clones_14d_total = 30; clones_14d_unique = 15
        key_benefits = @("easy mocking", "stdout capture"); twitter_posts = @("Test your CLI tools!"); linkedin_post = "Go testing helpers."; call_to_action = "Use it."; target_audience = "Go developers writing CLI tests"; readme = "Readme content."
    }
) | ConvertTo-Json -Depth 5

Invoke-WebRequest -Uri "http://localhost:8983/solr/promotions/update/json/docs?commit=true" `
    -Method POST -ContentType "application/json" -Body $docs -UseBasicParsing
```

### Step 2 — Start the app

```powershell
go run ./cmd/server/main.go
```

### Step 3 — Verify highlighting in API

#### Test A: Search "CLI" — verify highlights in response

```powershell
$r = Invoke-WebRequest -Uri "http://localhost:8080/api/search?q=CLI" -UseBasicParsing
$d = $r.Content | ConvertFrom-Json
Write-Host "=== Test A: Highlights for 'CLI' ==="
Write-Host "Results: $($d.count)"
foreach($id in $d.highlights.PSObject.Properties.Name){
    Write-Host "  Doc: $id"
    foreach($field in $d.highlights.$id.PSObject.Properties.Name){
        Write-Host "    $field => $($d.highlights.$id.$field)"
    }
}
```

**Expected:**
- `highlights` object is present and non-empty
- Each matching document has at least one field with `<mark>CLI</mark>` in the highlighted text
- `hl-go-cli` should have highlight in `headline` (contains "CLI")
- `hl-python-ml` should have highlight in `summary` (contains "CLI")
- `hl-go-testing` should have highlight in `headline` (contains "CLI")

#### Test B: Verify no highlights on browse (empty query)

```powershell
$r = Invoke-WebRequest -Uri "http://localhost:8080/api/search" -UseBasicParsing
$d = $r.Content | ConvertFrom-Json
Write-Host "=== Test B: Browse — no highlights ==="
Write-Host "Highlights present: $(if($d.highlights){$true}else{$false})"
```

**Expected:** `highlights` is null/absent (no query = no highlighting).

### Step 4 — Verify sort options in API

#### Test C: Sort by stars

```powershell
$r = Invoke-WebRequest -Uri "http://localhost:8080/api/search?sort=stars" -UseBasicParsing
$d = $r.Content | ConvertFrom-Json
Write-Host "=== Test C: Sort by stars ==="
$d.results | ForEach-Object { Write-Host "  $($_.repo_name) — stars: $($_.stars)" }
```

**Expected order:** `hl-python-ml` (2000), `hl-js-react` (800), `hl-go-cli` (500), `hl-go-testing` (150).

#### Test D: Sort by views

```powershell
$r = Invoke-WebRequest -Uri "http://localhost:8080/api/search?sort=views" -UseBasicParsing
$d = $r.Content | ConvertFrom-Json
Write-Host "=== Test D: Sort by views ==="
$d.results | ForEach-Object { Write-Host "  $($_.repo_name) — views: $($_.views_14d_total)" }
```

**Expected order:** `hl-python-ml` (5000), `hl-js-react` (3000), `hl-go-cli` (1200), `hl-go-testing` (400).

#### Test E: Sort by newest

```powershell
$r = Invoke-WebRequest -Uri "http://localhost:8080/api/search?sort=newest" -UseBasicParsing
$d = $r.Content | ConvertFrom-Json
Write-Host "=== Test E: Sort by newest ==="
$d.results | ForEach-Object { Write-Host "  $($_.repo_name) — created: $($_.created_at)" }
```

**Expected order:** `hl-go-cli` (10:00), `hl-python-ml` (09:00), `hl-js-react` (08:00), `hl-go-testing` (07:00).

#### Test F: Search with sort — "Go" sorted by stars

```powershell
$r = Invoke-WebRequest -Uri "http://localhost:8080/api/search?q=Go&sort=stars" -UseBasicParsing
$d = $r.Content | ConvertFrom-Json
Write-Host "=== Test F: q=Go, sort=stars ==="
$d.results | ForEach-Object { Write-Host "  $($_.repo_name) — stars: $($_.stars)" }
Write-Host "Has highlights: $(if($d.highlights){$true}else{$false})"
```

**Expected:** Go-matching results sorted by stars descending; highlights present with `<mark>Go</mark>`.

### Step 5 — Verify frontend

1. Open `http://localhost:8080` in a browser.
2. Go to the **Search & Browse** tab.

#### Highlighting tests:
3. Search for **"CLI"**.
4. **Verify**: In result cards, the word "CLI" appears with a **yellow highlight background** in headlines that contain it.
5. Click on a card to expand it. **Verify**: The word "CLI" is highlighted in the summary text if it appears there.
6. Clear the query and browse. **Verify**: No highlights visible (plain text).

#### Sort tests:
7. **Verify**: A "Sort by:" dropdown appears above the results.
8. Select **"Most Stars"**. **Verify**: Results reorder — `hl-python-ml` (2000 stars) first.
9. Select **"Most Views"**. **Verify**: Results reorder — `hl-python-ml` (5000 views) first.
10. Select **"Newest"**. **Verify**: Results order by date — `hl-go-cli` first.
11. Search for **"framework"** with sort set to **"Relevance"**. **Verify**: Results sorted by search relevance, "framework" highlighted.
12. Switch to **"Most Stars"** while search query is still "framework". **Verify**: Results re-sort by stars, highlights still present.

### Step 6 — Clean up test data

```powershell
$deletePayload = @{ delete = @(
    "https://github.com/test/hl-go-cli",
    "https://github.com/test/hl-python-ml",
    "https://github.com/test/hl-js-react",
    "https://github.com/test/hl-go-testing"
) } | ConvertTo-Json

Invoke-WebRequest -Uri "http://localhost:8983/solr/promotions/update?commit=true" `
    -Method POST -ContentType "application/json" -Body $deletePayload -UseBasicParsing
```

## Success criteria

- [ ] `go build ./...` succeeds
- [ ] `go test ./internal/store/... -tags integration` passes
- [ ] Test A: API returns `highlights` with `<mark>` snippets for search queries
- [ ] Test B: Browse (empty query) has no highlights
- [ ] Test C: `?sort=stars` sorts by star count descending
- [ ] Test D: `?sort=views` sorts by views descending
- [ ] Test E: `?sort=newest` sorts by created_at descending
- [ ] Test F: Search + sort works together with highlights
- [ ] Frontend: Highlighted terms have yellow background in cards
- [ ] Frontend: Sort dropdown changes result order
- [ ] Frontend: Highlights + sort work together
- [ ] Test data cleaned up

## Troubleshooting

If highlights are empty, check Solr directly:

```powershell
$r = Invoke-WebRequest -Uri "http://localhost:8983/solr/promotions/select?q=CLI&defType=edismax&qf=repo_name%5E3+headline%5E4+summary%5E2&hl=true&hl.fl=headline,summary&hl.simple.pre=%3Cmark%3E&hl.simple.post=%3C/mark%3E&hl.method=unified&wt=json&rows=5" -UseBasicParsing
$r.Content | ConvertFrom-Json | ConvertTo-Json -Depth 10
```

If the `highlighting` object is present in Solr but not in the app response, check that the handler includes `Highlights: sr.Highlights` in the response encoding.

If sort isn't working, check the Solr field types — `stars` and `views_14d_total` must be `pint` (not `text_general`) to sort numerically.

## Files to modify

None — this is a verification-only prompt.
