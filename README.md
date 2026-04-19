# Bardbase

[![Frontend](https://github.com/scottdkey/bardbase/actions/workflows/frontend-build.yml/badge.svg)](https://github.com/scottdkey/bardbase/actions/workflows/frontend-build.yml)
![Downloads](https://img.shields.io/github/downloads/scottdkey/bardbase/total)

A multi-edition Shakespeare reader with lexicon, cross-edition alignment, and reference works — built on SQLite, Go, and SvelteKit.

**Live:** [bardbase.scottkey.dev](https://bardbase.scottkey.dev)
**Preview:** [bardbase.pages.dev](https://bardbase.pages.dev)

## Warning --- alpha ----
This project should be considered in alpha. There are many issues left to resolve, if you end up using the app and find an issue, please report issues to the github repository. Within the web app there is an issue reporter in many places(but not all) in the application. Please keep in mind that this project is something that is worked on for free and in spare time. While I want to provide the best possible experience, some things are outside of my control.



## Architecture

```
Database build (local):
┌──────────────────────┐     ┌─────────────────────┐
│  capell Go pipeline  │────▶│  GitHub Release      │
│  (make capell run)   │     │  (bardbase.db)       │
└──────────────────────┘     └─────────────────────┘

CI deploy (on push to main):
┌─────────────────────┐     ┌──────────────────────┐     ┌─────────────────────┐
│  Download           │────▶│  SvelteKit build      │────▶│  Cloudflare Pages   │
│  bardbase.db        │     │  (prerender + bundle) │     │  (static + Workers) │
│  (GitHub Release)   │     │  reads SQLite direct  │     │                     │
└─────────────────────┘     └──────────────────────┘     └─────────────────────┘
```

- **capell** — Go pipeline that reads committed source texts and builds `bardbase.db`; published to GitHub Releases via `make capell release`
- **SvelteKit** — reads `bardbase.db` directly via `node:sqlite` during build; prerenders all pages; dynamic routes (search, etc.) run as Cloudflare Workers edge functions
- **Cloudflare Pages** — serves prerendered HTML + JS; no separate API server at runtime

## Quick Start

### Setup (first clone)

```bash
make setup      # configures local git hooks
```

### Local Development

```bash
# Build the database (requires Go)
make capell run          # full build → build/bardbase.db

# Start the SvelteKit dev server
make web run             # dev server on :5173, reads build/bardbase.db by default
```

The dev server reads `DB_PATH` from the environment (defaults to `../../build/bardbase.db` relative to `projects/web/`). Set it explicitly if your db is elsewhere:

```bash
DB_PATH=/path/to/bardbase.db make web run
```

### Build Database from Source

```bash
make capell run          # full build → build/bardbase.db
make capell run-fresh    # re-download sources first
make capell test         # run tests
```

## What's in it

| Content | Source | Count |
|---------|--------|-------|
| Shakespeare plays | Open Source Shakespeare / Moby | 37 plays |
| Modern-spelling text | Standard Ebooks | 37 plays + poetry |
| Globe edition (1864) | Perseus Digital Library | dual line numbering |
| First Folio (1623) | EEBO-TCP | original spelling |
| Folger Shakespeare | Folger Library | modern edition |
| Schmidt Lexicon | Perseus / Schmidt | 20K entries, ~200K citations |
| Onions Glossary | OCR import | 12.9K entries |
| Abbott Grammar | OCR import | 670 entries |
| Bartlett Concordance | OCR import | 84K entries |
| Cross-edition alignment | Generated (Needleman-Wunsch) | line mappings |
| Full-text search | Generated (FTS5) | lexicon + text |

## Features

- **Ebook reader** — swipe/arrow navigation, reading position memory, TOC panel
- **Multi-edition viewer** — up to 5 editions side-by-side with cross-edition alignment
- **Word references** — hover/tap words to see lexicon entries from Schmidt, Onions, Abbott, Bartlett
- **Reference browser** — search and filter all reference works by source and play
- **Correction system** — flag entries/citations, creates GitHub issues
- **PWA** — offline capable with workbox caching
- **Slug URLs** — `/text/hamlet/1/4` instead of `/text/8/1/4`
- **Light/dark mode** — warm parchment light theme, deep dark theme

## API Routes

SvelteKit server routes (run as Cloudflare Workers in production, Node.js in dev). All work endpoints accept slugs or numeric IDs (e.g. `/api/works/hamlet/toc` or `/api/works/8/toc`).

| Endpoint | Description |
|----------|-------------|
| `GET /api/works` | List plays and poetry |
| `GET /api/works/{id}/toc` | Act/scene structure |
| `GET /api/text/scene/{work}/{act}/{scene}` | Multi-edition aligned text |
| `GET /api/search?q=term` | FTS5 lexicon search |
| `GET /api/lexicon/entry/{id}` | Full lexicon entry detail |
| `GET /api/reference/entry/{id}` | Reference work entry (Onions, Abbott, etc.) |
| `GET /api/reference/search?q=&source=&work_id=` | Search reference entries |
| `GET /api/reference/sources` | List reference sources with counts |
| `GET /api/corrections?state=all` | GitHub issues labeled "correction" |
| `GET /api/attributions` | Footer attribution data |
| `GET /api/version` | Build version info |

## Structure

```
bardbase/
├── projects/
│   ├── sources/        source texts (committed, read-only)
│   ├── data/           reference JSON (work mappings, attributions)
│   ├── capell/         Go build pipeline
│   │   ├── cmd/build/  Database build pipeline (19-phase)
│   │   └── internal/   Parsers, importers, aligner, citation resolver
│   └── web/            SvelteKit frontend (Cloudflare Pages)
│       ├── src/routes/  Page routes + SvelteKit API routes
│       └── src/lib/     Components, stores, server DB access
├── build/              Generated database (gitignored)
├── Makefile            Project-level make targets
└── .github/workflows/  CI/CD pipelines
```

## Deployment

| Service | Platform | Trigger |
|---------|----------|---------|
| Database build | Local | `make capell release` — builds `bardbase.db` and publishes to GitHub Releases |
| Frontend deploy | Cloudflare Pages | Push to main (web changes) — downloads latest DB release, prerenders, deploys |

### Required Secrets

| Secret | Where | Purpose |
|--------|-------|---------|
| `CLOUDFLARE_API_TOKEN` | GitHub Actions | Cloudflare Pages deploy |
| `CLOUDFLARE_ACCOUNT_ID` | GitHub Actions | Cloudflare account |

## Documentation

| Doc | What it covers |
|-----|---------------|
| [docs/pipeline-overview.md](docs/pipeline-overview.md) | Build pipeline phases |
| [docs/line-alignment.md](docs/line-alignment.md) | Needleman-Wunsch cross-edition alignment |
| [docs/citation-resolution.md](docs/citation-resolution.md) | 5-strategy citation matching cascade |
| [docs/fts-search.md](docs/fts-search.md) | FTS5 setup and query examples |

## Sources & Attribution

See [ATTRIBUTION.md](ATTRIBUTION.md) and [projects/sources/SOURCES.md](projects/sources/SOURCES.md).
