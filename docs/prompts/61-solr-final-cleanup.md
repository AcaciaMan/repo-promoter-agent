# Prompt: Solr Migration — Final Cleanup of Stale SQLite References

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. Phase 2 (prompts 57–60) replaced the SQLite store with a Solr-backed implementation. This is **Phase 3, prompt 1 of 2** — cleaning up any remaining SQLite references across the codebase and docs.

**Current state**:
- `internal/store/store.go` — Solr-backed ✓
- `cmd/server/main.go` — uses `SOLR_URL` / `SOLR_CORE` ✓
- `go.mod` — no `modernc.org/sqlite` dependency ✓
- `README.md` — mostly updated, but **Tech Stack section still mentions SQLite and FTS5**

## Your task

Find and fix **every** remaining SQLite/FTS5 reference in the project. The project is now 100% Solr-backed — no SQLite remnants should exist.

## Requirements

### 1. Scan all Go source files for SQLite references

Run these searches and verify **zero** matches:

```powershell
cd C:\work\GitHub\repo-promoter-agent
Select-String -Path "internal\**\*.go","cmd\**\*.go" -Pattern "sqlite|sql\.DB|sql\.Open|sql\.Rows|database/sql|modernc|FTS5|fts5|DB_PATH|dbPath|promotions\.db" -Recurse
```

If any matches are found, fix them.

### 2. Scan go.mod and go.sum

```powershell
Select-String -Path "go.mod","go.sum" -Pattern "sqlite|modernc"
```

If `go.sum` has stale references, run `go mod tidy` to clean them up.

### 3. Fix README.md Tech Stack section

The Tech Stack section near the bottom of `README.md` currently says:

```markdown
## Tech Stack

- **Go** — HTTP server, agent client, GitHub client
- **SQLite (modernc.org/sqlite)** — Pure-Go SQLite driver, no CGO required
- **FTS5** — Full-text search virtual table with auto-sync triggers
- **go:embed** — Static asset embedding
- **godotenv** — `.env` file loading
```

Replace it with:

```markdown
## Tech Stack

- **Go** — HTTP server, agent client, GitHub client
- **Apache Solr 10** — Enterprise-grade full-text search and data store
- **go:embed** — Static asset embedding
- **godotenv** — `.env` file loading
```

### 4. Scan all markdown docs for stale SQLite references

```powershell
Select-String -Path "docs\**\*.md" -Pattern "SQLite|sqlite|FTS5|fts5|promotions\.db|DB_PATH" -Recurse
```

For any matches found:
- **Historical prompt files** (e.g., `docs/prompts/10-sqlite-storage.md`) — **leave as-is**. They document the original design.
- **Active contract/architecture docs** (e.g., `docs/contracts/api-env-contract.md`, `docs/Hackathon.md`) — update if they describe the current system inaccurately.

### 5. Check `docs/contracts/api-env-contract.md`

Read this file. If it lists `DB_PATH` as an environment variable, remove it and ensure `SOLR_URL` and `SOLR_CORE` are listed. If it doesn't mention DB_PATH, no changes needed.

### 6. Check `.env` file

```powershell
if (Test-Path .env) { Select-String -Path ".env" -Pattern "DB_PATH" }
```

If `DB_PATH` is present, remove that line. Ensure `SOLR_URL` and `SOLR_CORE` are present.

### 7. Delete leftover database files

```powershell
Get-ChildItem -Path "C:\work\GitHub\repo-promoter-agent" -Filter "*.db" -File -Recurse
```

Delete any `.db` files found.

### 8. Check `.gitignore`

If `.gitignore` has entries for `*.db` or `promotions.db`, they can optionally be removed (they're harmless to keep, but cleaning them shows the migration is complete).

### 9. Final build verification

```powershell
go build ./...
```

Must compile cleanly with zero errors.

## Verification checklist

- [ ] Zero SQLite/FTS5/DB_PATH references in any `.go` file
- [ ] `go.mod` and `go.sum` have no `modernc`/`sqlite` references
- [ ] README.md Tech Stack section references Solr, not SQLite
- [ ] `docs/contracts/api-env-contract.md` lists `SOLR_URL`/`SOLR_CORE`, not `DB_PATH`
- [ ] `.env` file has no `DB_PATH`
- [ ] No `*.db` files in the workspace
- [ ] `go build ./...` compiles cleanly

## Notes

- Historical prompt files in `docs/prompts/` (like `10-sqlite-storage.md`) should NOT be modified — they document past decisions.
- The `docs/Hackathon.md` file describes the original plan with SQLite — it's context documentation, not a live spec. Leave it as-is unless it's confusing.
