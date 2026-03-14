# Prompt: Faceted Search — End-to-End Smoke Test

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. This is **Phase 2, prompt 5 of 5** for faceted search and filtering.

**Prerequisites**: Prompts 66–69 are complete. The full stack now supports:
- **Backend**: `Search`/`List` return `SearchResult` with `Facets map[string][]Facet`; accept `SearchOptions` with `Tags`, `Channel`, `MinStars`; Solr `fq` filter queries for exact-match filtering
- **Handler**: `GET /api/search` accepts `?tag=X&channel=Y&min_stars=N` params, returns `{ results, count, facets }`
- **Frontend**: Facet chips rendered from `facets` response; clicking toggles filter; active filters shown as removable chips; "Clear all" resets

## Your task

Run an end-to-end smoke test to verify faceted search and filtering work correctly across all layers.

## Prerequisites check

```powershell
# 1. Verify Solr is running
try { $r = Invoke-WebRequest -Uri "http://localhost:8983/solr/promotions/admin/ping" -UseBasicParsing -TimeoutSec 5; "Solr OK: $($r.StatusCode)" } catch { "Solr not running — start it first" }

# 2. Build the app
go build ./...

# 3. Run tests
go test ./internal/store/... -tags integration
```

All must pass before proceeding.

## Test plan

### Step 1 — Index test documents with varied tags and channels

```powershell
$docs = @(
    @{
        id = "https://github.com/test/facet-go-cli"
        repo_url = "https://github.com/test/facet-go-cli"
        repo_name = "facet-go-cli"
        headline = "Go CLI Framework"
        summary = "A CLI framework for Go developers."
        tags = @("go", "cli", "framework")
        target_channel = "general"
        created_at = "2026-03-14T10:00:00Z"
        stars = 500; forks = 50; watchers = 40
        key_benefits = @("fast"); twitter_posts = @("Check it out!"); linkedin_post = "Pro tool."; call_to_action = "Star us."; target_audience = "Go developers"; readme = "Readme."
    },
    @{
        id = "https://github.com/test/facet-go-web"
        repo_url = "https://github.com/test/facet-go-web"
        repo_name = "facet-go-web"
        headline = "Go Web Framework"
        summary = "A web framework for building APIs in Go."
        tags = @("go", "web", "api")
        target_channel = "twitter"
        created_at = "2026-03-14T09:00:00Z"
        stars = 200; forks = 30; watchers = 25
        key_benefits = @("easy"); twitter_posts = @("Build APIs!"); linkedin_post = "API building."; call_to_action = "Try it."; target_audience = "backend developers"; readme = "Readme."
    },
    @{
        id = "https://github.com/test/facet-python-ml"
        repo_url = "https://github.com/test/facet-python-ml"
        repo_name = "facet-python-ml"
        headline = "Python ML Toolkit"
        summary = "Machine learning utilities for Python."
        tags = @("python", "ml", "data-science")
        target_channel = "linkedin"
        created_at = "2026-03-14T08:00:00Z"
        stars = 1000; forks = 200; watchers = 150
        key_benefits = @("powerful"); twitter_posts = @("ML made easy!"); linkedin_post = "AI toolkit."; call_to_action = "Fork it."; target_audience = "data scientists"; readme = "Readme."
    },
    @{
        id = "https://github.com/test/facet-go-testing"
        repo_url = "https://github.com/test/facet-go-testing"
        repo_name = "facet-go-testing"
        headline = "Go Testing Helpers"
        summary = "Testing utilities for Go projects."
        tags = @("go", "testing", "cli")
        target_channel = "general"
        created_at = "2026-03-14T07:00:00Z"
        stars = 80; forks = 10; watchers = 8
        key_benefits = @("easy testing"); twitter_posts = @("Test smarter!"); linkedin_post = "Testing."; call_to_action = "Use it."; target_audience = "Go developers"; readme = "Readme."
    },
    @{
        id = "https://github.com/test/facet-js-react"
        repo_url = "https://github.com/test/facet-js-react"
        repo_name = "facet-js-react"
        headline = "React Component Library"
        summary = "Reusable React components for web apps."
        tags = @("javascript", "react", "frontend")
        target_channel = "twitter"
        created_at = "2026-03-14T06:00:00Z"
        stars = 300; forks = 60; watchers = 45
        key_benefits = @("reusable"); twitter_posts = @("Beautiful UI!"); linkedin_post = "Components."; call_to_action = "Contribute."; target_audience = "frontend developers"; readme = "Readme."
    }
) | ConvertTo-Json -Depth 5

Invoke-WebRequest -Uri "http://localhost:8983/solr/promotions/update/json/docs?commit=true" `
    -Method POST -ContentType "application/json" -Body $docs -UseBasicParsing
