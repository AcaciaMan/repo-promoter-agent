# Prompt: Phase 4 — Analysis Search & Autocomplete Smoke Test

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. This is **Phase 4, prompt 5 of 5** for indexing analysis data and autocomplete.

**Prerequisites**: Prompts 75–78 are complete. The full stack now supports:
- **Solr schema**: 5 new indexed `analysis_*` fields (text_general) with copyField rules to `_text_`
- **Store.Save()**: Deserializes `AnalysisJSON` and populates `analysis_value_proposition`, `analysis_ideal_audience`, `analysis_key_features`, `analysis_differentiators`, `analysis_positioning` in the Solr document
- **Store.Search()**: `qf` includes analysis fields with boosts; `hl.fl` includes analysis fields for highlighting
- **Solr Suggester**: Configured with `AnalyzingInfixLookupFactory` on `headline`, `/suggest` handler enabled
- **Store.Suggest()**: Calls `/suggest`, parses response into `[]Suggestion`
- **Handler**: `GET /api/suggest?q=prefix` returns JSON array of `[{"term":"...","weight":N}]`
- **Frontend**: Search input has autocomplete dropdown with 300ms debounce, keyboard nav, mouse click

## Your task

Run an end-to-end smoke test to verify that:
1. Analysis data is searchable (terms appearing only in analysis fields produce results)
2. Autocomplete suggestions work via the API and the frontend

## Prerequisites check

```powershell
# 1. Verify Solr is running
try { $r = Invoke-WebRequest -Uri "http://localhost:8983/solr/promotions/admin/ping" -UseBasicParsing -TimeoutSec 5; "Solr OK: $($r.StatusCode)" } catch { "Solr not running — start it first" }

# 2. Verify analysis fields exist
$schema = Invoke-RestMethod -Uri "http://localhost:8983/solr/promotions/schema/fields" -Method GET
$schema.fields | Where-Object { $_.name -like "analysis_*" } | ForEach-Object { "$($_.name)  type=$($_.type)  indexed=$($_.indexed)" }
# Should list: analysis_value_proposition, analysis_ideal_audience, analysis_key_features, analysis_differentiators, analysis_positioning

# 3. Verify suggest handler exists
try { $r = Invoke-RestMethod -Uri "http://localhost:8983/solr/promotions/suggest?suggest.q=test" -Method GET; "Suggest handler OK" } catch { "Suggest handler not configured — run prompt 77 Solr config first" }

# 4. Build the app
go build ./...
```

## Test plan

### Step 1 — Index test documents with analysis data

These docs have analysis fields populated, including unique terms that do NOT appear in normal promotion fields — this lets us verify that analysis data is actually being searched.

