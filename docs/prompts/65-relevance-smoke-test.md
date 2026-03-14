# Prompt: Relevance Tuning — Smoke Test and Verification

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. This is **Phase 1, prompt 3 of 3** for relevance tuning.

**Prerequisites**: Prompts 63–64 are complete. The `Search` method in `internal/store/store.go` now uses edismax with:
- **Field boosting** (`qf`): `headline^4`, `repo_name^3`, `tags^3`, `summary^2`, etc.
- **Phrase-field boosting** (`pf`): `headline^6 summary^3 repo_name^4`
- **Phrase slop** (`ps`): `2` — tolerates up to 2 extra words in phrase matches
- **Minimum match** (`mm`): `2<-1 5<80%` — short queries require all terms, longer queries allow partial
- **Tie-breaker** (`tie`): `0.1` — non-top matching fields contribute 10% of their score

## Your task

Verify the relevance tuning works correctly with an end-to-end smoke test using the running application. This involves:
1. Ensuring Solr is running and the app starts cleanly
2. Indexing test data with known characteristics
3. Running search queries and verifying the ranking order

## Prerequisites check

Before testing, verify the environment:

```powershell
# 1. Verify Solr is running
try { $r = Invoke-WebRequest -Uri "http://localhost:8983/solr/promotions/admin/ping" -UseBasicParsing -TimeoutSec 5; "Solr OK: $($r.StatusCode)" } catch { "Solr not running — start it first" }

# 2. Build the app
go build ./...

# 3. Run existing tests
go test ./internal/store/...
```

If Solr is not running, start it first (see prompt 54 for instructions). All tests must pass before proceeding.

## Test plan

### Step 1 — Index test documents via the Solr API

Index 4 test documents directly into Solr with distinct content so we can verify ranking. Use PowerShell to POST to the Solr update endpoint:

```powershell
$docs = @(
    @{
        id = "https://github.com/test/headline-match"
        repo_url = "https://github.com/test/headline-match"
        repo_name = "headline-match"
        headline = "Fast CLI Testing Framework for Go"
        summary = "A tool for backend developers to run integration tests."
        tags = @("integration", "backend")
        key_benefits = @("saves time")
        twitter_posts = @("Check out this tool!")
        linkedin_post = "Professional tool for teams."
        call_to_action = "Star us on GitHub."
        target_channel = "general"
        target_audience = "backend developers"
        created_at = "2026-03-14T10:00:00Z"
        stars = 100; forks = 20; watchers = 15
        readme = "This project helps with various tasks."
    },
    @{
        id = "https://github.com/test/summary-match"
        repo_url = "https://github.com/test/summary-match"
        repo_name = "summary-match"
        headline = "A Great Developer Utility"
        summary = "This is the best CLI testing framework for Go developers who need fast feedback loops."
        tags = @("utility", "developer")
        key_benefits = @("feedback loops")
        twitter_posts = @("Great utility!")
        linkedin_post = "A useful utility."
        call_to_action = "Try it today."
        target_channel = "general"
        target_audience = "Go developers"
        created_at = "2026-03-14T09:00:00Z"
        stars = 200; forks = 40; watchers = 30
        readme = "Installation instructions."
    },
    @{
        id = "https://github.com/test/readme-match"
        repo_url = "https://github.com/test/readme-match"
        repo_name = "readme-match"
        headline = "Amazing Project"
        summary = "This project does something impressive."
        tags = @("impressive", "project")
        key_benefits = @("easy to use")
        twitter_posts = @("Amazing project!")
        linkedin_post = "An impressive project."
        call_to_action = "Contribute today."
        target_channel = "general"
        target_audience = "all developers"
        created_at = "2026-03-14T08:00:00Z"
        stars = 50; forks = 5; watchers = 10
        readme = "This readme mentions CLI testing framework for Go in the middle of a very long paragraph about other things."
    },
    @{
        id = "https://github.com/test/tag-match"
        repo_url = "https://github.com/test/tag-match"
        repo_name = "tag-match"
        headline = "Microservice Starter"
        summary = "Template for building microservices."
        tags = @("cli", "testing", "go")
        key_benefits = @("quick setup")
        twitter_posts = @("Start building today!")
        linkedin_post = "Professional microservice template."
        call_to_action = "Fork and start."
        target_channel = "general"
        target_audience = "platform engineers"
        created_at = "2026-03-14T07:00:00Z"
        stars = 80; forks = 15; watchers = 12
        readme = "A microservice template project."
    }
) | ConvertTo-Json -Depth 5

Invoke-WebRequest -Uri "http://localhost:8983/solr/promotions/update/json/docs?commit=true" `
    -Method POST -ContentType "application/json" -Body $docs -UseBasicParsing
