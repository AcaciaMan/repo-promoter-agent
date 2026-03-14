# Prompt: Solr Verification — Test Document Round-Trip

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. I'm replacing SQLite with **Apache Solr 10** as the sole data store. This is **Phase 1, prompt 3 of 3** — the final verification step for local Solr setup.

**Prerequisites**: Prompts 54–55 are complete — Solr 10 is running locally, the `promotions` core exists with all 20 custom fields and 9 copy-field rules.

This prompt posts a realistic test document, queries it back in multiple ways, and confirms the full setup is working before we move to Phase 2 (implementing the Go `SolrStore`).

## Your task

Post a test document matching the `Promotion` struct, verify it can be retrieved by direct lookup, full-text search, and listing — then clean up.

## Requirements

### 1. POST a realistic test document

Post a document that mirrors what the Go app will produce. Note: the `id` field is set to the `repo_url` value (this is how upsert works — same `id` replaces the old document).

```powershell
$testDoc = @(
    @{
        id               = "https://github.com/example/kube-helper"
        repo_url         = "https://github.com/example/kube-helper"
        repo_name        = "kube-helper"
        headline         = "Simplify Kubernetes deployments with one command"
        summary          = "kube-helper is a CLI tool that automates common Kubernetes deployment patterns. It reduces boilerplate YAML and provides sensible defaults for production workloads."
        key_benefits     = @(
            "Reduce deployment YAML by 80%"
            "Built-in health checks and rollback"
            "Works with any Kubernetes cluster"
        )
        tags             = @("kubernetes", "devops", "cli", "golang")
        twitter_posts    = @(
            "Tired of writing Kubernetes YAML? kube-helper cuts your deployment config by 80%. Try it now!"
            "Deploy to Kubernetes with one command. kube-helper handles the rest."
        )
        linkedin_post    = "Excited to share kube-helper — an open-source CLI tool that simplifies Kubernetes deployments. If you manage production clusters and are tired of boilerplate YAML, give it a try."
        call_to_action   = "Star the repo and try kube-helper on your next deployment!"
        target_channel   = "general"
        target_audience  = "backend developers working with Kubernetes"
        created_at       = "2026-03-14T12:00:00Z"
        stars            = 142
        forks            = 23
        watchers         = 15
        views_14d_total  = 1200
        views_14d_unique = 340
        clones_14d_total = 85
        clones_14d_unique = 42
        analysis_json    = '{"health":"strong","recommendation":"promote on DevOps communities"}'
    }
) | ConvertTo-Json -Depth 4

Invoke-WebRequest -Uri "http://localhost:8983/solr/promotions/update/json/docs?commit=true" `
    -Method POST `
    -ContentType "application/json" `
    -Body $testDoc `
    -UseBasicParsing
```

Confirm HTTP 200 response.

### 2. Retrieve by direct query (exact match on id)

```powershell
$response = Invoke-WebRequest -Uri "http://localhost:8983/solr/promotions/select?q=id:%22https://github.com/example/kube-helper%22&wt=json" -UseBasicParsing
$doc = ($response.Content | ConvertFrom-Json).response.docs[0]
Write-Host "repo_name: $($doc.repo_name)"
Write-Host "headline: $($doc.headline)"
Write-Host "tags: $($doc.tags -join ', ')"
Write-Host "stars: $($doc.stars)"
```

Expected: Outputs match the posted values.

### 3. Full-text search (via `_text_` catch-all)

Search for a word that appears in the summary:

```powershell
$response = Invoke-WebRequest -Uri "http://localhost:8983/solr/promotions/select?q=kubernetes&wt=json" -UseBasicParsing
$result = $response.Content | ConvertFrom-Json
Write-Host "Found $($result.response.numFound) document(s)"
Write-Host "First match: $($result.response.docs[0].repo_name)"
```

Expected: `numFound` = 1, match is `kube-helper`.

### 4. Search by tag (exact facet match)

Search for documents tagged "devops":

```powershell
$response = Invoke-WebRequest -Uri "http://localhost:8983/solr/promotions/select?q=tags:devops&wt=json" -UseBasicParsing
$result = $response.Content | ConvertFrom-Json
Write-Host "Found $($result.response.numFound) document(s) with tag 'devops'"
```

