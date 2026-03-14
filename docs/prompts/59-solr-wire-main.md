# Prompt: Wire Solr Store into main.go and Handlers

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. I'm replacing SQLite with **Apache Solr 10** as the sole data store. This is **Phase 2, prompt 3 of 4**.

**Prerequisites**: Prompts 57–58 are complete — `internal/store/store.go` is now a Solr-backed implementation with `New(solrURL, core string)` signature, and integration tests pass.

The following files still reference the **old** `store.New(dbPath)` single-argument constructor and need updating:

1. `cmd/server/main.go` — calls `store.New(dbPath)` and reads `DB_PATH` env var
2. `internal/handler/generate.go` — holds `*store.Store` (this type name hasn't changed, so may compile as-is, but double-check)
3. `internal/handler/search.go` — same

## Your task

Update `cmd/server/main.go` to instantiate the Solr store, and verify handlers compile against the new store.

## Requirements

### 1. Update `cmd/server/main.go`

#### a) Replace DB_PATH env vars with Solr env vars

Remove:
```go
dbPath := os.Getenv("DB_PATH")
if dbPath == "" {
    dbPath = "promotions.db"
}
```

Add:
```go
solrURL := os.Getenv("SOLR_URL")
if solrURL == "" {
    solrURL = "http://localhost:8983"
}
solrCore := os.Getenv("SOLR_CORE")
if solrCore == "" {
    solrCore = "promotions"
}
```

#### b) Replace store initialization

Remove:
```go
st, err := store.New(dbPath)
if err != nil {
    log.Fatalf("Failed to open database: %v", err)
}
defer st.Close()
```

Replace with:
```go
st, err := store.New(solrURL, solrCore)
if err != nil {
    log.Fatalf("Failed to connect to Solr: %v", err)
}
defer st.Close()
log.Printf("Connected to Solr at %s (core: %s)", solrURL, solrCore)
```

#### c) Everything else stays the same

The handler constructors `handler.NewGenerateHandler(agentClient, githubClient, st, analysisClient)` and `handler.NewSearchHandler(st)` take `*store.Store` — since we kept the struct name `Store`, these should compile without changes.

### 2. Verify handlers compile

The handlers use `*store.Store` — since the struct was renamed in-place (still called `Store`), and `Save`, `Search`, `List` method signatures are unchanged, the handlers should compile as-is.

Read through `internal/handler/generate.go` and `internal/handler/search.go` to verify:
- `h.store.Save(r.Context(), &promo)` — matches `Save(ctx context.Context, p *Promotion) error` ✓
- `h.store.Search(r.Context(), q, limit)` — matches `Search(ctx context.Context, query string, limit int) ([]Promotion, error)` ✓
- `h.store.List(r.Context(), limit)` — matches `List(ctx context.Context, limit int) ([]Promotion, error)` ✓

If there are any compile errors, fix them.

### 3. Remove SQLite dependency from go.mod

Run:
```powershell
go mod tidy
```

This should remove `modernc.org/sqlite` and all its transitive dependencies (`dustin/go-humanize`, `mattn/go-isatty`, `ncruces/go-strftime`, `remyoudompheng/bigfft`, `modernc.org/libc`, `modernc.org/mathutil`, `modernc.org/memory`).

Verify with:
```powershell
Select-String "sqlite" go.mod
Select-String "modernc" go.mod
```

Both should return no results.

### 4. Update `.env` file (if present)

If a `.env` file exists, add:
```
SOLR_URL=http://localhost:8983
SOLR_CORE=promotions
```

Remove any `DB_PATH` entry.

## Verification

```powershell
# Must compile cleanly
go build ./...

# No SQLite references in go.mod
Select-String "sqlite|modernc" go.mod

# Start the server (Solr must be running)
go run ./cmd/server/main.go
# Expected log: "Connected to Solr at http://localhost:8983 (core: promotions)"
# Expected: server starts on :8080 without errors
```

## Notes

- The `joho/godotenv` dependency stays — it's used for `.env` loading.
- Handler files should not need any changes since the store's type name and method signatures are preserved.
- If `go mod tidy` removes `golang.org/x/exp` or `golang.org/x/sys`, that's fine — they were transitive deps of the sqlite driver.
- The server won't be able to generate content yet without proper `AGENT_ENDPOINT` and `AGENT_ACCESS_KEY` — that's normal.
