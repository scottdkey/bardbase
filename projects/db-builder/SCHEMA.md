# Heminge Database Schema

SQLite database built by the Go pipeline from multiple open-source Shakespeare text sources.

This document reflects the **actual schema** produced by the SQL files in `projects/db-builder/internal/db/schema/`.

---

## Design Principles

1. **Every line belongs to an edition** — no ambiguity about which version you're reading
2. **Citations resolve to actual text** — lexicon entries link to retrievable passages via `citation_matches`
3. **Side-by-side comparison** — any two editions of the same work, aligned by `line_mappings`
4. **Attribution is first-class** — every source tracked with license requirements and display rules
5. **Cursor pagination** — all primary tables use `id > :cursor ORDER BY id LIMIT :limit`
6. **Single SQLite file** — ships with the app, no server needed
7. **Full-text search** — FTS5 indexes on both text and lexicon

---

## Tables

### `sources`

Where data comes from. Each row represents one upstream data provider.

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER PK | Auto-increment |
| `name` | TEXT UNIQUE | Full display name |
| `short_code` | TEXT UNIQUE | Abbreviation (e.g. `oss`, `se`, `perseus`) |
| `url` | TEXT | Homepage URL |
| `license` | TEXT | License identifier (e.g. `PD`, `CC-BY-SA-3.0`, `CC0`) |
| `license_url` | TEXT | Link to license text |
| `attribution_text` | TEXT | Required/courtesy attribution text |
| `attribution_required` | BOOLEAN | `1` if legally required |
| `notes` | TEXT | Additional context |
| `imported_at` | TIMESTAMP | When this source was imported |

### `works`

Canonical list of Shakespeare's works — one row per work regardless of editions.

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER PK | Auto-increment |
| `oss_id` | TEXT UNIQUE | OSS database identifier (e.g. `hamlet`, `12night`) |
| `title` | TEXT | Display title |
| `full_title` | TEXT | Full formal title |
| `short_title` | TEXT | Abbreviated title |
| `schmidt_abbrev` | TEXT | Schmidt lexicon abbreviation (e.g. `ham`, `tp`) |
| `work_type` | TEXT | `play`, `poem`, or `sonnet` |
| `date_composed` | INTEGER | Approximate year |
| `genre_type` | TEXT | `tragedy`, `comedy`, `history`, `poem`, `sonnet` |
| `total_words` | INTEGER | Word count (from OSS) |
| `total_paragraphs` | INTEGER | Paragraph count (from OSS) |
| `source_text` | TEXT | Source text identifier |
| `folger_url` | TEXT | Folger Shakespeare Library reference URL |
| `perseus_id` | TEXT | Perseus Digital Library text identifier |
| `notes` | TEXT | Additional context |

**Indexes**: `oss_id`, `schmidt_abbrev`, `work_type`

### `characters`

Dramatis personae per work.

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER PK | Auto-increment |
| `char_id` | TEXT UNIQUE | Character identifier (e.g. `hamlet_hamlet`) |
| `name` | TEXT | Display name |
| `abbrev` | TEXT | Abbreviation |
| `work_id` | INTEGER FK → works | Which work this character belongs to |
| `oss_work_id` | TEXT | OSS work identifier (for import linkage) |
| `description` | TEXT | Character description |
| `speech_count` | INTEGER | Number of speeches |

**Indexes**: `work_id`

### `editions`

A specific version/text from a specific source. Multiple editions can exist per work.

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER PK | Auto-increment |
| `name` | TEXT | Full name (e.g. `Globe/Moby Modern Text`) |
| `short_code` | TEXT UNIQUE | Abbreviation (e.g. `globe_moby`, `se_modern`) |
| `source_id` | INTEGER FK → sources | Which source provided this edition |
| `year` | INTEGER | Publication year |
| `editors` | TEXT | Editor(s) |
| `description` | TEXT | Description |
| `notes` | TEXT | Additional context |

### `attributions`

