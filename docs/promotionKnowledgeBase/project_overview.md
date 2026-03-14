# Project Overview

## One‑sentence pitch
Repo Promoter Agent turns public GitHub repositories into structured, reusable promotional content bundles, powered by DigitalOcean Gradient AI.

## Problem

- Developers struggle to effectively promote their GitHub repositories across channels like Twitter/X and LinkedIn.
- Writing clear, compelling descriptions and social posts is time‑consuming and requires marketing skills many developers don’t have or don’t enjoy using.
- Even when good copy exists, it is scattered across READMEs, issues, personal notes, and past posts, making it hard to search, reuse, and keep consistent.

Existing solutions (manual posting, generic social media tools, generic AI chat prompts) are not specialized for code repositories and do not leverage repository structure, topics, or metrics to generate high‑quality, repo‑aware promotional content.

## Solution

Repo Promoter Agent is a web app + AI agent that converts a public GitHub repository into a structured “promotion bundle” of content that can be searched, browsed, and reused later.

Key features:

- GitHub‑aware content generation  
  - User submits a public GitHub repo URL.  
  - The Go backend fetches name, description, README content, and optional metrics via the GitHub API.  
  - An AI agent on DigitalOcean Gradient transforms this structured input into tailored promotional content.

- Multi‑channel promotional copy  
  - The agent outputs a consistent set of assets: headline, summary, key benefits, tags, tweets, a LinkedIn post, and a call‑to‑action.  
  - Style can be lightly adapted based on the requested target channel (e.g., `twitter`, `linkedin`, `general`).

- Searchable content library  
  - Generated content is stored in Apache Solr with enterprise-grade full‑text search.  
  - Users can later search promotions by text, tags, or channel and quickly find reusable pieces of copy.

- Simple, focused UI  
  - A minimal HTML/CSS/JavaScript interface with a “Generate” page for new content and a “Search” page for browsing and reuse.

What makes it unique:

- Purpose‑built for GitHub repositories, not generic social media posting.
- Uses repository structure (README, topics, metrics) to generate context‑aware marketing copy.
- Treats promotional content as a searchable knowledge base rather than one‑off posts that disappear in timelines.

## Target users

- Open‑source maintainers who want to attract more users and contributors to their projects.
- Individual developers building portfolios and side projects who lack time or marketing experience.
- Small teams maintaining multiple repositories and wanting a central place to manage promotional copy for launches and updates.

Typical scenarios:

- A maintainer preparing launch posts for a new release of an open‑source tool.
- A solo developer polishing their GitHub projects to share on LinkedIn when job hunting.
- A small team coordinating consistent messaging across several related repositories.

## Current status

- Core design:
  - AI agent spec and JSON schema are defined (GitHub Repo Promotion Agent on Gradient AI).
  - High‑level architecture for frontend, Go backend, and Apache Solr storage is laid out.
- First version goals:
  - Single trusted user scenario (no complex auth).
  - Support for public repositories only.
  - One AI agent is sufficient; no multi‑agent orchestration.

Post‑hackathon, Repo Promoter Agent can be extended with multi‑user support (e.g., via a GitHub App), richer analytics, and more advanced content strategies.

## Links

- Project repo: https://github.com/AcaciaMan/repo-promoter-agent
- Hackathon: DigitalOcean Gradient AI Hackathon.
- Stack:
  - AI: DigitalOcean Gradient AI (LLM‑based agent).
  - Backend: Go service integrating GitHub API, Gradient AI, and Apache Solr.
  - Frontend: Simple HTML/CSS/JavaScript web UI.
