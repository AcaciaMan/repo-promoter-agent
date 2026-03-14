# Prompt: Solr Migration — End-to-End Smoke Test

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. Phase 2 (prompts 57–60) replaced SQLite with Solr. Prompt 61 cleaned up all stale SQLite references. This is **Phase 3, prompt 2 of 2** — a full end-to-end smoke test to confirm the app works correctly with Solr.

**Current state**:
- `internal/store/store.go` — Solr-backed ✓
- `cmd/server/main.go` — uses `SOLR_URL` / `SOLR_CORE` ✓
- `go.mod` — no SQLite dependency ✓
- `README.md` — updated to reference Solr ✓
- All stale SQLite references removed ✓

## Prerequisites

Before running this test:
1. **Solr 10 must be running** at `http://localhost:8983`
2. **The `promotions` core must exist** with the schema from prompt 55
3. **Required env vars** must be set: `AGENT_ENDPOINT`, `AGENT_ACCESS_KEY`
4. **Optional env vars** if you want to test traffic metrics: `GITHUB_TOKEN`

Verify Solr is up:
```powershell
try { $r = Invoke-WebRequest -Uri "http://localhost:8983/solr/promotions/admin/ping" -UseBasicParsing; "Solr OK: $($r.StatusCode)" } catch { "Solr not reachable: $_" }
```

## Your task

Start the server and perform a full manual smoke test covering every user-facing flow: generate, search, list, upsert, and the frontend UI.

## Test plan

### 1. Start the server

```powershell
cd C:\work\GitHub\repo-promoter-agent
go run ./cmd/server/main.go
```

Expected log output should include:
- `Connected to Solr at http://localhost:8983 (core: promotions)`
- `Server listening on http://localhost:8080`
- No errors or panics

### 2. Test: Generate promotional content

```powershell
$body = @{
    repo_url = "https://github.com/AcaciaMan/village-square"
    target_channel = "twitter"
    target_audience = "community organizers"
} | ConvertTo-Json

$response = Invoke-WebRequest -Uri "http://localhost:8080/api/generate" `
    -Method POST `
    -ContentType "application/json" `
    -Body $body `
    -UseBasicParsing

Write-Host "Status: $($response.StatusCode)"
$result = $response.Content | ConvertFrom-Json
Write-Host "repo_name: $($result.repo_name)"
Write-Host "headline: $($result.headline)"
Write-Host "tags: $($result.tags -join ', ')"
Write-Host "tweets: $($result.twitter_posts.Count)"
```

**Expected**:
- HTTP 200
- `repo_name` is non-empty
- `headline` is non-empty
- `tags` is a non-empty array
- `twitter_posts` has 1+ items

### 3. Test: Verify document exists in Solr directly

```powershell
$solrResp = Invoke-WebRequest -Uri "http://localhost:8983/solr/promotions/select?q=*:*&wt=json" -UseBasicParsing
$solrResult = $solrResp.Content | ConvertFrom-Json
Write-Host "Documents in Solr: $($solrResult.response.numFound)"
Write-Host "First doc repo_name: $($solrResult.response.docs[0].repo_name)"
```

**Expected**: `numFound` >= 1

### 4. Test: Search via API

```powershell
$searchResp = Invoke-WebRequest -Uri "http://localhost:8080/api/search?q=village" -UseBasicParsing
$searchResult = $searchResp.Content | ConvertFrom-Json
Write-Host "Search results: $($searchResult.count)"
Write-Host "First result: $($searchResult.results[0].repo_name)"
```

**Expected**: `count` >= 1, first result's `repo_name` matches

### 5. Test: List (no query) via API

```powershell
$listResp = Invoke-WebRequest -Uri "http://localhost:8080/api/search" -UseBasicParsing
$listResult = $listResp.Content | ConvertFrom-Json
Write-Host "Listed results: $($listResult.count)"
```

**Expected**: `count` >= 1

### 6. Test: Upsert — regenerate for the same repo

```powershell
$response2 = Invoke-WebRequest -Uri "http://localhost:8080/api/generate" `
    -Method POST `
    -ContentType "application/json" `
    -Body $body `
    -UseBasicParsing

Write-Host "Second generate status: $($response2.StatusCode)"

# Verify Solr has exactly 1 document for this repo_url (upsert, not duplicate)
$checkResp = Invoke-WebRequest -Uri 'http://localhost:8983/solr/promotions/select?q=repo_url:%22https://github.com/AcaciaMan/village-square%22&wt=json' -UseBasicParsing
$checkResult = $checkResp.Content | ConvertFrom-Json
Write-Host "Documents for this repo: $($checkResult.response.numFound)"
```

**Expected**: HTTP 200, `numFound` = 1 (upsert replaced the old doc)

### 7. Test: Generate for a second repo

