# Sources — Original Shakespeare Texts

⚠️ **READ-ONLY** — Nothing in this directory should ever be modified by any automated process.

These are unmodified original source files. If you need derived or curated data, put it in `projects/data/`.

## Contents

| Directory | Source | License | Files | Description |
|-----------|--------|---------|-------|-------------|
| `oss/` | [Open Source Shakespeare](https://opensourceshakespeare.org) | Public Domain | 1 SQL dump | Globe-based modern spelling, full MySQL dump |
| `se/` | [Standard Ebooks](https://standardebooks.org) | CC0 | 37 JSONs + 2 XHTMLs | Modern-spelling text for all 37 plays + sonnets + poems |
| `lexicon/` | [Perseus Digital Library](http://www.perseus.tufts.edu) | CC BY-SA 3.0 | 20,070 XMLs | Schmidt's Shakespeare Lexicon (TEI XML, one file per entry) |
| `perseus-plays/` | [Perseus Digital Library](http://www.perseus.tufts.edu) | CC BY-SA 3.0 | 43 XMLs | Globe edition play texts with dual Globe/First Folio line numbering |
| `eebo-tcp/` | [EEBO-TCP](https://github.com/textcreationpartnership) | CC0 | 8 XMLs | First Folio 1623 (A11954) + 7 early Quarto editions |

## Source Status

| Source | Files on Disk | Imported | Edition |
|--------|--------------|----------|---------|
| OSS/Moby | 1 SQL dump | ✅ | `globe_moby` |
| Standard Ebooks plays | 37 JSONs | ✅ | `se_modern` |
| Standard Ebooks poetry | 2 XHTMLs | ✅ | `se_modern` |
| Schmidt Lexicon | 20,070 XMLs | ✅ | — (lexicon tables) |
| Perseus Globe plays | 43 XMLs | ✅ | `perseus_globe` |
| First Folio (EEBO-TCP A11954) | 1 XML | ✅ | `first_folio` |
| Early Quartos (EEBO-TCP) | 7 XMLs | ⚠️ downloaded, not yet imported | — |

## Contract

Every other project in this monorepo treats `sources/` as **immutable input**:

- `capell` reads from here to build the SQLite database
- `web` never touches these files
- `data` contains curated mappings derived from studying these sources

If a source file needs correction, the fix belongs upstream with the original publisher — not here.

See [SOURCES.md](SOURCES.md) for the full catalog of all known Shakespeare text sources and their licensing details.
