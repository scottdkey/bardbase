# Sources — Original Shakespeare Texts

⚠️ **READ-ONLY** — Nothing in this directory should ever be modified by any automated process.

These are unmodified original source files. If you need derived or curated data, put it in `projects/data/`.

## Contents

| Directory | Source | License | Files | Description |
|-----------|--------|---------|-------|-------------|
| `oss/` | [Open Source Shakespeare](https://opensourceshakespeare.org) | Public Domain | 1 SQL dump | Globe-based modern spelling, full MySQL dump (38 works) |
| `se/` | [Standard Ebooks](https://standardebooks.org) | CC0 | 37 JSONs + 2 XHTMLs | Modern-spelling, verse-level text for all 37 plays + sonnets + poems |
| `lexicon/` | [Perseus Digital Library](http://www.perseus.tufts.edu) | CC BY-SA 3.0 | 20,070 XMLs | Schmidt's Shakespeare Lexicon (TEI XML, one file per entry) |

## Source Status

| Source | Total Expected | On Disk | Coverage |
|--------|---------------|---------|----------|
| OSS/Moby | 1 SQL dump | 1 | 100% |
| Standard Ebooks | 37 plays + sonnets + poems | 39 files | 100% |
| Schmidt Lexicon | 20,097 entries | 20,070 XMLs | 99.9% |

## Contract

Every other project in this monorepo treats `sources/` as **immutable input**:

- `db-builder` reads from here to build the SQLite database
- `web` never touches these files
- `data` contains curated mappings derived from studying these sources

If a source file needs correction, the fix belongs upstream with the original publisher — not here.

## Planned Additions

These directories will be added as new sources are downloaded:

| Directory | Source | License | Status |
|-----------|--------|---------|--------|
| `perseus-texts/` | Perseus Shakespeare Play Texts | CC BY-SA 3.0 | Not started — needs scraping |
| `eebo-tcp/` | EEBO-TCP First Folio + Quartos | Public Domain | Not started — available on GitHub |
| `gutenberg/` | Project Gutenberg texts | Public Domain | Not started |

See [SOURCES.md](SOURCES.md) for the full catalog of all known Shakespeare text sources.
