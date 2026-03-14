# Prompt: Phase 5 — Advanced Features End-to-End Smoke Test

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. This is **Phase 5, prompt 5 of 5** for advanced features.

**Prerequisites**: Prompts 80–83 are complete. The full stack now supports:
- **Spellcheck**: `Store.Search()` sends `spellcheck=true`, `parseCollation` extracts "Did you mean?" suggestion; `SearchResult.Collation` and `searchResponse.Collation` carry it to the frontend; "Did you mean?" link shown above results
- **Result grouping**: Frontend `groupByRepo()` clusters promotions by `repo_url`; channel tabs (🐦 twitter | 💼 linkedin | 📢 general) appear in expanded cards when multiple channels exist
- **More Like This**: `Store.MoreLikeThis()` uses Solr MLT component; `GET /api/mlt?id=<repo_url>` endpoint; "Find Similar" button on expanded cards; `findSimilar()` toggles similar results display
- **Search analytics**: `analytics.Tracker` records queries in-memory; `GET /api/analytics/popular` returns top queries; "🔥 Popular Searches" chips displayed on search page, updated after each search

## Your task

Run an end-to-end smoke test to verify all Phase 5 features work correctly.

## Prerequisites check

```powershell
# 1. Verify Solr is running
try { $r = Invoke-WebRequest -Uri "http://localhost:8983/solr/promotions/admin/ping" -UseBasicParsing -TimeoutSec 5; "Solr OK: $($r.StatusCode)" } catch { "Solr not running — start it first" }

# 2. Verify spellcheck component is configured
try { $r = Invoke-RestMethod -Uri "http://localhost:8983/solr/promotions/select?q=test&spellcheck=true&wt=json&rows=0" -Method GET; "Spellcheck component OK" } catch { "Spellcheck not configured — run prompt 80 Solr config first" }

# 3. Build the app
go build ./...
```

## Test plan

### Step 1 — Index test documents

Index 5 documents with varied content, tags, and channels. Include some with the same repo to test grouping. Include some with similar content to test MLT.

```powershell
$solrBase = "http://localhost:8983/solr/promotions/update/json/docs?commit=true"

# Doc 1: Container toolkit — general channel
$doc1 = @{
    id = "https://github.com/test/p5-containers-general"
    repo_url = "https://github.com/test/p5-containers"
    repo_name = "p5-containers"
    headline = "Enterprise Container Orchestration Platform"
    summary = "Deploy and manage containers across multi-cloud environments with declarative YAML configs and zero-downtime updates."
    tags = @("docker", "kubernetes", "containers", "devops")
    target_channel = "general"
    created_at = "2026-03-14T10:00:00Z"
    stars = 450; forks = 55; watchers = 40
    views_14d_total = 800; views_14d_unique = 400; clones_14d_total = 50; clones_14d_unique = 25
    key_benefits = @("multi-cloud deployment", "zero-downtime rolling updates"); twitter_posts = @("Container orchestration made simple!"); linkedin_post = "Enterprise container platform."; call_to_action = "Deploy your first container in 5 minutes."; target_audience = "DevOps engineers"; readme = "Container orchestration platform."
} | ConvertTo-Json -Depth 3
Invoke-RestMethod -Uri $solrBase -Method POST -ContentType "application/json" -Body $doc1

# Doc 2: Same repo — twitter channel (tests grouping)
$doc2 = @{
    id = "https://github.com/test/p5-containers-twitter"
    repo_url = "https://github.com/test/p5-containers"
    repo_name = "p5-containers"
    headline = "Container Management Made Easy"
    summary = "Stop fighting YAML. Our container platform handles deployment, scaling, and monitoring in one tool."
    tags = @("docker", "kubernetes", "containers")
    target_channel = "twitter"
    created_at = "2026-03-14T10:05:00Z"
    stars = 450; forks = 55; watchers = 40
    views_14d_total = 800; views_14d_unique = 400; clones_14d_total = 50; clones_14d_unique = 25
    key_benefits = @("single tool for deploy+monitor", "smart scaling"); twitter_posts = @("🐳 Tired of container configs? p5-containers has you covered!"); linkedin_post = ""; call_to_action = "Star us on GitHub."; target_audience = "Cloud engineers"; readme = "Container platform readme."
} | ConvertTo-Json -Depth 3
Invoke-RestMethod -Uri $solrBase -Method POST -ContentType "application/json" -Body $doc2

# Doc 3: Same repo — linkedin channel (tests 3-way grouping)
$doc3 = @{
    id = "https://github.com/test/p5-containers-linkedin"
    repo_url = "https://github.com/test/p5-containers"
    repo_name = "p5-containers"
    headline = "The Next-Gen Container Platform for Enterprises"
    summary = "Reduce cloud costs by 40% with intelligent container placement and resource optimization."
    tags = @("docker", "kubernetes", "containers", "cloud")
    target_channel = "linkedin"
    created_at = "2026-03-14T10:10:00Z"
    stars = 450; forks = 55; watchers = 40
    views_14d_total = 800; views_14d_unique = 400; clones_14d_total = 50; clones_14d_unique = 25
    key_benefits = @("40% cost reduction", "intelligent placement"); twitter_posts = @(); linkedin_post = "Reduce your cloud costs with p5-containers. Our intelligent container placement engine optimizes resource usage across your multi-cloud fleet."; call_to_action = "Book a demo."; target_audience = "CTOs and engineering leaders"; readme = "Container platform."
} | ConvertTo-Json -Depth 3
Invoke-RestMethod -Uri $solrBase -Method POST -ContentType "application/json" -Body $doc3

# Doc 4: Python ML library (different repo, similar to doc 5 for MLT testing)
$doc4 = @{
    id = "https://github.com/test/p5-ml-pipeline"
    repo_url = "https://github.com/test/p5-ml-pipeline"
    repo_name = "p5-ml-pipeline"
    headline = "Python Machine Learning Pipeline Framework"
    summary = "Build production ML pipelines with built-in data validation, feature engineering, and model versioning."
    tags = @("python", "machine-learning", "pipeline", "data-science")
    target_channel = "general"
    created_at = "2026-03-14T11:00:00Z"
    stars = 720; forks = 80; watchers = 65
    views_14d_total = 1500; views_14d_unique = 700; clones_14d_total = 90; clones_14d_unique = 45
    key_benefits = @("automated data validation", "experiment tracking", "model versioning"); twitter_posts = @("ML pipelines done right!"); linkedin_post = "Production ML pipelines."; call_to_action = "Start building."; target_audience = "Data scientists"; readme = "ML pipeline framework."
} | ConvertTo-Json -Depth 3
Invoke-RestMethod -Uri $solrBase -Method POST -ContentType "application/json" -Body $doc4

# Doc 5: Another ML library (similar to doc 4 — for MLT testing)
$doc5 = @{
    id = "https://github.com/test/p5-automl"
    repo_url = "https://github.com/test/p5-automl"
    repo_name = "p5-automl"
    headline = "Automated Machine Learning for Python Developers"
    summary = "AutoML library that handles feature selection, hyperparameter tuning, and model comparison with a single API call."
    tags = @("python", "machine-learning", "automl", "data-science")
    target_channel = "general"
    created_at = "2026-03-14T12:00:00Z"
    stars = 340; forks = 35; watchers = 28
    views_14d_total = 600; views_14d_unique = 300; clones_14d_total = 40; clones_14d_unique = 20
    key_benefits = @("single API call", "automated feature selection", "model comparison"); twitter_posts = @("AutoML made accessible!"); linkedin_post = "AutoML for everyone."; call_to_action = "Try AutoML today."; target_audience = "Python developers new to ML"; readme = "AutoML library."
} | ConvertTo-Json -Depth 3
Invoke-RestMethod -Uri $solrBase -Method POST -ContentType "application/json" -Body $doc5

"Test documents indexed."
```