Expected: `numFound` = 1.

### 5. List all documents sorted by `created_at` descending

This is the Solr equivalent of the `List()` method:

```powershell
$response = Invoke-WebRequest -Uri "http://localhost:8983/solr/promotions/select?q=*:*&sort=created_at+desc&rows=10&wt=json" -UseBasicParsing
$result = $response.Content | ConvertFrom-Json
Write-Host "Total documents: $($result.response.numFound)"
Write-Host "First: $($result.response.docs[0].repo_name)"
```

Expected: Total = 1, first (and only) document is `kube-helper`.

### 6. Verify upsert — POST same `id`, different data

Post a document with the same `id` but a changed headline:

```powershell
$updatedDoc = @(
    @{
        id               = "https://github.com/example/kube-helper"
        repo_url         = "https://github.com/example/kube-helper"
        repo_name        = "kube-helper"
        headline         = "UPDATED: The easiest way to deploy to Kubernetes"
        summary          = "kube-helper is a CLI tool that automates common Kubernetes deployment patterns."
        key_benefits     = @("Reduce deployment YAML by 80%")
        tags             = @("kubernetes", "cli")
        twitter_posts    = @("Deploy to Kubernetes with one command.")
        linkedin_post    = "Check out kube-helper."
        call_to_action   = "Star the repo!"
        target_channel   = "twitter"
        target_audience  = "devops engineers"
        created_at       = "2026-03-14T13:00:00Z"
        stars            = 150
        forks            = 25
        watchers         = 16
        views_14d_total  = 1300
        views_14d_unique = 360
        clones_14d_total = 90
        clones_14d_unique = 45
        analysis_json    = '{"health":"strong"}'
    }
) | ConvertTo-Json -Depth 4

Invoke-WebRequest -Uri "http://localhost:8983/solr/promotions/update/json/docs?commit=true" `
    -Method POST `
    -ContentType "application/json" `
    -Body $updatedDoc `
    -UseBasicParsing
```

Then verify there is still only 1 document and the headline changed:

```powershell
$response = Invoke-WebRequest -Uri "http://localhost:8983/solr/promotions/select?q=*:*&wt=json" -UseBasicParsing
$result = $response.Content | ConvertFrom-Json
Write-Host "Total documents: $($result.response.numFound)"
Write-Host "Headline: $($result.response.docs[0].headline)"
```

Expected: `numFound` = 1, headline starts with "UPDATED:".

### 7. Clean up — delete the test document

```powershell
$deleteCmd = '{"delete": {"id": "https://github.com/example/kube-helper"}, "commit": {}}'

Invoke-WebRequest -Uri "http://localhost:8983/solr/promotions/update" `
    -Method POST `
    -ContentType "application/json" `
    -Body $deleteCmd `
    -UseBasicParsing
```

Verify empty:

```powershell
$response = Invoke-WebRequest -Uri "http://localhost:8983/solr/promotions/select?q=*:*&wt=json" -UseBasicParsing
$result = $response.Content | ConvertFrom-Json
Write-Host "Total documents after cleanup: $($result.response.numFound)"
```

Expected: `numFound` = 0.

## Verification checklist

- [ ] Test document POSTs successfully (HTTP 200)
- [ ] Direct ID lookup returns the correct document with all fields
- [ ] Full-text search for "kubernetes" finds the document via `_text_` copy fields
- [ ] Tag filter `tags:devops` returns exact match
- [ ] `*:*` with `sort=created_at desc` returns documents in expected order
- [ ] Upsert: re-posting same `id` replaces the document (total count stays 1)
- [ ] Delete by `id` removes the document (total count = 0)

## Notes

- This test validates every operation the Go `SolrStore` will need: insert, upsert, full-text search, list, and delete.
- The `id` = `repo_url` strategy is confirmed working for upsert. No auto-generated IDs needed.
- Multi-valued fields (`key_benefits`, `tags`, `twitter_posts`) are passed as JSON arrays and Solr handles them natively.
- `analysis_json` is stored as a plain string — the Go code will parse it to `json.RawMessage` on read.
- After this prompt, Phase 1 is complete and we move to Phase 2: implementing the Go `SolrStore` in `internal/store/store.go`.