```

### Step 2 — Start the app

```powershell
go run ./cmd/server/main.go
```

### Step 3 — Verify facets in API response

#### Test A: Browse (no query) — verify facets are returned

```powershell
$r = Invoke-WebRequest -Uri "http://localhost:8080/api/search" -UseBasicParsing
$d = $r.Content | ConvertFrom-Json
Write-Host "=== Test A: Browse — facets ==="
Write-Host "Results: $($d.count)"
Write-Host "Tag facets: $(($d.facets.tags | ForEach-Object { "$($_.value):$($_.count)" }) -join ', ')"
Write-Host "Channel facets: $(($d.facets.target_channel | ForEach-Object { "$($_.value):$($_.count)" }) -join ', ')"
```

**Expected:**
- 5 results
- Tag facets: `go:3` (appears in 3 docs), `cli:2`, `testing:1`, `framework:1`, `web:1`, `api:1`, `python:1`, `ml:1`, `data-science:1`, `javascript:1`, `react:1`, `frontend:1`
- Channel facets: `general:2`, `twitter:2`, `linkedin:1`

#### Test B: Filter by tag — `?tag=go`

```powershell
$r = Invoke-WebRequest -Uri "http://localhost:8080/api/search?tag=go" -UseBasicParsing
$d = $r.Content | ConvertFrom-Json
Write-Host "=== Test B: Filter tag=go ==="
Write-Host "Results: $($d.count)"
$d.results | ForEach-Object { Write-Host " - $($_.repo_name) (tags: $($_.tags -join ', '))" }
```

**Expected:** 3 results: `facet-go-cli`, `facet-go-web`, `facet-go-testing` — all have tag "go".

#### Test C: Filter by multiple tags — `?tag=go&tag=cli`

```powershell
$r = Invoke-WebRequest -Uri "http://localhost:8080/api/search?tag=go&tag=cli" -UseBasicParsing
$d = $r.Content | ConvertFrom-Json
Write-Host "=== Test C: Filter tag=go AND tag=cli ==="
Write-Host "Results: $($d.count)"
$d.results | ForEach-Object { Write-Host " - $($_.repo_name) (tags: $($_.tags -join ', '))" }
```

**Expected:** 2 results: `facet-go-cli` and `facet-go-testing` — both have tags "go" AND "cli".

#### Test D: Filter by channel — `?channel=twitter`

```powershell
$r = Invoke-WebRequest -Uri "http://localhost:8080/api/search?channel=twitter" -UseBasicParsing
$d = $r.Content | ConvertFrom-Json
Write-Host "=== Test D: Filter channel=twitter ==="
Write-Host "Results: $($d.count)"
$d.results | ForEach-Object { Write-Host " - $($_.repo_name) (channel: $($_.target_channel))" }
```

**Expected:** 2 results: `facet-go-web` and `facet-js-react` — both have `target_channel = "twitter"`.

#### Test E: Filter by min stars — `?min_stars=250`

```powershell
$r = Invoke-WebRequest -Uri "http://localhost:8080/api/search?min_stars=250" -UseBasicParsing
$d = $r.Content | ConvertFrom-Json
Write-Host "=== Test E: Filter min_stars=250 ==="
Write-Host "Results: $($d.count)"
$d.results | ForEach-Object { Write-Host " - $($_.repo_name) (stars: $($_.stars))" }
```

**Expected:** 3 results with stars >= 250: `facet-python-ml` (1000), `facet-go-cli` (500), `facet-js-react` (300).

#### Test F: Combined — query + filter

```powershell
$r = Invoke-WebRequest -Uri "http://localhost:8080/api/search?q=framework&tag=go" -UseBasicParsing
$d = $r.Content | ConvertFrom-Json
Write-Host "=== Test F: q=framework AND tag=go ==="
Write-Host "Results: $($d.count)"
$d.results | ForEach-Object { Write-Host " - $($_.repo_name) — $($_.headline)" }
```

**Expected:** Results that match "framework" AND have tag "go": `facet-go-cli` ("Go CLI Framework") and `facet-go-web` ("Go Web Framework").

### Step 4 — Verify frontend

1. Open `http://localhost:8080` in a browser.
2. Click the **Search & Browse** tab.
3. Click **Search** with empty query.
4. **Verify**: Facet chips appear below the search bar with tag names/counts and channel names/counts.
5. **Click** the "go" tag chip.
6. **Verify**: The chip highlights (blue), a yellow "tag: go ×" filter chip appears, and results narrow to 3 Go repos.
7. **Click** the "cli" tag chip while "go" is still active.
8. **Verify**: Two filter chips ("tag: go" and "tag: cli"), results narrow to 2 repos.
9. **Click** the × on the "go" filter chip.
10. **Verify**: Only "tag: cli" remains, and results show the 2 repos with "cli" tag.
11. **Click** "Clear all ×".
12. **Verify**: All filters removed, all results shown again.
13. **Click** a channel chip (e.g., "twitter").
14. **Verify**: Only twitter-channel results appear.

