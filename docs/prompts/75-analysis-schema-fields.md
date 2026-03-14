# Prompt: Add Indexed Solr Fields for Analysis Data

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. The app stores AI-generated promotional content for GitHub repos in **Apache Solr 10** and exposes full-text search via `GET /api/search?q=...`.

Phase 3 (highlighting & sort, prompts 71–74) is complete. This is **Phase 4, prompt 1 of 5** for indexing analysis data and autocomplete.

**Problem**: The `analysis_json` field is stored as a raw string in Solr but is **NOT indexed** — meaning the AI analysis output (value proposition, ideal audience, key features, differentiators, positioning angles) is invisible to search. Users searching for terms that appear only in analysis data get no results.

## Current state

### `AnalysisOutput` struct in `internal/agent/analysis.go`

```go
type AnalysisOutput struct {
    RepoURL                     string   `json:"repo_url"`
    RepoName                    string   `json:"repo_name"`
    PrimaryValueProposition     string   `json:"primary_value_proposition"`
    IdealAudience               []string `json:"ideal_audience"`
    KeyFeatures                 []string `json:"key_features"`
    Differentiators             []string `json:"differentiators"`
    RiskOrLimitations           []string `json:"risk_or_limitations"`
    SocialProofSignals          []string `json:"social_proof_signals"`
    RecommendedPositioningAngle []string `json:"recommended_positioning_angle"`
}
```

### `Store.Save()` — analysis section in `internal/store/store.go`

```go
if p.AnalysisJSON != nil {
    doc["analysis_json"] = string(p.AnalysisJSON)
}
```

Currently `analysis_json` is stored as a single opaque string. It's not searchable because it's defined as `stored=true, indexed=false` (or by default Solr doesn't tokenize unknown string fields for full-text purposes).

## Your task

Use the Solr Schema API to add dedicated, indexed fields for the key analysis data points. This makes analysis content searchable alongside the existing promotion fields.

## Requirements

### 1. Add new Solr fields via Schema API

Run the following commands in PowerShell to add the new fields to the `promotions` core. Use `text_general` field type for full-text searchability, and set `multiValued=true` for array fields.

```powershell
# Add analysis fields to Solr schema
$solrBase = "http://localhost:8983/solr/promotions/schema"

# 1. analysis_value_proposition — single-valued text_general
$body = '{"add-field": {"name": "analysis_value_proposition", "type": "text_general", "stored": true, "indexed": true, "multiValued": false}}'
Invoke-RestMethod -Uri $solrBase -Method POST -ContentType "application/json" -Body $body

# 2. analysis_ideal_audience — multiValued text_general
$body = '{"add-field": {"name": "analysis_ideal_audience", "type": "text_general", "stored": true, "indexed": true, "multiValued": true}}'
Invoke-RestMethod -Uri $solrBase -Method POST -ContentType "application/json" -Body $body

# 3. analysis_key_features — multiValued text_general
$body = '{"add-field": {"name": "analysis_key_features", "type": "text_general", "stored": true, "indexed": true, "multiValued": true}}'
Invoke-RestMethod -Uri $solrBase -Method POST -ContentType "application/json" -Body $body

# 4. analysis_differentiators — multiValued text_general
$body = '{"add-field": {"name": "analysis_differentiators", "type": "text_general", "stored": true, "indexed": true, "multiValued": true}}'
Invoke-RestMethod -Uri $solrBase -Method POST -ContentType "application/json" -Body $body

# 5. analysis_positioning — multiValued text_general (from recommended_positioning_angle)
$body = '{"add-field": {"name": "analysis_positioning", "type": "text_general", "stored": true, "indexed": true, "multiValued": true}}'
Invoke-RestMethod -Uri $solrBase -Method POST -ContentType "application/json" -Body $body
```

### 2. Add copyField rules to feed analysis data into `_text_` catch-all

```powershell
$solrBase = "http://localhost:8983/solr/promotions/schema"

$body = '{"add-copy-field": {"source": "analysis_value_proposition", "dest": "_text_"}}'
Invoke-RestMethod -Uri $solrBase -Method POST -ContentType "application/json" -Body $body

$body = '{"add-copy-field": {"source": "analysis_ideal_audience", "dest": "_text_"}}'
Invoke-RestMethod -Uri $solrBase -Method POST -ContentType "application/json" -Body $body

$body = '{"add-copy-field": {"source": "analysis_key_features", "dest": "_text_"}}'
Invoke-RestMethod -Uri $solrBase -Method POST -ContentType "application/json" -Body $body

$body = '{"add-copy-field": {"source": "analysis_differentiators", "dest": "_text_"}}'
Invoke-RestMethod -Uri $solrBase -Method POST -ContentType "application/json" -Body $body

$body = '{"add-copy-field": {"source": "analysis_positioning", "dest": "_text_"}}'
Invoke-RestMethod -Uri $solrBase -Method POST -ContentType "application/json" -Body $body
```

### 3. Verify the fields exist

```powershell
$schema = Invoke-RestMethod -Uri "http://localhost:8983/solr/promotions/schema/fields" -Method GET
$analysisFields = $schema.fields | Where-Object { $_.name -like "analysis_*" }
$analysisFields | ForEach-Object { "$($_.name)  type=$($_.type)  multiValued=$($_.multiValued)  indexed=$($_.indexed)" }
```

Expected output should list 5 new fields (plus the existing `analysis_json`) all with `type=text_general` and `indexed=True`.

### 4. Verify copyField rules

```powershell
$schema = Invoke-RestMethod -Uri "http://localhost:8983/solr/promotions/schema/copyfields" -Method GET
$schema.copyFields | Where-Object { $_.source -like "analysis_*" } | ForEach-Object { "$($_.source) -> $($_.dest)" }
```

Expected output: 5 rows, each `analysis_* -> _text_`.

## What NOT to do

- Do **NOT** modify any Go code in this prompt — that happens in prompt 76
- Do **NOT** remove or alter the existing `analysis_json` field — it's kept for backward compatibility
- Do **NOT** add fields for `risk_or_limitations` or `social_proof_signals` — these are less useful for search and would add noise

## Verification

After running all commands above, confirm:
1. All 5 `analysis_*` fields are listed in the schema with correct types
2. All 5 copyField rules to `_text_` are registered
3. No Solr errors in the responses
