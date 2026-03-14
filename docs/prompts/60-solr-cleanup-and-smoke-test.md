# Prompt: Solr Migration — Cleanup and End-to-End Smoke Test

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. I'm replacing SQLite with **Apache Solr 10** as the sole data store. This is **Phase 2, prompt 4 of 4** — the final cleanup and verification step.

**Prerequisites**: Prompts 57–59 are complete:
- `internal/store/store.go` — Solr-backed implementation ✓
- `internal/store/store_test.go` — integration tests passing ✓
- `cmd/server/main.go` — reads `SOLR_URL`/`SOLR_CORE`, instantiates Solr store ✓
- `go mod tidy` — SQLite dependency removed ✓

## Your task

Clean up leftover SQLite artifacts, update documentation, and perform a full end-to-end smoke test.

## Requirements

### 1. Delete leftover SQLite files

Check for and delete any `*.db` files in the workspace root:

```powershell
Get-ChildItem -Path "C:\work\GitHub\repo-promoter-agent" -Filter "*.db" -File
# If any found:
Remove-Item "C:\work\GitHub\repo-promoter-agent\promotions.db" -ErrorAction SilentlyContinue
```

### 2. Verify no SQLite references remain in code

```powershell
cd C:\work\GitHub\repo-promoter-agent
Select-String -Path "internal\**\*.go" -Pattern "sqlite|sql\.DB|sql\.Open|sql\.Rows|database/sql|modernc|FTS5|fts5" -Recurse
Select-String -Path "cmd\**\*.go" -Pattern "sqlite|sql\.DB|DB_PATH|dbPath|promotions\.db" -Recurse
```

Both should return **no results**. If any references are found, remove them.

### 3. Verify the build

```powershell
go build ./...
```

Must compile with zero errors.

### 4. Update README.md

Make these changes to `README.md`:

#### a) Replace SQLite references with Solr

Find and replace any mentions of:
- "SQLite" → "Apache Solr"
- "database file" → "Solr core"
- "FTS5" or "Full-Text Search (FTS5)" → "Solr full-text search"
- "promotions.db" → remove or replace with Solr reference

#### b) Update the Environment Variables table

Remove `DB_PATH` row. Add two new rows:

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `SOLR_URL` | No | `http://localhost:8983` | Solr server URL |
| `SOLR_CORE` | No | `promotions` | Solr core name |

#### c) Add Solr setup section

Add a section (after Environment Variables or before "Running") explaining local Solr prerequisites:

```markdown
## Solr Setup (Local Development)

The application requires Apache Solr 10 running locally.

### Prerequisites
- Java 21 (e.g., Eclipse Temurin or Microsoft Build of OpenJDK)
- Apache Solr 10 ([download](https://solr.apache.org/downloads.html))

### Quick Start
1. Start Solr: `bin\solr.cmd start`
2. Create the core: `bin\solr.cmd create -c promotions`
3. Apply the schema — see `docs/prompts/55-solr-schema-definition.md` for the full Schema API commands.
4. Set environment variables (or add to `.env`):
   ```
   SOLR_URL=http://localhost:8983
   SOLR_CORE=promotions
   ```
```

#### d) Update Project Structure listing

Replace the `internal/store/` entry. If it says something like:
```
internal/store/store.go  — SQLite persistence with FTS5
```
Change to:
```
internal/store/store.go  — Solr-backed store for promotional content
```

### 5. Update `docs/contracts/api-env-contract.md`

If this file lists `DB_PATH`, remove it and add `SOLR_URL` and `SOLR_CORE` with their defaults and descriptions.

### 6. End-to-end smoke test

With Solr running and the server started:

```powershell
# Start the server (in a separate terminal or background)
go run ./cmd/server/main.go
```

#### a) Generate content

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

$response.StatusCode  # Expected: 200
$result = $response.Content | ConvertFrom-Json
$result.repo_name     # Expected: non-empty
$result.headline      # Expected: non-empty
```

#### b) Verify it's in Solr

```powershell
$solrResponse = Invoke-WebRequest -Uri "http://localhost:8983/solr/promotions/select?q=*:*&wt=json" -UseBasicParsing
$solrResult = $solrResponse.Content | ConvertFrom-Json
$solrResult.response.numFound  # Expected: >= 1
```

#### c) Search via API

```powershell
$searchResponse = Invoke-WebRequest -Uri "http://localhost:8080/api/search?q=village" -UseBasicParsing
$searchResult = $searchResponse.Content | ConvertFrom-Json
$searchResult.count  # Expected: >= 1
$searchResult.results[0].repo_name  # Expected: repo name
```

#### d) List via API (no query)

```powershell
$listResponse = Invoke-WebRequest -Uri "http://localhost:8080/api/search" -UseBasicParsing
$listResult = $listResponse.Content | ConvertFrom-Json
$listResult.count  # Expected: >= 1
```

#### e) Upsert — generate again for same repo

```powershell
$response2 = Invoke-WebRequest -Uri "http://localhost:8080/api/generate" `
    -Method POST `
    -ContentType "application/json" `
    -Body $body `
    -UseBasicParsing

# Verify Solr still has only 1 document for this repo
$solrResponse2 = Invoke-WebRequest -Uri 'http://localhost:8983/solr/promotions/select?q=repo_url:"https://github.com/AcaciaMan/village-square"&wt=json' -UseBasicParsing
$solrCheck = $solrResponse2.Content | ConvertFrom-Json
$solrCheck.response.numFound  # Expected: 1 (upsert replaced the old one)
```

#### f) Check the frontend

Open `http://localhost:8080` in a browser. Verify:
- Generate page loads and works
- Search page loads and shows results

### 7. Verify no `*.db` files were created

```powershell
Get-ChildItem -Path "C:\work\GitHub\repo-promoter-agent" -Filter "*.db" -File
# Expected: no results
```

## Verification checklist

- [ ] No `*.db` files in the workspace
- [ ] No SQLite references in any `.go` files
- [ ] `go build ./...` compiles cleanly
- [ ] `go.mod` has no `modernc.org/sqlite` or related deps
- [ ] `README.md` updated: Solr env vars, setup instructions, no SQLite references
- [ ] `POST /api/generate` succeeds and stores in Solr
- [ ] `GET /api/search?q=...` returns results from Solr
- [ ] `GET /api/search` (no query) lists recent promotions
- [ ] Upsert works — same repo URL produces 1 document in Solr
- [ ] Frontend loads and works at `http://localhost:8080`

## Notes

- After this prompt, Phase 2 is complete. The app is 100% Solr-backed with no SQLite remnants.
- Phase 3 (DigitalOcean deployment) is deferred until you're ready.
- The `docs/prompts/10-sqlite-storage.md` prompt is historical documentation and should be kept as-is — it documents the original design decisions.
