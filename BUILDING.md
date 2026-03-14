# Building the Shakespeare Database

## Quick Start

The pre-built `shakespeare.db` is included in the repository. If you want to rebuild from scratch:

```bash
# Phase 0 + Phase 1: Parse OSS SQL dump + Perseus lexicon XMLs
python3 tools/shakespeare-db-builder.py --rebuild

# Phase 2: Import Standard Ebooks texts (37 plays + 6 poems)
python3 tools/standard-ebooks-importer.py

# Phase 2b: Import poetry (sonnets, Venus & Adonis, etc.) + Folger URLs
python3 tools/se-poetry-importer.py
```

## Database Contents

| Table | Count | Description |
|-------|-------|-------------|
| **works** | 43 | All Shakespeare works with Schmidt abbreviations |
| **characters** | 1,265 | Every character from every play |
| **text_divisions** | 1,703 | Acts/scenes with line counts |
| **text_lines** | 136,878 | Full text across 2 editions |
| **lexicon_entries** | 7,577+ | Schmidt dictionary entries (growing as scraper runs) |
| **lexicon_senses** | 9,780+ | Numbered definitions within entries |
| **lexicon_citations** | 81,602+ | References to Shakespeare passages |

## Editions

| Edition | Source | License | Lines | Works |
|---------|--------|---------|-------|-------|
| **OSS/Moby** | Open Source Shakespeare | Public Domain | 35,629 | 43 |
| **Standard Ebooks** | standardebooks.org | CC0 1.0 | 101,249 | 43 |

## Data Sources

### 1. Open Source Shakespeare / Moby (`oss_sql/oss-db-full.sql`)
- Globe-based modern spelling text
- Already in the repository
- Public Domain — no attribution required

### 2. Perseus Digital Library — Schmidt Shakespeare Lexicon
- XML files scraped from Perseus (`xml/entries/`)
- CC BY-SA 3.0 — **attribution required**
- Scraper runs as a background service, adding entries over time
- Rebuild the database after new XMLs are downloaded

### 3. Standard Ebooks
- Clean XHTML with semantic markup (CC0)
- Downloaded from GitHub repos at build time (cached locally in `standard-ebooks-cache/`)
- All 37 plays + 6 poems (sonnets, Venus & Adonis, Rape of Lucrece, etc.)
- No attribution legally required

### 4. Folger Shakespeare Library (Reference Only)
- URLs stored in `works.folger_url` — **never downloaded**
- Links to https://www.folger.edu/explore/shakespeares-works/

## Future Sources (Planned)

- **Perseus Shakespeare play texts** — TEI XML with dual Globe/F1 line numbering
- **EEBO-TCP First Folio** — Original spelling diplomatic transcription (Public Domain)
- **EEBO-TCP Quartos** — Per-play quarto transcriptions
- **Project Gutenberg** — Additional modern-spelling editions

## Attribution

The database includes a `sources` table that tracks licensing and attribution requirements
for every data source. Any application using this database should display attribution for
sources where `attribution_required = 1`.

Currently required attributions:
- **Perseus Digital Library**: "Alexander Schmidt, Shakespeare Lexicon and Quotation Dictionary.
  Provided by the Perseus Digital Library, Tufts University. Licensed under CC BY-SA 3.0."
