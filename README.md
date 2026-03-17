# Heminge

[![Build](https://github.com/scottdkey/heminge/actions/workflows/build.yml/badge.svg)](https://github.com/scottdkey/heminge/actions/workflows/build.yml)
[![Latest Release](https://img.shields.io/github/v/release/scottdkey/heminge)](https://github.com/scottdkey/heminge/releases/latest)
[![Download](https://img.shields.io/github/downloads/scottdkey/heminge/total)](https://github.com/scottdkey/heminge/releases/latest)

A SQLite database of Shakespeare's complete works — multiple editions, full-text search, and a complete Shakespeare lexicon.

## Quick Start

**Download the pre-built database** from the [latest release](https://github.com/scottdkey/heminge/releases/latest).

**Build from source** (requires Go 1.22+):

```bash
git clone https://github.com/scottdkey/heminge.git
cd heminge
make db-builder test   # run tests
make db-builder run    # build → build/heminge.db
```

## Make Commands

```bash
# db-builder
make db-builder build          # compile binary
make db-builder test           # run all tests
make db-builder run            # full build pipeline (uses cached sources)
make db-builder run-fresh      # full build, re-download source files
make db-builder lint           # go vet
make db-builder cover          # test coverage report
make db-builder clean          # remove artifacts

# sources
make sources verify            # check all source files exist
make sources stats             # file counts per source

# cross-project
make test-all                  # tests across all projects
make clean-all                 # clean all projects
make help                      # show all commands
```

## What's in it

| Content | Source | Status |
|---------|--------|--------|
| 43 works, 1,265+ characters, ~36K lines | Open Source Shakespeare / Moby | ✅ |
| Modern-spelling text, all 37 plays | Standard Ebooks | ✅ |
| 154 sonnets + 5 poems | Standard Ebooks | ✅ |
| Globe edition with dual line numbering | Perseus Digital Library | ✅ |
| First Folio 1623 (original spelling) | EEBO-TCP (A11954) | ✅ |
| 20,070 lexicon entries, ~200K citations | Perseus / Schmidt Lexicon | ✅ |
| Cross-edition line mappings | Generated | ✅ |
| Full-text search (FTS5) | Generated | ✅ |

## Structure

```
heminge/
├── projects/
│   ├── sources/       source texts — committed, read-only
│   ├── data/          reference JSON (work mappings, attributions)
│   ├── db-builder/    Go pipeline → produces SQLite database
│   └── web/           SvelteKit PWA (future)
├── Makefile
└── .github/workflows/
```

## Schema

See [projects/db-builder/SCHEMA.md](projects/db-builder/SCHEMA.md) for all tables, indexes, and example queries.

## Documentation

| Doc | What it covers |
|-----|---------------|
| [docs/pipeline-overview.md](docs/pipeline-overview.md) | How all 9 build phases connect — OSS → SE plays → SE poetry → lexicon → Perseus → First Folio → alignments → citations → FTS |
| [docs/line-alignment.md](docs/line-alignment.md) | Deep dive on Needleman-Wunsch sequence alignment used to produce cross-edition `line_mappings` |
| [docs/citation-resolution.md](docs/citation-resolution.md) | Deep dive on the 5-strategy cascade that links ~200K Schmidt citations to actual `text_lines` rows |
| [docs/fts-search.md](docs/fts-search.md) | FTS5 setup, Porter stemming, and query examples for `lexicon_fts` and `text_fts` |

## Sources & Attribution

See [ATTRIBUTION.md](ATTRIBUTION.md) and [projects/sources/SOURCES.md](projects/sources/SOURCES.md).

## CI/CD

Pushes and PRs run tests + build. Releases require a manual workflow dispatch with the release flag enabled — never automatic.
