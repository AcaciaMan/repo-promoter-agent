# Prompt: Solr Schema — Create Core and Define Fields

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. I'm replacing SQLite with **Apache Solr 10** as the sole data store. This is **Phase 1, prompt 2 of 3** for local Solr setup.

**Prerequisite**: Prompt 54 is complete — Java 21 and Solr 10 are installed and Solr is running at `http://localhost:8983`.

This prompt creates the `promotions` core and defines the schema to match the existing Go `Promotion` struct in `internal/store/store.go`.

## Current Promotion struct (from `internal/store/store.go`)

```go
type Promotion struct {
    ID              int64           `json:"id"`
    RepoURL         string          `json:"repo_url"`
    RepoName        string          `json:"repo_name"`
    Headline        string          `json:"headline"`
    Summary         string          `json:"summary"`
    KeyBenefits     []string        `json:"key_benefits"`
    Tags            []string        `json:"tags"`
    TwitterPosts    []string        `json:"twitter_posts"`
    LinkedInPost    string          `json:"linkedin_post"`
    CallToAction    string          `json:"call_to_action"`
    TargetChannel   string          `json:"target_channel"`
    TargetAudience  string          `json:"target_audience"`
    CreatedAt       time.Time       `json:"created_at"`
    Stars           int             `json:"stars"`
    Forks           int             `json:"forks"`
    Watchers        int             `json:"watchers"`
    Views14dTotal   int             `json:"views_14d_total"`
    Views14dUnique  int             `json:"views_14d_unique"`
    Clones14dTotal  int             `json:"clones_14d_total"`
    Clones14dUnique int             `json:"clones_14d_unique"`
    AnalysisJSON    json.RawMessage `json:"analysis"`
}
```

## Your task

Create the Solr `promotions` core and define its schema via the Solr Schema API. Run all commands in the terminal.

## Requirements

### 1. Create the `promotions` core

```powershell
cd c:\solr\solr-10.x.x
bin\solr.cmd create -c promotions
```

Verify it was created:

```powershell
Invoke-WebRequest -Uri "http://localhost:8983/solr/admin/cores?action=STATUS&core=promotions" -UseBasicParsing
```

### 2. Define schema fields via the Schema API

