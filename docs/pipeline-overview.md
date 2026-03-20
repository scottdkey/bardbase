# Build Pipeline Overview

The `capell` binary executes a deterministic, multi-phase pipeline that reads from committed source files and writes a single SQLite database (`build/bardbase.db`). Phases run sequentially; each phase is logged to the `import_log` table with its duration and row count.

## Phase Order

```
 1. oss            → works, characters, text_lines (globe_moby edition)
 2. lexicon        → lexicon_entries, lexicon_senses, lexicon_citations
 3. se             → text_lines (se_modern edition, 37 plays)
 4. poetry         → text_lines (se_modern edition, sonnets + poems)
 5. perseus        → text_lines (perseus_globe edition, Globe line numbers)
 6. folio          → text_lines (first_folio edition, original spelling)
 7. folger         → text_lines (folger_shakespeare edition, CC BY-NC 3.0)
 8. eebo-quartos   → text_lines (per-quarto editions, original spelling)
 9. onions         → reference_entries (Shakespeare Glossary 1911)
10. abbott         → reference_entries (Shakespearian Grammar 1877)
11. bartlett       → reference_entries (Concordance 1896)
12. henley-farmer  → reference_entries (Slang Dictionary 1890-1904)
13. standalone     → text_lines (biblical/classical passages cited by Schmidt)
14. attributions   → attributions (display rules for all sources)
15. mappings       → line_mappings (Needleman-Wunsch alignment)
16. citations      → citation_matches (multi-strategy matching)
17. ref-citations  → reference_citation_matches (reference entry citation resolution)
18. fts            → lexicon_fts, text_fts virtual tables
```

## What Each Phase Does

### Phase 1 — OSS (`oss`)

**Source**: `projects/sources/oss/` (MySQL dump from opensourceshakespeare.org)

The OSS parser (`parser/mysql.go`) reads the MySQL dump and extracts:
- `works` — 43 canonical works with metadata (genre, date, word count)
- `characters` — 1,265+ dramatis personae, linked to works
- `text_lines` — all speeches, stage directions, prologues, epilogues, songs

This phase establishes the canonical `works` table that every subsequent phase references. The `oss_id` (e.g. `hamlet`, `12night`) is the foreign-key anchor for all cross-phase linking.

**Edition created**: `globe_moby` — Globe/Moby Modern Text

### Phase 2 — Lexicon (`lexicon`)

**Source**: `projects/sources/lexicon/entries/` (TEI XML files from Perseus/Schmidt)

The lexicon parser (`parser/lexicon.go`) reads Schmidt's *Shakespeare Lexicon and Quotation Dictionary* in TEI XML format. For each `<entry>` it produces:
- One `lexicon_entries` row (headword, full definition text, raw XML)
- One or more `lexicon_senses` rows (numbered senses)
- One `lexicon_citations` row per `<bibl>` element (work abbreviation, act, scene, line, quoted text)

Citations are parsed but not yet resolved to `text_lines` — that happens in Phase 16.

See [citation-resolution.md](citation-resolution.md) for the full citation resolution algorithm.

### Phase 3 — Standard Ebooks Plays (`se`)

**Source**: `projects/sources/se/` (XHTML files, downloaded/cached)

The SE parser (`parser/seplay.go`) reads Standard Ebooks XHTML and maps each play to its `works` row via a name-normalization lookup. It produces scene-relative sequential line numbers used for citation matching.

**Edition created**: `se_modern`

### Phase 4 — Standard Ebooks Poetry (`poetry`)

**Source**: `projects/sources/se/` (XHTML files, same cache as Phase 3)

Same parser family (`parser/sepoetry.go`), handles sonnets and longer poems. Sonnet number and stanza are extracted and stored in `text_lines.sonnet_number` / `text_lines.stanza`.

**Edition created**: `se_modern` (same edition record, different works)

### Phase 5 — Perseus (`perseus`)

**Source**: `projects/sources/perseus-plays/` (TEI XML)

The Perseus parser (`parser/perseus.go`) reads the Perseus Digital Library TEI texts of Shakespeare's plays, which carry Globe edition line numbers in `<l n="...">` attributes. These become the authoritative scene-relative `line_number` values used by citation resolution.

**Edition created**: `perseus_globe`

### Phase 6 — First Folio (`folio`)

**Source**: `projects/sources/eebo-tcp/A11954.xml` (EEBO-TCP TEI XML)

The Folio parser (`parser/folio.go`) reads the 1623 First Folio in its EEBO-TCP diplomatic transcription. Long-s (`ſ`) is normalized to `s`; `<g ref="char:EOLhyphen">` line-end hyphens are dropped; `<gap>` elements become `[?]`. Both prose (`<p>`) and verse (`<l>`) speech elements are extracted.

**Edition created**: `first_folio`

### Phase 7 — Folger Shakespeare (`folger`)

**Source**: `projects/sources/folger/teisimple/` (TEIsimple XML files)

The Folger parser (`parser/folger.go`) reads TEIsimple XML files from the Folger Shakespeare Library. These include word-by-word POS annotation (MorphAdorner) and Folger Through Line Numbers (FTLNs). Word annotations are stored in the `word_annotations` table; Folger-specific stage direction metadata (`stage_type`, `stage_who`) is preserved.

**Edition created**: `folger_shakespeare` — tagged with `license_tier = 'cc-by-nc'` for downstream filtering.

### Phase 8 — EEBO-TCP Early Quartos (`eebo-quartos`)

**Source**: `projects/sources/eebo-tcp/` (TEI XML files), metadata from `projects/data/eebo_quartos.json`

Parses EEBO-TCP diplomatic transcriptions of early quartos (Q1 Hamlet 1603, Q1 1H4 1598, etc.). Each quarto gets its own edition record (e.g. `q1_hamlet_1603`). These are textually distinct from the First Folio and provide unique comparison data. Uses the same EEBO-TCP source as the First Folio importer.

