# Prompt: Solr Prerequisites — Java 21 and Solr 10 Installation

## Context

I'm building a **repo-promoter-agent** for the DigitalOcean Gradient AI Hackathon. All previous phases (prompts 01–53) are complete. I'm now replacing SQLite with **Apache Solr** as the sole data store to get enterprise-grade full-text search, faceting, and scalable querying. This is **Phase 1 of the Solr migration** — local infrastructure setup.

The project currently uses SQLite via `internal/store/store.go` with FTS5 for search. Solr will fully replace it — no rollback path, no dual-backend.

This prompt (54) ensures Java 21 and Solr 10 are installed and working locally. Subsequent prompts will:
- 55: Create the `promotions` core and define the Solr schema
- 56: POST a test document, verify query, and validate the setup end-to-end

## Your task

Help me verify (or install) the prerequisites for running Apache Solr 10 locally on Windows.

## Requirements

### 1. Verify Java 21

Run `java -version` in the terminal.

- If Java 21+ is installed, confirm it and move on.
- If not, guide me to download and install **Eclipse Temurin JDK 21** (or equivalent) from https://adoptium.net/temurin/releases/ — pick the Windows x64 `.msi` installer.
- Verify `JAVA_HOME` is set correctly by running `echo $env:JAVA_HOME` in PowerShell.
- If `JAVA_HOME` is not set, set it:
  ```powershell
  [System.Environment]::SetEnvironmentVariable("JAVA_HOME", "C:\Program Files\Eclipse Adoptium\jdk-21...", "User")
  ```

### 2. Download Solr 10

- Check if Solr is already available by running `solr.cmd version` or checking if `c:\solr` (or similar) exists.
- If not installed, download **Apache Solr 10** from https://solr.apache.org/downloads.html — the binary `.zip` release for Windows.
- Extract to a known path, e.g., `c:\solr\solr-10.x.x\`.
- Verify the `bin\solr.cmd` script exists in the extracted directory.

### 3. Start Solr

Start Solr in standalone/foreground mode:

```powershell
cd c:\solr\solr-10.x.x
bin\solr.cmd start
```

If Solr starts successfully, it will be available at `http://localhost:8983/solr/`.

### 4. Verify Solr is running

- Open `http://localhost:8983/solr/` in a browser or run:
  ```powershell
  Invoke-WebRequest -Uri "http://localhost:8983/solr/admin/info/system" -UseBasicParsing | Select-Object StatusCode
  ```
- Confirm HTTP 200 response.

## Verification checklist

- [ ] `java -version` shows Java 21+
- [ ] `JAVA_HOME` env var points to the JDK 21 installation
- [ ] Solr 10 is extracted to a known directory
- [ ] `bin\solr.cmd start` runs without errors
- [ ] `http://localhost:8983/solr/` responds with HTTP 200

## Notes

- Solr will be used as the **sole data store** — both for persistence and full-text search.
- No Docker — Solr runs directly on the local machine.
- The default port 8983 is fine for local development.
- If Solr is already running from a previous session, `bin\solr.cmd status` will show its state. Use `bin\solr.cmd restart` if needed.