```powershell
$solrBase = "http://localhost:8983/solr/promotions/update/json/docs?commit=true"

# Doc 1: Has analysis with unique term "microservice orchestration" only in value_proposition
$doc1 = @{
    id = "https://github.com/test/analysis-search-alpha"
    repo_url = "https://github.com/test/analysis-search-alpha"
    repo_name = "analysis-search-alpha"
    headline = "Container Management Toolkit"
    summary = "Deploy and manage containers effortlessly with simple YAML configs."
    tags = @("docker", "containers", "devops")
    target_channel = "general"
    created_at = "2026-03-14T10:00:00Z"
    stars = 320; forks = 40; watchers = 30
    views_14d_total = 500; views_14d_unique = 200; clones_14d_total = 30; clones_14d_unique = 15
    key_benefits = @("simple YAML", "multi-cloud support"); twitter_posts = @("Docker made easy!"); linkedin_post = "Container toolkit."; call_to_action = "Try it today."; target_audience = "DevOps engineers"; readme = "Readme."
    analysis_json = '{"primary_value_proposition":"Simplifies microservice orchestration across cloud providers","ideal_audience":["platform engineers","SRE teams"],"key_features":["declarative container configs","multi-cloud deployment"],"differentiators":["zero-downtime rolling updates","built-in service mesh"],"recommended_positioning_angle":["position as cloud-agnostic orchestrator"]}'
    analysis_value_proposition = "Simplifies microservice orchestration across cloud providers"
    analysis_ideal_audience = @("platform engineers", "SRE teams")
    analysis_key_features = @("declarative container configs", "multi-cloud deployment")
    analysis_differentiators = @("zero-downtime rolling updates", "built-in service mesh")
    analysis_positioning = @("position as cloud-agnostic orchestrator")
} | ConvertTo-Json -Depth 3
Invoke-RestMethod -Uri $solrBase -Method POST -ContentType "application/json" -Body $doc1

# Doc 2: Has unique term "geospatial indexing" only in analysis_key_features
$doc2 = @{
    id = "https://github.com/test/analysis-search-beta"
    repo_url = "https://github.com/test/analysis-search-beta"
    repo_name = "analysis-search-beta"
    headline = "Fast Database Engine for IoT Data"
    summary = "Time-series database optimized for sensor data ingestion at scale."
    tags = @("database", "iot", "timeseries")
    target_channel = "twitter"
    created_at = "2026-03-14T11:00:00Z"
    stars = 850; forks = 90; watchers = 70
    views_14d_total = 1200; views_14d_unique = 500; clones_14d_total = 60; clones_14d_unique = 30
    key_benefits = @("sub-millisecond writes", "compression"); twitter_posts = @("IoT data done right!"); linkedin_post = "IoT database engine."; call_to_action = "Star us."; target_audience = "IoT developers"; readme = "Readme."
    analysis_json = '{"primary_value_proposition":"Purpose-built for high-throughput sensor pipelines","ideal_audience":["IoT engineers","data platform teams"],"key_features":["geospatial indexing","columnar compression"],"differentiators":["10x faster writes than Postgres for time-series"],"recommended_positioning_angle":["lead with performance benchmarks"]}'
    analysis_value_proposition = "Purpose-built for high-throughput sensor pipelines"
    analysis_ideal_audience = @("IoT engineers", "data platform teams")
    analysis_key_features = @("geospatial indexing", "columnar compression")
    analysis_differentiators = @("10x faster writes than Postgres for time-series")
    analysis_positioning = @("lead with performance benchmarks")
} | ConvertTo-Json -Depth 3
Invoke-RestMethod -Uri $solrBase -Method POST -ContentType "application/json" -Body $doc2

# Doc 3: Has unique term "compliance auditing" only in analysis_differentiators
$doc3 = @{
    id = "https://github.com/test/analysis-search-gamma"
    repo_url = "https://github.com/test/analysis-search-gamma"
    repo_name = "analysis-search-gamma"
    headline = "Enterprise Access Control Library"
    summary = "Fine-grained RBAC and ABAC for Go services with built-in middleware."
    tags = @("go", "security", "rbac")
    target_channel = "linkedin"
    created_at = "2026-03-14T12:00:00Z"
    stars = 180; forks = 20; watchers = 15
    views_14d_total = 300; views_14d_unique = 150; clones_14d_total = 20; clones_14d_unique = 10
    key_benefits = @("declarative policies", "middleware integration"); twitter_posts = @("Secure your APIs!"); linkedin_post = "Access control for Go."; call_to_action = "Get started."; target_audience = "Backend developers"; readme = "Readme."
    analysis_json = '{"primary_value_proposition":"Enterprise-grade access control with audit trail","ideal_audience":["security engineers","compliance officers"],"key_features":["policy-as-code engine","LDAP integration"],"differentiators":["built-in compliance auditing with exportable reports"],"recommended_positioning_angle":["emphasize regulatory compliance readiness"]}'
    analysis_value_proposition = "Enterprise-grade access control with audit trail"
    analysis_ideal_audience = @("security engineers", "compliance officers")
    analysis_key_features = @("policy-as-code engine", "LDAP integration")
    analysis_differentiators = @("built-in compliance auditing with exportable reports")
    analysis_positioning = @("emphasize regulatory compliance readiness")
} | ConvertTo-Json -Depth 3
Invoke-RestMethod -Uri $solrBase -Method POST -ContentType "application/json" -Body $doc3

# Doc 4: No analysis — tests that docs without analysis still work
$doc4 = @{
    id = "https://github.com/test/analysis-search-delta"
    repo_url = "https://github.com/test/analysis-search-delta"
    repo_name = "analysis-search-delta"
    headline = "Markdown Preview Server"
    summary = "Live-reload markdown preview for writers and documentation teams."
    tags = @("markdown", "documentation", "preview")
    target_channel = "general"
    created_at = "2026-03-14T13:00:00Z"
    stars = 60; forks = 5; watchers = 8
    views_14d_total = 100; views_14d_unique = 50; clones_14d_total = 10; clones_14d_unique = 5
    key_benefits = @("live reload", "GFM support"); twitter_posts = @("Preview your docs!"); linkedin_post = "Markdown preview."; call_to_action = "Install now."; target_audience = "Technical writers"; readme = "Readme."
} | ConvertTo-Json -Depth 3
Invoke-RestMethod -Uri $solrBase -Method POST -ContentType "application/json" -Body $doc4

# Rebuild suggestion index
Invoke-RestMethod -Uri "http://localhost:8983/solr/promotions/suggest?suggest.build=true" -Method GET | Out-Null
"Test documents indexed and suggester rebuilt."
```

