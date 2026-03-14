# Prompt: Populate Analysis Fields on Save & Add to Search Boosting

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. The app stores AI-generated promotional content in **Apache Solr 10** and exposes full-text search via `GET /api/search?q=...`.

This is **Phase 4, prompt 2 of 5**. Prompt 75 added 5 new indexed Solr fields for analysis data:
- `analysis_value_proposition` (text_general, single-valued)
- `analysis_ideal_audience` (text_general, multiValued)  
- `analysis_key_features` (text_general, multiValued)
- `analysis_differentiators` (text_general, multiValued)
- `analysis_positioning` (text_general, multiValued)

These fields exist in the Solr schema but are never populated by the Go app. This prompt updates `Store.Save()` to extract analysis data from `AnalysisJSON` and populate these fields, and adds them to the `qf` boost list in `Store.Search()`.

## Current state

### `Store.Save()` in `internal/store/store.go` — analysis section

```go
if p.AnalysisJSON != nil {
    doc["analysis_json"] = string(p.AnalysisJSON)
}
```

### `Store.Search()` — qf parameter

```go
"qf": {"repo_name^3 headline^4 summary^2 key_benefits^1.5 tags^3 twitter_posts^1 linkedin_post^1 call_to_action^1 target_audience^1.5 readme^0.5"},
```

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

### Current imports in `store.go`

```go
import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "strings"
    "time"
)
```

## Your task

Modify `internal/store/store.go` to:
1. Extract analysis data from `AnalysisJSON` during Save and populate the new indexed fields
2. Add analysis fields to the edismax `qf` boost list in Search
3. Add analysis fields to highlighting

## Requirements

### 1. Update `Save()` to populate analysis indexed fields

In the `Save` method, after the existing `if p.AnalysisJSON != nil` block that sets `doc["analysis_json"]`, add code to deserialize the JSON and populate the new indexed fields.

Replace this block:

```go
if p.AnalysisJSON != nil {
    doc["analysis_json"] = string(p.AnalysisJSON)
}
```

With:

```go
if p.AnalysisJSON != nil {
    doc["analysis_json"] = string(p.AnalysisJSON)

    // Deserialize analysis JSON to populate indexed analysis fields.
    var analysis struct {
        PrimaryValueProposition     string   `json:"primary_value_proposition"`
        IdealAudience               []string `json:"ideal_audience"`
        KeyFeatures                 []string `json:"key_features"`
        Differentiators             []string `json:"differentiators"`
        RecommendedPositioningAngle []string `json:"recommended_positioning_angle"`
    }
    if err := json.Unmarshal(p.AnalysisJSON, &analysis); err == nil {
        if analysis.PrimaryValueProposition != "" {
            doc["analysis_value_proposition"] = analysis.PrimaryValueProposition
        }
        if len(analysis.IdealAudience) > 0 {
            doc["analysis_ideal_audience"] = analysis.IdealAudience
        }
        if len(analysis.KeyFeatures) > 0 {
            doc["analysis_key_features"] = analysis.KeyFeatures
        }
        if len(analysis.Differentiators) > 0 {
            doc["analysis_differentiators"] = analysis.Differentiators
        }
        if len(analysis.RecommendedPositioningAngle) > 0 {
            doc["analysis_positioning"] = analysis.RecommendedPositioningAngle
        }
    }
}
```

**Key points**:
- Use a local anonymous struct — don't import the `agent` package (avoids circular dependency)
- Only set fields when they have actual values (avoid empty strings/slices in Solr)
- If unmarshal fails (corrupted JSON), silently skip — the raw `analysis_json` is still stored
- No `log` import needed — just skip on error

### 2. Add analysis fields to `qf` in `Search()`

Update the `qf` parameter to include the new analysis fields with appropriate boost weights:

Replace:

```go
"qf": {"repo_name^3 headline^4 summary^2 key_benefits^1.5 tags^3 twitter_posts^1 linkedin_post^1 call_to_action^1 target_audience^1.5 readme^0.5"},
```

With:

```go
"qf": {"repo_name^3 headline^4 summary^2 key_benefits^1.5 tags^3 twitter_posts^1 linkedin_post^1 call_to_action^1 target_audience^1.5 readme^0.5 analysis_value_proposition^2 analysis_key_features^1.5 analysis_differentiators^1.5 analysis_ideal_audience^1 analysis_positioning^1"},
```

**Boost rationale**:
- `analysis_value_proposition^2` — high value, concise summary of the repo's purpose
- `analysis_key_features^1.5` — useful feature descriptions the human author may not have written
- `analysis_differentiators^1.5` — unique selling points that distinguish this repo
- `analysis_ideal_audience^1` — audience terms, baseline weight (already covered by `target_audience`)
- `analysis_positioning^1` — marketing angles, baseline weight

### 3. Add analysis fields to highlighting

Update the `hl.fl` parameter to include analysis fields so highlighted snippets are returned for analysis matches too.

Replace:

```go
"hl.fl": {"headline,summary,key_benefits,linkedin_post,call_to_action,target_audience"},
```

With:

```go
"hl.fl": {"headline,summary,key_benefits,linkedin_post,call_to_action,target_audience,analysis_value_proposition,analysis_key_features,analysis_differentiators"},
```

(Skip `analysis_ideal_audience` and `analysis_positioning` from highlighting — they're less useful as display snippets.)

## What NOT to do

- Do **NOT** import the `agent` package in `store.go` — use a local anonymous struct
- Do **NOT** modify any other files — handler and frontend changes are not needed for this prompt
- Do **NOT** remove the `analysis_json` field from the document — keep it for backward compatibility
- Do **NOT** add `log` statements for the unmarshal error — silent skip is intentional

## Verification

```powershell
# Build to confirm no compile errors
go build ./...
```

Then manually verify the logic:
1. The `Save` method will now populate `analysis_value_proposition`, `analysis_ideal_audience`, `analysis_key_features`, `analysis_differentiators`, `analysis_positioning` when a promotion has `AnalysisJSON`
2. The `Search` method will now match queries against analysis fields with appropriate boost weights
3. Highlighting will show `<mark>` snippets for matches in analysis fields