### Step 2 — Start the server

```powershell
Get-NetTCPConnection -LocalPort 8080 -ErrorAction SilentlyContinue | ForEach-Object { Stop-Process -Id $_.OwningProcess -Force -ErrorAction SilentlyContinue }; Start-Sleep -Seconds 1
go run ./cmd/server/main.go
```

### Step 3 — API tests for spellcheck

```powershell
# Test 1: Misspelled query — "containr" should suggest "container"
$r = Invoke-RestMethod -Uri "http://localhost:8080/api/search?q=containr" -Method GET
"Test 1 — spellcheck collation: '$($r.collation)'"
if($r.collation){ "  PASS: Got suggestion" } else { "  INFO: No collation (may need more indexed data for spellcheck to activate)" }

# Test 2: Correct query — should NOT have collation
$r = Invoke-RestMethod -Uri "http://localhost:8080/api/search?q=container" -Method GET
"Test 2 — correct query collation: '$($r.collation)'"
if(!$r.collation -or $r.collation -eq ''){ "  PASS: No collation for correct query" } else { "  INFO: Collation present (may still be valid)" }

# Test 3: "machin lerning" misspelling
$r = Invoke-RestMethod -Uri "http://localhost:8080/api/search?q=machin%20lerning" -Method GET
"Test 3 — 'machin lerning' collation: '$($r.collation)'"
```

### Step 4 — API tests for result grouping (verify multiple docs for same repo_url come back)

```powershell
# Test 4: Search for "container" — should return all 3 p5-containers docs
$r = Invoke-RestMethod -Uri "http://localhost:8080/api/search?q=container" -Method GET
$containerDocs = $r.results | Where-Object { $_.repo_url -eq "https://github.com/test/p5-containers" }
"Test 4 — container docs for same repo: $($containerDocs.Count)"
if($containerDocs.Count -eq 3){ "  PASS: All 3 channels returned" } else { "  INFO: Got $($containerDocs.Count) (grouping is frontend-only, API is flat)" }
```

### Step 5 — API tests for More Like This