### Step 2 — Start the server

```powershell
# Make sure port 8080 is free
Get-NetTCPConnection -LocalPort 8080 -ErrorAction SilentlyContinue | ForEach-Object { Stop-Process -Id $_.OwningProcess -Force -ErrorAction SilentlyContinue }; Start-Sleep -Seconds 1

# Start server (in background or separate terminal)
go run ./cmd/server/main.go
```

### Step 3 — API tests for analysis field search

```powershell
# Test 1: Search for "microservice orchestration" — only in analysis_value_proposition
$r = Invoke-RestMethod -Uri "http://localhost:8080/api/search?q=microservice%20orchestration" -Method GET
"Test 1 — microservice orchestration: $($r.count) results"
if($r.count -gt 0){ "  Match: $($r.results[0].repo_name)" } else { "  FAIL: expected analysis-search-alpha" }

# Test 2: Search for "geospatial indexing" — only in analysis_key_features
$r = Invoke-RestMethod -Uri "http://localhost:8080/api/search?q=geospatial%20indexing" -Method GET
"Test 2 — geospatial indexing: $($r.count) results"
if($r.count -gt 0){ "  Match: $($r.results[0].repo_name)" } else { "  FAIL: expected analysis-search-beta" }

# Test 3: Search for "compliance auditing" — only in analysis_differentiators
$r = Invoke-RestMethod -Uri "http://localhost:8080/api/search?q=compliance%20auditing" -Method GET
"Test 3 — compliance auditing: $($r.count) results"
if($r.count -gt 0){ "  Match: $($r.results[0].repo_name)" } else { "  FAIL: expected analysis-search-gamma" }

# Test 4: Search for "platform engineers" — only in analysis_ideal_audience
$r = Invoke-RestMethod -Uri "http://localhost:8080/api/search?q=platform%20engineers" -Method GET
"Test 4 — platform engineers: $($r.count) results"
if($r.count -gt 0){ "  Match: $($r.results[0].repo_name)" } else { "  FAIL: expected analysis-search-alpha" }

# Test 5: Search for "markdown" — normal field, no analysis needed (doc4 has no analysis)
$r = Invoke-RestMethod -Uri "http://localhost:8080/api/search?q=markdown" -Method GET
"Test 5 — markdown (no analysis): $($r.count) results"
if($r.count -gt 0){ "  Match: $($r.results[0].repo_name)" } else { "  FAIL: expected analysis-search-delta" }

# Test 6: Verify highlights include analysis fields
$r = Invoke-RestMethod -Uri "http://localhost:8080/api/search?q=geospatial" -Method GET
if($r.highlights){
    $docHl = $r.highlights."https://github.com/test/analysis-search-beta"
    if($docHl -and ($docHl | Get-Member -MemberType NoteProperty | Where-Object { $_.Name -like "analysis_*" })){
        "Test 6 — Analysis highlight: PASS"
    } else { "Test 6 — Analysis highlight: no analysis_* highlight found (may be OK depending on Solr config)" }
} else { "Test 6 — No highlights at all — check hl config" }
```

