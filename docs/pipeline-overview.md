# Build Pipeline Overview

The `capell` binary executes a deterministic, multi-phase pipeline that reads from committed source files and writes a single SQLite database (`build/bardbase.db`). Phases run sequentially; each phase is logged to the `import_log` table with its duration and row count.

## Phase Order

```
 1. oss          → works, characters, text_lines (globe_moby edition)
 2. lexicon      → lexicon_entries, lexicon_senses, lexicon_citations
 3. se           → text_lines (se_modern edition, 37 plays)
 4. poetry       → text_lines (se_modern edition, sonnets + poems)
 5. perseus      → text_lines (perseus_globe edition, Globe line numbers)
 6. folio        → text_lines (first_folio edition, original spelling)
 7. attributions → attributions (display rules for all sources)
 8. mappings     → line_mappings (Needleman-Wunsch alignment)
 9. citations    → citation_matches (multi-strategy matching)
10. fts          → lexicon_fts, text_fts virtual tables
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

Citations are parsed but not yet resolved to `text_lines` — that happens in Phase 9.

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

**Source**: `projects/sources/perseus/` (TEI XML)

The Perseus parser (`parser/perseus.go`) reads the Perseus Digital Library TEI texts of Shakespeare's plays, which carry Globe edition line numbers in `<l n="...">` attributes. These become the authoritative scene-relative `line_number` values used by citation resolution.

**Edition created**: `perseus_globe`

### Phase 6 — First Folio (`folio`)

**Source**: `projects/sources/folio/` (EEBO-TCP TEI XML, document A11954)

The Folio parser (`parser/folio.go`) reads the 1623 First Folio in its EEBO-TCP diplomatic transcription. Long-s (`ſ`) is normalized to `s`; `<g ref="char:EOLhyphen">` line-end hyphens are dropped; `<gap>` elements become `[?]`. Both prose (`<p>`) and verse (`<l>`) speech elements are extracted.

**Edition created**: `first_folio`

### Phase 7 — Attributions (`attributions`)

Populates the `attributions` table with display rules for all sources — which require legal attribution, what text to show, where to show it (footer, credits page, inline), and whether a link-back or license notice is required. This runs after all source data is imported so it can reference the fully populated `sources` table.

### Phase 8 — Cross-edition Mappings (`mappings`)

Reads all loaded editions and computes pairwise scene-level alignments, storing results in `line_mappings`. Must run before citations because the citation resolver uses cross-edition propagation to fill gaps. The alignment algorithm is Needleman-Wunsch with Jaccard-similarity scoring; large scenes fall back to simple positional alignment.

See [line-alignment.md](line-alignment.md) for the full algorithm.

### Phase 9 — Citation Resolution (`citations`)

Iterates every `lexicon_citations` row and attempts to find the matching `text_lines` row in the `perseus_globe` edition (which carries authoritative Globe line numbers). Uses a cascade of strategies from exact quote matching down to headword propagation. Requires mappings to be built first (Phase 8) for cross-edition propagation.

See [citation-resolution.md](citation-resolution.md) for the full algorithm.

### Phase 10 — FTS5 Indexing (`fts`)

Populates the two FTS5 virtual tables (`lexicon_fts`, `text_fts`) from the now-complete content tables. Runs last so all text is present before indexing.

See [fts-search.md](fts-search.md) for setup and query examples.

## Editions in the Final Database

| `short_code` | Description | Source |
|---|---|---|
| `globe_moby` | Globe / Moby Modern Text | Open Source Shakespeare |
| `se_modern` | Standard Ebooks Modern Text | Standard Ebooks |
| `perseus_globe` | Globe edition with dual line numbers | Perseus Digital Library |
| `first_folio` | First Folio 1623 (original spelling) | EEBO-TCP A11954 |

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
go run ./cmd/build -step attributions
go run ./cmd/build -step mappings
go run ./cmd/build -step citations
go run ./cmd/build -step fts
```

Progress is printed to stdout in a step-banner format and recorded in the `import_log` table for post-hoc analysis.
