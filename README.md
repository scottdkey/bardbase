# Bardbase

[![Build](https://github.com/scottdkey/bardbase/actions/workflows/build.yml/badge.svg)](https://github.com/scottdkey/bardbase/actions/workflows/build.yml)
[![Latest Release](https://img.shields.io/github/v/release/scottdkey/bardbase)](https://github.com/scottdkey/bardbase/releases/latest)
[![Download](https://img.shields.io/github/downloads/scottdkey/bardbase/total)](https://github.com/scottdkey/bardbase/releases/latest)

A SQLite database of Shakespeare's complete works — multiple editions, full-text search, and a complete Shakespeare lexicon.

## Quick Start

**Download the pre-built database** from the [latest release](https://github.com/scottdkey/bardbase/releases/latest).

**Build from source** (requires Go 1.22+):

```bash
git clone https://github.com/scottdkey/bardbase.git
cd bardbase
make capell test   # run tests
make capell run    # build → build/bardbase.db
```

## Make Commands

```bash
# capell
make capell build          # compile binary
make capell test           # run all tests
make capell run            # full build pipeline (uses cached sources)
make capell run-fresh      # full build, re-download source files
make capell lint           # go vet
make capell cover          # test coverage report
make capell clean          # remove artifacts

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
bardbase/
├── projects/
│   ├── sources/       source texts — committed, read-only
│   ├── data/          reference JSON (work mappings, attributions)
│   ├── capell/        Go pipeline → produces SQLite database
│   └── web/           SvelteKit PWA — Variorum
├── Makefile
└── .github/workflows/
```

## Schema

See [projects/capell/SCHEMA.md](projects/capell/SCHEMA.md) for all tables, indexes, and example queries.

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
