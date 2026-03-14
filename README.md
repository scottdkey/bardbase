# Shakespeare Database

[![Build](https://github.com/scottdkey/shakespeare_db/actions/workflows/build.yml/badge.svg)](https://github.com/scottdkey/shakespeare_db/actions/workflows/build.yml)
[![Latest Release](https://img.shields.io/github/v/release/scottdkey/shakespeare_db)](https://github.com/scottdkey/shakespeare_db/releases/latest)
[![Download](https://img.shields.io/github/downloads/scottdkey/shakespeare_db/total)](https://github.com/scottdkey/shakespeare_db/releases/latest)

A comprehensive SQLite database of Shakespeare's complete works, built from multiple open-source and public domain sources.

## What's in it

| Content | Source | License |
|---------|--------|---------|
| 43 works, 1,265+ characters, ~36K text lines | [Open Source Shakespeare](https://opensourceshakespeare.org) / Moby | Public Domain |
| Modern-spelling verse-level text for all 37 plays | [Standard Ebooks](https://standardebooks.org) | CC0 |
| 154 sonnets + 5 poems (Venus & Adonis, etc.) | Standard Ebooks | CC0 |
| 1,800+ lexicon entries with 20,000+ citations | [Perseus Digital Library](http://www.perseus.tufts.edu) — Schmidt Lexicon | CC BY-SA 3.0 |
| Full-text search (FTS5) across text + lexicon | Generated | — |
| Folger Shakespeare Library reference URLs | [Folger](https://www.folger.edu) | — |

## Quick start

### Download pre-built

Grab `shakespeare.db` from the [latest release](https://github.com/scottdkey/shakespeare_db/releases/latest).

### Build from source

Requires **Go 1.22+**.

```bash
git clone https://github.com/scottdkey/shakespeare_db.git
cd shakespeare_db

# Install dependencies
make setup

# Run all tests
make test

# Build the database (~55 MB)
make run
# → build/shakespeare.db
```

### Build options

```bash
# Skip SE downloads (use cached data only)
make run-cached

# Run a single build step
make run-step-oss       # OSS/Moby import only
make run-step-lexicon   # Schmidt lexicon only
make run-step-se        # Standard Ebooks plays only
make run-step-poetry    # Poetry + sonnets + Folger URLs
make run-step-fts       # Rebuild FTS indexes

# Or use the binary directly
go run ./cmd/build -output build -skip-download -step oss
```

## Project structure

```
cmd/build/              CLI entry point
internal/
  constants/            Reference data (Schmidt maps, schema DDL)
  parser/               Pure parsing functions (MySQL, XML, XHTML)
  db/                   SQLite connection & schema management
  fetch/                HTTP client with retries
  importer/             Build steps (OSS, lexicon, SE plays, poetry, FTS)
sources/
  oss/                  OSS MySQL dump (Public Domain)
  lexicon/entries/      Schmidt XML files (CC BY-SA 3.0)
```

## Testing

Every parser is a pure function with dedicated tests:

```bash
# All tests
go test ./...

# Specific package
go test ./internal/parser/ -v -run TestParseMySQLValues

# With coverage
make cover
```

## Schema

See [SCHEMA.md](SCHEMA.md) for the full database schema.

## Sources & licensing

See [SOURCES.md](SOURCES.md) for detailed source attribution and licensing.

## CI/CD

On push to `main` (when source data or build code changes):
1. Tests run
2. Database builds from scratch
3. GitHub Release created with the `.db` file attached