Display rules for source credits. Tracks attribution requirements for ALL sources — both legally required (CC BY-SA) and voluntary (public domain courtesy credits). Consuming applications use this to build credits pages and inline attribution.

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER PK | Auto-increment |
| `source_id` | INTEGER FK → sources (UNIQUE) | One attribution per source |
| `required` | BOOLEAN | `1` if legally required |
| `attribution_text` | TEXT | Text to display |
| `attribution_html` | TEXT | HTML-formatted attribution |
| `display_format` | TEXT | Where to show: `footer`, `credits_page`, `inline` |
| `display_context` | TEXT | When to show: `always`, `on_source_content`, `credits_only` |
| `display_priority` | INTEGER | Sort order (lower = more prominent) |
| `requires_link_back` | BOOLEAN | Must link to source URL |
| `link_back_url` | TEXT | URL to link back to |
| `requires_license_notice` | BOOLEAN | Must display license text |
| `license_notice_text` | TEXT | License notice to display |
| `requires_author_credit` | BOOLEAN | Must credit specific author |
| `author_credit_text` | TEXT | Author credit text |
| `share_alike_required` | BOOLEAN | CC BY-SA: derived works must use compatible license |
| `commercial_use_allowed` | BOOLEAN | License permits commercial use |
| `modification_allowed` | BOOLEAN | License permits modifications |
| `notes` | TEXT | Additional context |
| `created_at` | TIMESTAMP | When created |

### `text_lines`

Every line of text in every edition. This is the largest table.

`line_number` is a **scene-relative sequential line number** used for consistent cross-edition referencing and citation matching (Globe-style numbering).

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER PK | Auto-increment |
| `work_id` | INTEGER FK → works | Which work |
| `edition_id` | INTEGER FK → editions | Which edition |
| `act` | INTEGER | Act number (NULL for poems/sonnets) |
| `scene` | INTEGER | Scene number (NULL for poems/sonnets) |
| `paragraph_num` | INTEGER | OSS paragraph number |
| `line_number` | INTEGER | Scene-relative line number |
| `character_id` | INTEGER FK → characters | Speaker (NULL for stage directions) |
| `char_name` | TEXT | Speaker name (denormalized for display) |
| `content` | TEXT | The actual text |
| `content_type` | TEXT | `speech`, `stage_direction`, `prologue`, `epilogue`, `song` |
| `word_count` | INTEGER | Words in this line |
| `oss_paragraph_id` | INTEGER | OSS paragraph ID (for import linkage) |
| `sonnet_number` | INTEGER | Sonnet number (sonnets only) |
| `stanza` | INTEGER | Stanza number (poems/sonnets) |

**Indexes**: `(work_id, edition_id)`, `(work_id, act, scene)`, `(work_id, edition_id, id)` (cursor), `(work_id, edition_id, act, scene, line_number)`

### `text_divisions`

Structural divisions (acts/scenes) per edition.

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER PK | Auto-increment |
| `work_id` | INTEGER FK → works | Which work |
| `edition_id` | INTEGER FK → editions | Which edition |
| `act` | INTEGER | Act number |
| `scene` | INTEGER | Scene number |
| `description` | TEXT | Scene description/location |
| `line_count` | INTEGER | Lines in this scene |

**Unique constraint**: `(work_id, edition_id, act, scene)`

### `lexicon_entries`

Schmidt's Shakespeare Lexicon — one row per headword. Contains ~20,000 entries.

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER PK | Auto-increment |
| `key` | TEXT UNIQUE | Normalized headword (for lookup) |
| `letter` | TEXT | First letter (for browsing/pagination) |
| `orthography` | TEXT | Display form with original spelling |
| `entry_type` | TEXT | `main` or `cross_ref` |
| `full_text` | TEXT | Full definition text (all senses combined) |
| `raw_xml` | TEXT | Original TEI XML (for re-parsing) |
| `source_file` | TEXT | Source XML filename |
| `created_at` | TIMESTAMP | When imported |

