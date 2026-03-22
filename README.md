# Bardbase

[![Build](https://github.com/scottdkey/bardbase/actions/workflows/build.yml/badge.svg)](https://github.com/scottdkey/bardbase/actions/workflows/build.yml)
[![API](https://github.com/scottdkey/bardbase/actions/workflows/api-deploy.yml/badge.svg)](https://github.com/scottdkey/bardbase/actions/workflows/api-deploy.yml)
[![Frontend](https://github.com/scottdkey/bardbase/actions/workflows/frontend-build.yml/badge.svg)](https://github.com/scottdkey/bardbase/actions/workflows/frontend-build.yml)

A multi-edition Shakespeare reader with lexicon, cross-edition alignment, and reference works — built on SQLite, Go, and SvelteKit.

**Live:** [bardbase.pages.dev](https://bardbase.pages.dev)

## Architecture

```
┌─────────────────────┐     ┌──────────────────────┐
│  Cloudflare Pages   │────▶│  Go API (Railway)     │
│  SvelteKit SSR      │     │  SQLite + bardbase.db │
└─────────────────────┘     └──────────────────────┘
```

- **Go HTTP API** — serves `bardbase.db` via native SQLite, REST endpoints, API key auth
- **SvelteKit on Cloudflare** — SSR at the edge, server routes call the Go API
- **Docker Compose** — local dev with hot reload (air + vite)

## Quick Start

### Local Development

```bash
# Build the database (requires Go)
make capell run

# Start dev stack (requires podman/docker)
podman compose up --build

# Or run services individually
make api run    # Go API on :8080
make web run    # SvelteKit on :5173
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

## API Endpoints

All endpoints accept work slugs or numeric IDs (e.g. `/api/works/hamlet/toc` or `/api/works/8/toc`).

| Endpoint | Description |
|----------|-------------|
| `GET /health` | Health check (no auth) |
| `GET /api/works` | List plays and poetry |
| `GET /api/works/{id}/toc` | Act/scene structure |
| `GET /api/works/{id}/editions` | Available editions |
| `GET /api/text/scene/{work}/{act}/{scene}` | Multi-edition aligned text |
| `GET /api/text/scene/{work}/{act}/{scene}/references` | Lexicon + reference citations for scene |
| `GET /api/search?q=term` | FTS5 lexicon search |
| `GET /api/lexicon/entry/{id}` | Full lexicon entry detail |
| `GET /api/reference/entry/{id}` | Reference work entry (Onions, Abbott, etc.) |
| `GET /api/reference/search?q=&source=&work_id=` | Search reference entries |
| `GET /api/reference/sources` | List reference sources with counts |
| `GET /api/resolve/{slug}` | Resolve work slug to ID |
| `GET /api/corrections?state=all` | GitHub issues labeled "correction" |
| `GET /api/attributions` | Footer attribution data |
| `GET /api/stats` | Database statistics |

## Structure

```
bardbase/
├── projects/
│   ├── sources/        source texts (committed, read-only)
│   ├── data/           reference JSON (work mappings, attributions)
│   ├── capell/         Go build pipeline + HTTP API
│   │   ├── cmd/api/    API server entry point
│   │   ├── cmd/build/  Database build pipeline
│   │   └── internal/   API handlers, DB queries
│   └── web/            SvelteKit frontend (Cloudflare Pages)
│       ├── src/routes/  Page routes + API proxy routes
│       └── src/lib/     Components, stores, utilities
├── docker-compose.yml  Dev stack (Go API + SvelteKit)
├── Makefile            Project-level make targets
└── .github/workflows/  CI/CD pipelines
```

## Deployment

| Service | Platform | Trigger |
|---------|----------|---------|
| Database build | GitHub Actions | Manual dispatch |
| Go API image | GitHub Container Registry | Push to main (capell changes) |
| Go API deploy | Railway | Auto-deploys from GHCR image |
| Frontend deploy | Cloudflare Pages | Push to main (web changes) |

### Required Secrets

| Secret | Where | Purpose |
|--------|-------|---------|
| `CLOUDFLARE_API_TOKEN` | GitHub Actions | Cloudflare Pages deploy |
| `CLOUDFLARE_ACCOUNT_ID` | GitHub Actions | Cloudflare account |
| `API_KEY` | Railway + Cloudflare | Shared auth between frontend and API |

## Documentation

| Doc | What it covers |
|-----|---------------|
| [docs/pipeline-overview.md](docs/pipeline-overview.md) | Build pipeline phases |
| [docs/line-alignment.md](docs/line-alignment.md) | Needleman-Wunsch cross-edition alignment |
| [docs/citation-resolution.md](docs/citation-resolution.md) | 5-strategy citation matching cascade |
| [docs/fts-search.md](docs/fts-search.md) | FTS5 setup and query examples |

## Sources & Attribution

See [ATTRIBUTION.md](ATTRIBUTION.md) and [projects/sources/SOURCES.md](projects/sources/SOURCES.md).