### Step 5 — Clean up test data

```powershell
$deletePayload = @{ delete = @(
    "https://github.com/test/facet-go-cli",
    "https://github.com/test/facet-go-web",
    "https://github.com/test/facet-python-ml",
    "https://github.com/test/facet-go-testing",
    "https://github.com/test/facet-js-react"
) } | ConvertTo-Json

Invoke-WebRequest -Uri "http://localhost:8983/solr/promotions/update?commit=true" `
    -Method POST -ContentType "application/json" -Body $deletePayload -UseBasicParsing
```

## Success criteria

- [ ] `go build ./...` succeeds
- [ ] `go test ./internal/store/... -tags integration` passes
- [ ] Test A: Browse returns facets with correct counts
- [ ] Test B: Single tag filter narrows results correctly
- [ ] Test C: Multiple tag filters use AND logic
- [ ] Test D: Channel filter works
- [ ] Test E: Min stars filter works
- [ ] Test F: Query + filter combined works
- [ ] Frontend: Facet chips render with counts
- [ ] Frontend: Clicking facet chip toggles filter and re-searches
- [ ] Frontend: Active filter chips show with × removal
- [ ] Frontend: "Clear all" removes all filters
- [ ] Test data cleaned up

## Troubleshooting

If facets are missing from the response, check the raw Solr response:

```powershell
$r = Invoke-WebRequest -Uri "http://localhost:8983/solr/promotions/select?q=*:*&facet=true&facet.field=tags&facet.field=target_channel&facet.mincount=1&wt=json&rows=0" -UseBasicParsing
$r.Content | ConvertFrom-Json | ConvertTo-Json -Depth 10
```

If filters aren't working, check the `fq` parameter is being sent:

```powershell
$r = Invoke-WebRequest -Uri "http://localhost:8983/solr/promotions/select?q=*:*&fq=tags:%22go%22&wt=json&rows=10" -UseBasicParsing
($r.Content | ConvertFrom-Json).response.numFound
```

## Files to modify

None — this is a verification-only prompt.