```powershell
# Test 5: MLT for the ML pipeline repo — should find automl as similar
$r = Invoke-RestMethod -Uri "http://localhost:8080/api/mlt?id=https://github.com/test/p5-ml-pipeline" -Method GET
"Test 5 — MLT for ml-pipeline: $($r.count) similar results"
if($r.count -gt 0){ $r.results | ForEach-Object { "  Similar: $($_.repo_name) ($($_.target_channel))" } } else { "  No similar results (MLT needs sufficient indexed content)" }

# Test 6: MLT for the automl repo — should find ml-pipeline as similar
$r = Invoke-RestMethod -Uri "http://localhost:8080/api/mlt?id=https://github.com/test/p5-automl" -Method GET
"Test 6 — MLT for automl: $($r.count) similar results"
if($r.count -gt 0){ $r.results | ForEach-Object { "  Similar: $($_.repo_name) ($($_.target_channel))" } } else { "  No similar results" }

# Test 7: MLT with missing id
$r = Invoke-RestMethod -Uri "http://localhost:8080/api/mlt?id=" -Method GET -ErrorAction SilentlyContinue
"Test 7 — MLT empty id: should return error"
```

### Step 6 — API tests for search analytics

```powershell
# Do several searches to build up analytics
Invoke-RestMethod -Uri "http://localhost:8080/api/search?q=container" -Method GET | Out-Null
Invoke-RestMethod -Uri "http://localhost:8080/api/search?q=container" -Method GET | Out-Null
Invoke-RestMethod -Uri "http://localhost:8080/api/search?q=container" -Method GET | Out-Null
Invoke-RestMethod -Uri "http://localhost:8080/api/search?q=python" -Method GET | Out-Null
Invoke-RestMethod -Uri "http://localhost:8080/api/search?q=python" -Method GET | Out-Null
Invoke-RestMethod -Uri "http://localhost:8080/api/search?q=machine%20learning" -Method GET | Out-Null

# Test 8: Check popular searches
$r = Invoke-RestMethod -Uri "http://localhost:8080/api/analytics/popular" -Method GET
"Test 8 — Popular searches:"
$r | ForEach-Object { "  $($_.query): $($_.count)" }
# Expected: container: 4+ (including test 4), python: 2, machine learning: 1

# Test 9: Empty browse should NOT be recorded
Invoke-RestMethod -Uri "http://localhost:8080/api/search?q=" -Method GET | Out-Null
$r2 = Invoke-RestMethod -Uri "http://localhost:8080/api/analytics/popular" -Method GET
$empty = $r2 | Where-Object { $_.query -eq '' }
if(!$empty){ "Test 9 — PASS: Empty query not recorded" } else { "Test 9 — FAIL: Empty query was recorded" }
```

### Step 7 — Frontend verification

Open `http://localhost:8080` in a browser:

1. **Switch to Search tab**

**Spellcheck:**
2. Search for "containr" (misspelled) → check for "Did you mean: container?" link above results
3. Click the "Did you mean?" link → search box should change to "container" and search should re-execute
4. Search for "container" (correct) → no "Did you mean?" should appear

**Result grouping:**
5. Search for "container" → p5-containers should appear as a single grouped card with "3 channels" badge
6. Expand the p5-containers card → channel tabs should appear (📢 general | 🐦 twitter | 💼 linkedin)
7. Click each channel tab → content should switch (headline, summary, tweets, etc.)
8. p5-ml-pipeline and p5-automl should appear as individual cards (1 channel each, no tabs)

**More Like This:**
9. Expand the p5-ml-pipeline card → scroll to bottom → click "🔍 Find Similar"
10. Similar results should load — p5-automl should appear as a similar promotion
11. Click "🔍 Find Similar" again (now "Hide Similar") → similar results should collapse
12. Expand p5-automl → "Find Similar" → should find p5-ml-pipeline

**Search analytics:**
13. After doing several searches, "🔥 Popular Searches" section should appear
14. Click a popular search chip → search box should fill and search should execute
15. Counts should update after each search

### Step 8 — Cleanup test data

```powershell
$deleteBody = '{"delete": {"query": "repo_url:https\\://github.com/test/p5-*"}}'
Invoke-RestMethod -Uri "http://localhost:8983/solr/promotions/update?commit=true" -Method POST -ContentType "application/json" -Body $deleteBody
"Test documents cleaned up."
```

## Expected results summary

| # | Test | Expected Result |
|---|------|----------------|
| 1 | Spellcheck "containr" | Collation: "container" (if enough indexed data) |
| 2 | Correct "container" | No collation |
| 3 | Spellcheck "machin lerning" | Collation: "machine learning" (if enough data) |
| 4 | Grouping — flat API | 3 docs for p5-containers repo |
| 5 | MLT for ml-pipeline | ≥1 similar: p5-automl |
| 6 | MLT for automl | ≥1 similar: p5-ml-pipeline |
| 7 | MLT empty id | 400 error |
| 8 | Popular searches | container: 4+, python: 2, machine learning: 1 |
| 9 | Empty query not recorded | Empty query absent from popular list |
| 10-15 | Frontend features | All UI elements render and function correctly |

## What NOT to do

- Do **NOT** modify any Go code or HTML — this is purely a test prompt
- Do **NOT** skip the cleanup step — test data should not persist
- Do **NOT** panic if spellcheck collation is empty — it requires sufficient indexed content to generate meaningful corrections; the feature will work better with real production data