```powershell
$body2 = @{
    repo_url = "https://github.com/golang/go"
    target_channel = "linkedin"
    target_audience = "software engineers"
} | ConvertTo-Json

$response3 = Invoke-WebRequest -Uri "http://localhost:8080/api/generate" `
    -Method POST `
    -ContentType "application/json" `
    -Body $body2 `
    -UseBasicParsing

Write-Host "Second repo generate status: $($response3.StatusCode)"

# Total docs should now be 2
$totalResp = Invoke-WebRequest -Uri "http://localhost:8983/solr/promotions/select?q=*:*&wt=json" -UseBasicParsing
$totalResult = $totalResp.Content | ConvertFrom-Json
Write-Host "Total documents in Solr: $($totalResult.response.numFound)"
```

**Expected**: HTTP 200, `numFound` = 2

### 8. Test: Search returns correct results

```powershell
# Search should find the Go repo
$searchGo = Invoke-WebRequest -Uri "http://localhost:8080/api/search?q=golang" -UseBasicParsing
$goResult = $searchGo.Content | ConvertFrom-Json
Write-Host "Search 'golang' results: $($goResult.count)"

# Search should find the village repo
$searchVillage = Invoke-WebRequest -Uri "http://localhost:8080/api/search?q=village" -UseBasicParsing
$villageResult = $searchVillage.Content | ConvertFrom-Json
Write-Host "Search 'village' results: $($villageResult.count)"
```

**Expected**: Each search returns >= 1 relevant result

### 9. Test: List respects ordering (most recent first)

```powershell
$listResp2 = Invoke-WebRequest -Uri "http://localhost:8080/api/search" -UseBasicParsing
$listResult2 = $listResp2.Content | ConvertFrom-Json
Write-Host "Total listed: $($listResult2.count)"
Write-Host "First (most recent): $($listResult2.results[0].repo_name)"
Write-Host "Second: $($listResult2.results[1].repo_name)"
```

**Expected**: The second repo generated (`golang/go`) appears first (most recent)

### 10. Test: Traffic metrics (if GITHUB_TOKEN is set)

Only if `GITHUB_TOKEN` is configured:

```powershell
$searchVillage2 = Invoke-WebRequest -Uri "http://localhost:8080/api/search?q=village" -UseBasicParsing
$v = ($searchVillage2.Content | ConvertFrom-Json).results[0]
Write-Host "views_14d_total: $($v.views_14d_total)"
Write-Host "clones_14d_total: $($v.clones_14d_total)"
```

**Expected**: For AcaciaMan repos, traffic metrics should be non-zero integers if the token has access. For other repos, they should be 0.

### 11. Test: Frontend UI

Open `http://localhost:8080` in a browser and verify:

- [ ] **Generate page** loads — URL input field, channel selector, generate button visible
- [ ] Pasting a repo URL and clicking Generate shows results (headline, tweets, LinkedIn post, tags)
- [ ] **Search page** loads — search bar visible
- [ ] Typing a query and searching shows result cards
- [ ] Copy buttons work on generated content
- [ ] No JavaScript console errors

### 12. Test: Special characters in search

```powershell
$specialSearch = Invoke-WebRequest -Uri "http://localhost:8080/api/search?q=C%2B%2B+(advanced)" -UseBasicParsing
Write-Host "Special char search status: $($specialSearch.StatusCode)"
```

**Expected**: HTTP 200, no 500 error (query is sanitized)

### 13. Test: No `.db` files created

```powershell
Get-ChildItem -Path "C:\work\GitHub\repo-promoter-agent" -Filter "*.db" -File
```

**Expected**: No results — the app no longer creates SQLite files.

## Verification checklist

- [ ] Server starts and connects to Solr without errors
- [ ] `POST /api/generate` creates a promotion in Solr
- [ ] `GET /api/search?q=...` returns relevant full-text search results
- [ ] `GET /api/search` (no query) lists all promotions, most recent first
- [ ] Upsert: regenerating for the same repo replaces the document (count stays 1)
- [ ] Multiple repos stored correctly (count reflects distinct repos)
- [ ] Special characters in search queries don't cause errors
- [ ] Frontend UI loads and all features work
- [ ] No `.db` files created at runtime
- [ ] Traffic metrics work for AcaciaMan repos (if `GITHUB_TOKEN` set)

## Clean up test data (optional)

If you want to clear the test documents from Solr after testing:

```powershell
$deleteAll = '{"delete":{"query":"*:*"},"commit":{}}'
Invoke-WebRequest -Uri "http://localhost:8983/solr/promotions/update" `
    -Method POST `
    -ContentType "application/json" `
    -Body $deleteAll `
    -UseBasicParsing
```

## Notes

- After this prompt, **Phase 3 is complete**. The app is fully Solr-backed, tested end-to-end, with no SQLite remnants.
- Phase 4 (DigitalOcean deployment) is deferred until you're ready.
- If any test fails, debug the specific issue and fix it before marking Phase 3 complete.