Use the Solr Schema API to add all fields. The unique key is `id` (Solr's default) — we will use `repo_url` as the value for `id` to enable upsert-by-URL.

Send a single POST request to add all fields at once:

```powershell
$schemaUpdate = @{
    "add-field" = @(
        @{ name = "repo_url";         type = "string";       stored = $true; indexed = $true }
        @{ name = "repo_name";        type = "text_general";  stored = $true }
        @{ name = "headline";         type = "text_general";  stored = $true }
        @{ name = "summary";          type = "text_general";  stored = $true }
        @{ name = "key_benefits";     type = "text_general";  stored = $true; multiValued = $true }
        @{ name = "tags";             type = "string";        stored = $true; indexed = $true; multiValued = $true }
        @{ name = "twitter_posts";    type = "text_general";  stored = $true; multiValued = $true }
        @{ name = "linkedin_post";    type = "text_general";  stored = $true }
        @{ name = "call_to_action";   type = "text_general";  stored = $true }
        @{ name = "target_channel";   type = "string";        stored = $true; indexed = $true }
        @{ name = "target_audience";  type = "text_general";  stored = $true }
        @{ name = "created_at";       type = "pdate";         stored = $true; indexed = $true }
        @{ name = "stars";            type = "pint";          stored = $true; indexed = $true }
        @{ name = "forks";            type = "pint";          stored = $true; indexed = $true }
        @{ name = "watchers";         type = "pint";          stored = $true; indexed = $true }
        @{ name = "views_14d_total";  type = "pint";          stored = $true }
        @{ name = "views_14d_unique"; type = "pint";          stored = $true }
        @{ name = "clones_14d_total"; type = "pint";          stored = $true }
        @{ name = "clones_14d_unique";type = "pint";          stored = $true }
        @{ name = "analysis_json";    type = "string";        stored = $true; indexed = $false }
    )
} | ConvertTo-Json -Depth 4

Invoke-WebRequest -Uri "http://localhost:8983/solr/promotions/schema" `
    -Method POST `
    -ContentType "application/json" `
    -Body $schemaUpdate `
    -UseBasicParsing
```

### 3. Verify the schema

Retrieve the schema and confirm all fields exist:

```powershell
$response = Invoke-WebRequest -Uri "http://localhost:8983/solr/promotions/schema/fields" -UseBasicParsing
$response.Content | ConvertFrom-Json | Select-Object -ExpandProperty fields | Format-Table name, type, multiValued
```

Expected fields (in addition to Solr's built-in fields like `id`, `_version_`, `_text_`):

| Field Name | Type | Multi-Valued | Notes |
|---|---|---|---|
| `repo_url` | string | no | Indexed — used for exact-match lookups |
| `repo_name` | text_general | no | Full-text searchable |
| `headline` | text_general | no | Full-text searchable |
| `summary` | text_general | no | Full-text searchable |
| `key_benefits` | text_general | **yes** | Array of benefits |
| `tags` | string | **yes** | Exact-match for faceting |
| `twitter_posts` | text_general | **yes** | Array of tweets |
| `linkedin_post` | text_general | no | Full-text searchable |
| `call_to_action` | text_general | no | Full-text searchable |
| `target_channel` | string | no | Exact-match filter |
| `target_audience` | text_general | no | Full-text searchable |
| `created_at` | pdate | no | Sortable date |
| `stars` | pint | no | Numeric metric |
| `forks` | pint | no | Numeric metric |
| `watchers` | pint | no | Numeric metric |
| `views_14d_total` | pint | no | Traffic metric |
| `views_14d_unique` | pint | no | Traffic metric |
| `clones_14d_total` | pint | no | Traffic metric |
| `clones_14d_unique` | pint | no | Traffic metric |
| `analysis_json` | string | no | Stored only (not indexed) — raw JSON blob |

### 4. Configure copy fields for default search

Copy all `text_general` fields into the built-in `_text_` catch-all field so that unqualified queries (`q=kubernetes`) search across all text:

```powershell
$copyFields = @{
    "add-copy-field" = @(
        @{ source = "repo_name";       dest = "_text_" }
        @{ source = "headline";        dest = "_text_" }
        @{ source = "summary";         dest = "_text_" }
        @{ source = "key_benefits";    dest = "_text_" }
        @{ source = "twitter_posts";   dest = "_text_" }
        @{ source = "linkedin_post";   dest = "_text_" }
        @{ source = "call_to_action";  dest = "_text_" }
        @{ source = "target_audience"; dest = "_text_" }
        @{ source = "tags";            dest = "_text_" }
    )
} | ConvertTo-Json -Depth 4

Invoke-WebRequest -Uri "http://localhost:8983/solr/promotions/schema" `
    -Method POST `
    -ContentType "application/json" `
    -Body $copyFields `
    -UseBasicParsing
```

### 5. Verify copy fields

```powershell
$response = Invoke-WebRequest -Uri "http://localhost:8983/solr/promotions/schema/copyfields" -UseBasicParsing
$response.Content | ConvertFrom-Json | Select-Object -ExpandProperty copyFields | Format-Table source, dest
```

All 9 sources should be listed with dest `_text_`.

## Verification checklist

- [ ] `promotions` core exists and responds at `http://localhost:8983/solr/promotions/select?q=*:*`
- [ ] All 20 custom fields are visible in the schema
- [ ] `tags` and `key_benefits` and `twitter_posts` are multiValued
- [ ] `analysis_json` is stored but not indexed
- [ ] 9 copy-field rules target `_text_`

## Notes

- The `id` field is Solr's default unique key — we'll set its value to `repo_url` when posting documents. This enables upsert: posting a document with the same `id` replaces the previous one.
- `tags` is typed `string` (not `text_general`) so it can be used for exact-match faceting. It is still copied to `_text_` for full-text search.
- `text_general` fields use Solr's default analyzer: case-insensitive, standard tokenizer.
- `pint` and `pdate` are Solr's point-based numeric/date types (recommended in Solr 10).
