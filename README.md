# Shakespeare Database

[![Build](https://github.com/scottdkey/shakespeare_db/actions/workflows/build.yml/badge.svg)](https://github.com/scottdkey/shakespeare_db/actions/workflows/build.yml)
[![Latest Release](https://img.shields.io/github/v/release/scottdkey/shakespeare_db)](https://github.com/scottdkey/shakespeare_db/releases/latest)
[![Download](https://img.shields.io/github/downloads/scottdkey/shakespeare_db/total)](https://github.com/scottdkey/shakespeare_db/releases/latest)

A comprehensive SQLite database of Shakespeare's complete works with full-text search, multi-edition comparison, and a complete Shakespeare lexicon — built from multiple open-source and public domain sources.

## What's in it

| Content | Source | License | Status |
|---------|--------|---------|--------|
| 43 works, 1,265+ characters, ~36K text lines | [Open Source Shakespeare](https://opensourceshakespeare.org) / Moby | Public Domain | ✅ Imported |
| Modern-spelling verse-level text for all 37 plays | [Standard Ebooks](https://standardebooks.org) | CC0 | ✅ Imported |
| 154 sonnets + 5 poems (Venus & Adonis, etc.) | Standard Ebooks | CC0 | ✅ Imported |
| 20,070 lexicon entries with ~200K+ citations | [Perseus Digital Library](http://www.perseus.tufts.edu) — Schmidt Lexicon | CC BY-SA 3.0 | ✅ Imported |
| Cross-edition line mappings (OSS ↔ SE) | Generated (Needleman-Wunsch alignment) | — | ✅ Generated |
| Citation-to-text resolution with confidence scores | Generated | — | ✅ Generated |
| Full-text search (FTS5) across text + lexicon | Generated | — | ✅ Generated |
| Folger Shakespeare Library reference URLs | [Folger](https://www.folger.edu) | — | ⚠️ Partial |

### Planned Sources (Not Yet Imported)

| Content | Source | License | Priority |
|---------|--------|---------|----------|
| Globe text with First Folio line numbers | [Perseus Digital Library](http://www.perseus.tufts.edu) — Play Texts | CC BY-SA 3.0 | P2 |
| First Folio diplomatic transcription (1623) | [EEBO-TCP](https://github.com/textcreationpartnership) | Public Domain | P2 |
| Early Quarto diplomatic transcriptions | EEBO-TCP | Public Domain | P3 |
| Modern-spelling alternative editions | [Project Gutenberg](https://www.gutenberg.org) | Public Domain | P3 |

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

# Build the database
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

## The Go Pipeline

The `db-builder` project is a Go pipeline that reads from `sources/` and produces a single SQLite database. It runs in 8 phases:

1. **Schema** — Creates all tables, indexes, and FTS5 virtual tables
2. **OSS Import** — Parses the MySQL dump, imports works, characters, and ~36K text lines
3. **SE Plays** — Downloads (or reads cached) Standard Ebooks plays, imports as a second edition
4. **SE Poetry** — Imports all 154 sonnets + 5 poems from Standard Ebooks
5. **Lexicon** — Parses 20,070 Schmidt Lexicon XML entries with senses and citations
6. **Citations** — Resolves lexicon citations to actual text lines with confidence scores
7. **Line Mappings** — Aligns OSS and SE editions scene-by-scene using Needleman-Wunsch
8. **FTS + Summary** — Builds full-text search indexes and logs build statistics

All parsers are pure functions with dedicated tests. **197 tests** across 4 packages.

## Testing

```bash
# All tests
make db-builder test

# With coverage
make db-builder cover

# Verbose
cd projects/db-builder && go test ./... -v
```

## Schema

See [projects/db-builder/SCHEMA.md](projects/db-builder/SCHEMA.md) for the complete database schema with all tables, indexes, and example queries.

**Key tables**: `sources`, `works`, `characters`, `editions`, `attributions`, `text_lines`, `text_divisions`, `lexicon_entries`, `lexicon_senses`, `lexicon_citations`, `citation_matches`, `line_mappings`, `import_log`

**FTS5 tables**: `lexicon_fts` (search lexicon), `text_fts` (search all text)

## Sources & Licensing

See [projects/sources/SOURCES.md](projects/sources/SOURCES.md) for detailed source attribution, licensing requirements, and the full catalog of known Shakespeare text sources.

**Attribution summary:**
- **Perseus Digital Library** (CC BY-SA 3.0) — attribution **legally required**
- **Open Source Shakespeare** (Public Domain) — courtesy credit
- **Standard Ebooks** (CC0) — no requirements
- **EEBO-TCP** (Public Domain) — courtesy credit appreciated

## CI/CD

- **Push to main / PR** → tests + build + upload artifact
- **Manual trigger** → tests + build + optional GitHub Release (must check "Create a release")

Releases are never automatic — they require a manual workflow dispatch with the release flag enabled.

All CI actions use Node.js 24+ (`actions/checkout@v6`, `actions/setup-go@v6`, `actions/upload-artifact@v7`).