```

### Step 2 — Start the app

```powershell
# Start the server (ensure SOLR_URL, SOLR_CORE, GITHUB_TOKEN, and AGENT_ENDPOINT are set)
go run ./cmd/server/main.go
```

### Step 3 — Run search queries and verify ranking

Open a separate terminal and run these searches:

#### Test A: "CLI testing" — headline match should rank first

```powershell
$r = Invoke-WebRequest -Uri "http://localhost:8080/api/search?q=CLI+testing" -UseBasicParsing
$results = ($r.Content | ConvertFrom-Json).results
Write-Host "=== Test A: 'CLI testing' ==="
for ($i = 0; $i -lt $results.Count; $i++) {
    Write-Host "$($i+1). $($results[$i].repo_name) — $($results[$i].headline)"
}
```

**Expected ranking:**
1. `headline-match` — phrase "CLI Testing" appears **in the headline** (^4 boost + ^6 phrase boost)
2. `tag-match` — terms "cli" and "testing" appear **in tags** (^3 boost each)
3. `summary-match` — phrase "CLI testing" appears **in summary** (^2 boost + ^3 phrase boost)
4. `readme-match` — phrase appears **in readme** (^0.5 boost, lowest)

#### Test B: "Go developers" — repo_name and target_audience match

```powershell
$r = Invoke-WebRequest -Uri "http://localhost:8080/api/search?q=Go+developers" -UseBasicParsing
$results = ($r.Content | ConvertFrom-Json).results
Write-Host "=== Test B: 'Go developers' ==="
for ($i = 0; $i -lt $results.Count; $i++) {
    Write-Host "$($i+1). $($results[$i].repo_name) — audience: $($results[$i].target_audience)"
}
```

**Expected:** `summary-match` (target_audience = "Go developers", exact phrase) should rank highly.

#### Test C: Single-term query "microservice" — exact match

```powershell
$r = Invoke-WebRequest -Uri "http://localhost:8080/api/search?q=microservice" -UseBasicParsing
$results = ($r.Content | ConvertFrom-Json).results
Write-Host "=== Test C: 'microservice' ==="
for ($i = 0; $i -lt $results.Count; $i++) {
    Write-Host "$($i+1). $($results[$i].repo_name) — $($results[$i].headline)"
}
```

**Expected:** `tag-match` (headline = "Microservice Starter") should rank first — headline boost ^4.

#### Test D: Multi-term query with partial match — "CLI tool Go backend integration"

```powershell
$r = Invoke-WebRequest -Uri "http://localhost:8080/api/search?q=CLI+tool+Go+backend+integration" -UseBasicParsing
$results = ($r.Content | ConvertFrom-Json).results
Write-Host "=== Test D: 'CLI tool Go backend integration' (5 terms, mm=80%) ==="
for ($i = 0; $i -lt $results.Count; $i++) {
    Write-Host "$($i+1). $($results[$i].repo_name) — $($results[$i].headline)"
}
```

**Expected:** With `mm=80%`, at least 4 of 5 terms must match. `headline-match` should rank well (CLI + Go + integration + backend across fields). Documents matching only 1–2 terms should not appear.

### Step 4 — Clean up test data

After verification, remove the test documents:

```powershell
$deletePayload = @{ delete = @(
    "https://github.com/test/headline-match",
    "https://github.com/test/summary-match",
    "https://github.com/test/readme-match",
    "https://github.com/test/tag-match"
) } | ConvertTo-Json

Invoke-WebRequest -Uri "http://localhost:8983/solr/promotions/update?commit=true" `
    -Method POST -ContentType "application/json" -Body $deletePayload -UseBasicParsing
```

## Success criteria

- [ ] `go build ./...` succeeds
- [ ] `go test ./internal/store/...` passes
- [ ] Test A: `headline-match` ranks first for "CLI testing"
- [ ] Test B: `summary-match` ranks highly for "Go developers"
- [ ] Test C: `tag-match` ranks first for "microservice"
- [ ] Test D: Multi-term mm=80% filters out low-overlap results
- [ ] Test data cleaned up after verification

## Troubleshooting

If ranking seems wrong, inspect the raw Solr score with debugQuery:

```powershell
$r = Invoke-WebRequest -Uri "http://localhost:8983/solr/promotions/select?q=CLI+testing&defType=edismax&qf=repo_name%5E3+headline%5E4+summary%5E2+key_benefits%5E1.5+tags%5E3+twitter_posts%5E1+linkedin_post%5E1+call_to_action%5E1+target_audience%5E1.5+readme%5E0.5&pf=headline%5E6+summary%5E3+repo_name%5E4&ps=2&mm=2%3C-1+5%3C80%25&tie=0.1&debugQuery=true&fl=repo_name,headline,score&wt=json" -UseBasicParsing
$r.Content | ConvertFrom-Json | ConvertTo-Json -Depth 10
```

This shows the per-field score contribution so you can verify boosts are applied correctly.

## Files to modify

None — this is a verification-only prompt.