**Indexes**: `key`, `letter`, `(letter, id)` (cursor)

### `lexicon_senses`

Individual numbered senses within a lexicon entry.

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER PK | Auto-increment |
| `entry_id` | INTEGER FK → lexicon_entries | Parent entry |
| `sense_number` | INTEGER | 1, 2, 3... |
| `definition_text` | TEXT | Definition for this sense |

**Unique constraint**: `(entry_id, sense_number)`  
**Indexes**: `entry_id`

### `lexicon_citations`

Individual citations within lexicon senses — links to specific play/act/scene/line.

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER PK | Auto-increment |
| `entry_id` | INTEGER FK → lexicon_entries | Parent entry |
| `sense_id` | INTEGER FK → lexicon_senses | Parent sense (nullable) |
| `work_id` | INTEGER FK → works | Resolved work (nullable if unresolved) |
| `work_abbrev` | TEXT | Schmidt's abbreviation (e.g. `ham`, `tp`) |
| `perseus_ref` | TEXT | Perseus bibl `n=""` value |
| `act` | INTEGER | Act number |
| `scene` | INTEGER | Scene number |
| `line` | INTEGER | Line number |
| `quote_text` | TEXT | Quoted text from Schmidt |
| `display_text` | TEXT | Formatted display text |
| `raw_bibl` | TEXT | Raw XML bibl element |

**Indexes**: `entry_id`, `work_id`, `(work_abbrev, act, scene, line)`, `(entry_id, id)` (cursor)

### `citation_matches`

Resolved links from lexicon citations to actual text lines. Maps lexicon citations to `text_lines` rows in each edition. Generated by a 5-strategy matching cascade — see [docs/citation-resolution.md](../../docs/citation-resolution.md).

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER PK | Auto-increment |
| `citation_id` | INTEGER FK → lexicon_citations | Which citation |
| `text_line_id` | INTEGER FK → text_lines | Matched text line |
| `edition_id` | INTEGER FK → editions | Which edition the match is in |
| `match_type` | TEXT | `exact`, `fuzzy`, `positional` |
| `confidence` | REAL | 1.0 = exact quote, 0.9 = line match, 0.7 = fuzzy, 0.5 = guess |
| `matched_text` | TEXT | The text that was matched |
| `notes` | TEXT | Match details |

**Indexes**: `citation_id`, `text_line_id`, `(citation_id, id)` (cursor)

### `line_mappings`

Cross-edition line alignment for side-by-side comparison. Each row aligns one display position across two editions. Generated by Needleman-Wunsch sequence alignment — see [docs/line-alignment.md](../../docs/line-alignment.md).

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER PK | Auto-increment |
| `work_id` | INTEGER FK → works | Which work |
| `act` | INTEGER | Act number |
| `scene` | INTEGER | Scene number |
| `align_order` | INTEGER | Sequential position in comparison view |
| `edition_a_id` | INTEGER FK → editions | First edition |
| `edition_b_id` | INTEGER FK → editions | Second edition |
| `line_a_id` | INTEGER FK → text_lines | Line in edition A (nullable) |
| `line_b_id` | INTEGER FK → text_lines | Line in edition B (nullable) |
| `match_type` | TEXT | `aligned`, `modified`, `only_a`, `only_b` |
| `similarity` | REAL | Jaccard similarity score (0.0–1.0) |

**Unique constraint**: `(work_id, act, scene, align_order, edition_a_id, edition_b_id)`  
**Indexes**: `(work_id, act, scene)`, `(work_id, act, scene, id)` (cursor)

### `import_log`

Build tracking — records each phase of the import pipeline.

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER PK | Auto-increment |
| `phase` | TEXT | Pipeline phase name |
| `action` | TEXT | What happened |
| `details` | TEXT | Additional context |
| `count` | INTEGER | Row count affected |
| `duration_secs` | REAL | How long the phase took |
| `timestamp` | TIMESTAMP | When it happened |

---

## Full-Text Search (FTS5)

