# Sources — Original Shakespeare Texts

⚠️ **READ-ONLY** — Nothing in this directory should ever be modified by any automated process.

These are unmodified original source files. If you need derived or curated data, put it in `projects/data/`.

## Contents

| Directory | Source | License | Description |
|-----------|--------|---------|-------------|
| `oss/` | [Open Source Shakespeare](https://opensourceshakespeare.org) | Public Domain | Globe-based modern spelling, full MySQL dump |
| `se/` | [Standard Ebooks](https://standardebooks.org) | CC0 | Modern-spelling, verse-level JSON exports |
| `lexicon/` | [Perseus Digital Library](http://www.perseus.tufts.edu) | CC BY-SA 3.0 | Schmidt's Shakespeare Lexicon XML entries |

## Contract

Every other project in this monorepo treats `sources/` as **immutable input**:

- `db-builder` reads from here to build the SQLite database
- `web` never touches these files
- `data` contains curated mappings derived from studying these sources

If a source file needs correction, the fix belongs upstream with the original publisher — not here.
