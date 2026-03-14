# Shakespeare Database

[![Build](https://github.com/scottdkey/shakespeare_db/actions/workflows/build.yml/badge.svg)](https://github.com/scottdkey/shakespeare_db/actions/workflows/build.yml)
[![Latest Release](https://img.shields.io/github/v/release/scottdkey/shakespeare_db)](https://github.com/scottdkey/shakespeare_db/releases/latest)
[![Download](https://img.shields.io/github/downloads/scottdkey/shakespeare_db/total)](https://github.com/scottdkey/shakespeare_db/releases/latest)

A comprehensive SQLite database of Shakespeare's complete works with a web UI, built from multiple open-source and public domain sources.

## What's in it

| Content | Source | License |
|---------|--------|---------|
| 43 works, 1,265+ characters, ~36K text lines | [Open Source Shakespeare](https://opensourceshakespeare.org) / Moby | Public Domain |
| Modern-spelling verse-level text for all 37 plays | [Standard Ebooks](https://standardebooks.org) | CC0 |
| 154 sonnets + 5 poems (Venus & Adonis, etc.) | Standard Ebooks | CC0 |
| 1,800+ lexicon entries with 20,000+ citations | [Perseus Digital Library](http://www.perseus.tufts.edu) — Schmidt Lexicon | CC BY-SA 3.0 |
| Full-text search (FTS5) across text + lexicon | Generated | — |
| Folger Shakespeare Library reference URLs | [Folger](https://www.folger.edu) | — |

## Monorepo Structure

```
shakespeare_db/
├── projects/
│   ├── sources/       Original texts — READ-ONLY (OSS, SE, Perseus)
│   ├── data/          Shared reference JSON (work mappings, attributions)
│   ├── db-builder/    Go pipeline → produces SQLite database
│   └── web/           SvelteKit PWA → deployed to Cloudflare (future)
├── Makefile           Root: delegates to per-project Makefiles
└── .github/workflows/ CI/CD
```

Each project has its own Makefile (or package.json) with its own actions. The root Makefile delegates via namespace:

```bash
make <project> <action>
```

## Quick Start

### Download pre-built

Grab `shakespeare.db` from the [latest release](https://github.com/scottdkey/shakespeare_db/releases/latest).

### Build from source

Requires **Go 1.22+**.

```bash
git clone https://github.com/scottdkey/shakespeare_db.git
cd shakespeare_db

# Run all tests
make db-builder test

# Build the database (~55 MB)
make db-builder run
# → build/shakespeare.db

# Skip SE downloads (use cached data)
make db-builder run-cached
```

### All Make Commands

```bash
# db-builder (Go pipeline)
make db-builder build        # Compile binary
make db-builder test         # Run all tests
make db-builder run          # Full pipeline
make db-builder run-cached   # Skip downloads
make db-builder lint         # go vet
make db-builder cover        # Test coverage report
make db-builder clean        # Remove artifacts

# sources (read-only originals)
make sources verify          # Check all files exist
make sources list            # List all source files
make sources stats           # File counts per source

# data (reference mappings)
make data validate           # Validate all JSON files
make data list               # List all data files

# web (SvelteKit — future)
make web dev                 # Dev server
make web build               # Production build
make web test                # Run tests

# Cross-project
make test-all                # Tests across all projects
make clean-all               # Clean all projects
make help                    # Show all commands
```

## Testing

Every parser is a pure function with dedicated tests. The full suite includes unit tests, integration tests, and end-to-end verification:

```bash
# All tests
make db-builder test

# With coverage
make db-builder cover
```

## Schema

See [projects/db-builder/SCHEMA.md](projects/db-builder/SCHEMA.md) for the full database schema.

## Sources & Licensing

See [projects/sources/SOURCES.md](projects/sources/SOURCES.md) for detailed source attribution and licensing.

## CI/CD

- **Push to main / PR** → tests + build + upload artifact
- **Manual trigger** → tests + build + optional GitHub Release (must check "Create a release")

Releases are never automatic — they require a manual workflow dispatch with the release flag enabled.