See [docs/fts-search.md](../../docs/fts-search.md) for detailed setup and query examples.

### `lexicon_fts`

Full-text search over lexicon entries. Uses Porter stemming for English-language search.

```sql
CREATE VIRTUAL TABLE lexicon_fts USING fts5(
    key, orthography, full_text,
    content='lexicon_entries',
    content_rowid='id',
    tokenize='porter unicode61'
);
```

### `text_fts`

Full-text search over all text lines across all editions.

```sql
CREATE VIRTUAL TABLE text_fts USING fts5(
    content, char_name,
    content='text_lines',
    content_rowid='id',
    tokenize='porter unicode61'
);
```

---

## Cursor Pagination

All primary tables support cursor-based pagination using their auto-increment `id` as the cursor. This is more efficient than OFFSET for large datasets and infinite scroll.

**Pattern:**
```sql
SELECT * FROM lexicon_entries
WHERE letter = ? AND id > :cursor
ORDER BY id ASC
LIMIT :limit;
```

Composite indexes `(filter_column, id)` are provided on all major tables for efficient filtered cursor queries.

---

## Key Queries

### Dictionary Lookup (Full-Text Search)
```sql
SELECT e.key, e.orthography, e.full_text
FROM lexicon_fts f
JOIN lexicon_entries e ON e.id = f.rowid
WHERE lexicon_fts MATCH 'abandon*'
ORDER BY rank;
```

### All Citations for a Lexicon Entry
```sql
SELECT s.sense_number, s.definition_text, c.work_abbrev, c.act, c.scene, c.line,
       c.quote_text, w.title
FROM lexicon_senses s
JOIN lexicon_citations c ON c.sense_id = s.id
LEFT JOIN works w ON w.id = c.work_id
WHERE s.entry_id = ?
ORDER BY s.sense_number, c.id;
```

### Retrieve Text for a Citation
```sql
-- Get passage around a citation (e.g. Hamlet Act 3, Scene 2, line 47 ± 5 lines)
SELECT tl.line_number, tl.content, tl.char_name, tl.content_type
FROM text_lines tl
WHERE tl.edition_id = ? AND tl.work_id = ? AND tl.act = 3 AND tl.scene = 2
  AND tl.line_number BETWEEN 42 AND 52
ORDER BY tl.line_number;
```

### Side-by-Side Edition Comparison
```sql
SELECT lm.align_order, lm.match_type, lm.similarity,
       a.content AS text_a, a.char_name AS speaker_a,
       b.content AS text_b, b.char_name AS speaker_b
FROM line_mappings lm
LEFT JOIN text_lines a ON a.id = lm.line_a_id
LEFT JOIN text_lines b ON b.id = lm.line_b_id
WHERE lm.work_id = ? AND lm.act = ? AND lm.scene = ?
ORDER BY lm.align_order;
```

### Full-Text Search Across All Editions
```sql
SELECT e.name AS edition, w.title, tl.act, tl.scene, tl.line_number, tl.content
FROM text_fts f
JOIN text_lines tl ON tl.id = f.rowid
JOIN editions e ON e.id = tl.edition_id
JOIN works w ON w.id = tl.work_id
WHERE text_fts MATCH '"to be or not to be"'
ORDER BY w.title, e.id, tl.act, tl.scene, tl.line_number;
```

### Get Full Scene
```sql
SELECT tl.line_number, tl.content, tl.content_type, tl.char_name
FROM text_lines tl
WHERE tl.edition_id = ? AND tl.work_id = ? AND tl.act = 1 AND tl.scene = 2
ORDER BY tl.line_number;
```

---

## Entity Relationship Summary

```
sources 1──∞ editions 1──∞ text_lines ∞──1 characters
   │                          │
   1                          │
   │                     text_divisions
attributions                  │
                              ∞
                        citation_matches
                              ∞
                              │
lexicon_entries 1──∞ lexicon_senses 1──∞ lexicon_citations
                                              │
                                              ∞
                                        line_mappings
```