### Step 4 — API tests for autocomplete

```powershell
# Test 7: Suggest with a known prefix
$r = Invoke-RestMethod -Uri "http://localhost:8080/api/suggest?q=container" -Method GET
"Test 7 — suggest 'container': $($r.Count) suggestions"
if($r.Count -gt 0){ "  First: $($r[0].term)" } else { "  No suggestions (may need more indexed data for the suggester dictionary)" }

# Test 8: Suggest with another prefix
$r = Invoke-RestMethod -Uri "http://localhost:8080/api/suggest?q=fast" -Method GET
"Test 8 — suggest 'fast': $($r.Count) suggestions"
if($r.Count -gt 0){ "  First: $($r[0].term)" } else { "  No suggestions" }

# Test 9: Empty query returns empty array
$r = Invoke-RestMethod -Uri "http://localhost:8080/api/suggest?q=" -Method GET
"Test 9 — empty suggest: $($r.Count) items (expected 0)"

# Test 10: Single char returns empty (backend requires non-empty, frontend filters <2)
$r = Invoke-RestMethod -Uri "http://localhost:8080/api/suggest?q=a" -Method GET
"Test 10 — single char suggest: $($r.Count) items"
```

### Step 5 — Frontend verification

Open `http://localhost:8080` in a browser and test:

1. **Switch to Search tab**
2. **Search for "microservice orchestration"** → should find analysis-search-alpha
3. **Search for "geospatial"** → should find analysis-search-beta, check for `<mark>` highlight in analysis fields
4. **Type "con" in the search box** → after ~300ms, an autocomplete dropdown should appear with suggestions (if the suggester has indexed enough data)
5. **Press Arrow Down** → highlight should move through suggestions
6. **Press Enter on a highlighted suggestion** → search input should fill and search should execute
7. **Press Escape** → dropdown should close
8. **Click a suggestion** → search input should fill and search should execute
9. **Click outside the dropdown** → dropdown should close
10. **Type a single character** → no dropdown should appear (minimum 2 chars)
11. **Search for "markdown"** → should find analysis-search-delta (no analysis doc still works)
12. **Clear search and browse** → all test docs should appear in the list

### Step 6 — Cleanup test data

```powershell
$deleteBody = '{"delete": {"query": "repo_url:https\\://github.com/test/analysis-search-*"}}'
Invoke-RestMethod -Uri "http://localhost:8983/solr/promotions/update?commit=true" -Method POST -ContentType "application/json" -Body $deleteBody
"Test documents cleaned up."

# Rebuild suggester after cleanup
Invoke-RestMethod -Uri "http://localhost:8983/solr/promotions/suggest?suggest.build=true" -Method GET | Out-Null
```

## Expected results summary

| # | Test | Expected Result |
|---|------|----------------|
| 1 | Search "microservice orchestration" | 1 result: analysis-search-alpha |
| 2 | Search "geospatial indexing" | 1 result: analysis-search-beta |
| 3 | Search "compliance auditing" | 1 result: analysis-search-gamma |
| 4 | Search "platform engineers" | 1 result: analysis-search-alpha |
| 5 | Search "markdown" | 1 result: analysis-search-delta |
| 6 | Highlights include analysis fields | `<mark>` tags in analysis matches |
| 7 | Suggest "container" | ≥1 suggestion with "Container" |
| 8 | Suggest "fast" | ≥1 suggestion with "Fast" |
| 9 | Empty suggest | Empty array `[]` |
| 10 | Single char suggest | Empty or minimal results |
| 11-12 | Frontend autocomplete UX | Dropdown appears, keyboard/mouse nav works |

## What NOT to do

- Do **NOT** modify any Go code or HTML — this is purely a test prompt
- Do **NOT** skip the cleanup step — test data should not persist