**Editions created**: One per quarto (e.g. `q1_hamlet_1603`)

### Phase 9 — Onions Shakespeare Glossary (`onions`)

**Source**: `projects/sources/onions/shakespeare-glossary-1911.txt` (OCR plain text)

Imports C. T. Onions' *Shakespeare Glossary* (1911 edition, public domain). The OCR text is split into individual glossary entries by headword detection heuristics. Raw entry text is stored in `reference_entries`; citations are resolved later in Phase 17.

### Phase 10 — Abbott Shakespearian Grammar (`abbott`)

**Source**: `projects/sources/abbott/shakespearian-grammar-1877.txt` (OCR plain text)

Imports E. A. Abbott's *Shakespearian Grammar* (1877 edition, public domain). The grammar is organized as numbered paragraphs (§1 – §515+). Each paragraph is stored as a `reference_entries` row with the section number as headword.

### Phase 11 — Bartlett's Concordance (`bartlett`)

**Source**: `projects/sources/bartlett/concordance-1896.txt` (OCR plain text)

Imports John Bartlett's *Complete Concordance to Shakespeare* (1896, public domain). Each headword group is stored as a `reference_entries` row. ALL-CAPS OCR headers and page numbers are filtered out.

### Phase 12 — Henley & Farmer Slang Dictionary (`henley-farmer`)

**Source**: `projects/sources/henley-farmer/slang-vol01.txt` through `slang-vol07.txt` (OCR plain text)

Imports Shakespeare-related entries from Henley & Farmer's *Slang and Its Analogues* (1890-1904, 7 volumes, public domain). Only entries containing "Shak" (Shakespeare citations) are imported.

### Phase 13 — Standalone Passages (`standalone`)

**Source**: `projects/data/standalone_passages.json`

Imports text_line rows for passages from non-Shakespeare works cited by Schmidt (biblical, classical). These have no existing DB representation; adding them allows citation resolution to find matches. Must run before Phase 16.

### Phase 14 — Attributions (`attributions`)

Populates the `attributions` table with display rules for all sources — which require legal attribution, what text to show, where to show it (footer, credits page, inline), and whether a link-back or license notice is required. This runs after all source data is imported so it can reference the fully populated `sources` table.

### Phase 15 — Cross-edition Mappings (`mappings`)

Reads all loaded editions and computes pairwise scene-level alignments, storing results in `line_mappings`. Must run before citations because the citation resolver uses cross-edition propagation to fill gaps. The alignment algorithm is Needleman-Wunsch with Jaccard-similarity scoring; large scenes fall back to simple positional alignment.

See [line-alignment.md](line-alignment.md) for the full algorithm.

### Phase 16 — Citation Resolution (`citations`)

Iterates every `lexicon_citations` row and attempts to find the matching `text_lines` row in the `perseus_globe` edition (which carries authoritative Globe line numbers). Uses a cascade of strategies from exact quote matching down to headword propagation. Requires mappings to be built first (Phase 15) for cross-edition propagation.

See [citation-resolution.md](citation-resolution.md) for the full algorithm.

### Phase 17 — Reference Citation Resolution (`ref-citations`)

Resolves citations found within reference entries (Onions, Abbott, Bartlett, Henley-Farmer) to `text_lines` rows. Each reference source uses source-specific abbreviation translation to map to Schmidt abbreviations before resolution.

### Phase 18 — FTS5 Indexing (`fts`)

Populates the two FTS5 virtual tables (`lexicon_fts`, `text_fts`) from the now-complete content tables. Runs last so all text is present before indexing.

See [fts-search.md](fts-search.md) for setup and query examples.

## Editions in the Final Database

| `short_code` | Description | Source | License |
|---|---|---|---|
| `globe_moby` | Globe / Moby Modern Text | Open Source Shakespeare | Public Domain |
| `se_modern` | Standard Ebooks Modern Text | Standard Ebooks | CC0 |
| `perseus_globe` | Globe edition with dual line numbers | Perseus Digital Library | CC BY-SA 3.0 |
| `first_folio` | First Folio 1623 (original spelling) | EEBO-TCP A11954 | CC0 |
| `folger_shakespeare` | Folger Shakespeare (modern scholarly) | Folger Shakespeare Library | CC BY-NC 3.0 |
| `q1_*` | EEBO-TCP Early Quartos (per-quarto) | EEBO-TCP | CC0 |

## Running the Pipeline

```bash
make capell run          # full pipeline, uses cached SE source files
make capell run-fresh    # full pipeline, re-downloads SE sources first
make capell test         # run all unit tests
```

Individual steps can be run in isolation (useful during development):

```bash
go run ./cmd/build -step oss
go run ./cmd/build -step lexicon
go run ./cmd/build -step se
go run ./cmd/build -step poetry
go run ./cmd/build -step perseus
go run ./cmd/build -step folio
go run ./cmd/build -step folger
go run ./cmd/build -step eebo-quartos
go run ./cmd/build -step onions
go run ./cmd/build -step abbott
go run ./cmd/build -step bartlett
go run ./cmd/build -step henley-farmer
go run ./cmd/build -step standalone
go run ./cmd/build -step attributions
go run ./cmd/build -step mappings
go run ./cmd/build -step citations
go run ./cmd/build -step ref-citations
go run ./cmd/build -step fts
```

Sources can be excluded from the build:

```bash
go run ./cmd/build --exclude folger          # skip Folger (CC BY-NC)
go run ./cmd/build --exclude folger,bartlett # skip multiple sources
```

Progress is printed to stdout in a step-banner format and recorded in the `import_log` table for post-hoc analysis.
