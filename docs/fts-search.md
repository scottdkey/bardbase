# Full-Text Search (FTS5)

**Source file**: `projects/capell/internal/importer/fts.go`
**Output tables**: `lexicon_fts`, `text_fts`

The database ships two FTS5 virtual tables that provide fast, ranked full-text search over lexicon definitions and play text respectively.

## FTS5 Configuration

Both tables use the same tokenizer configuration:

```sql
tokenize='porter unicode61'
```

- **`unicode61`** — Unicode-aware tokenizer that handles accented characters, ligatures, and non-ASCII punctuation correctly. Splits on whitespace and punctuation boundaries.
- **`porter`** — Applies the Porter stemming algorithm on top of `unicode61`. This reduces words to their root form so that searching `abandon` also matches `abandoned`, `abandoning`, `abandonment`, etc.

Both tables are **content tables** (`content=''`), meaning FTS5 stores only the index and defers to the underlying content table for the actual text. This keeps the database size small.

## `lexicon_fts`

Indexes Schmidt's lexicon for dictionary-style search.

```sql
CREATE VIRTUAL TABLE lexicon_fts USING fts5(
    key,          -- normalized headword (e.g. "abandon")
    orthography,  -- display form with original spelling
    full_text,    -- complete definition text (all senses combined)
    content='lexicon_entries',
    content_rowid='id',
    tokenize='porter unicode61'
);
```

Searching `key` and `orthography` finds headwords; searching `full_text` finds words used in definitions, which is useful for thematic browsing (e.g. find all entries that mention "honour").

## `text_fts`

Indexes all lines of text across all editions.

```sql
CREATE VIRTUAL TABLE text_fts USING fts5(
    content,    -- line text
    char_name,  -- speaker name
    content='text_lines',
    content_rowid='id',
    tokenize='porter unicode61'
);
```

Because every edition is indexed, a phrase search will return matches from `globe_moby`, `se_modern`, `perseus_globe`, and `first_folio`. Filter by `edition_id` or `work_id` via a JOIN to narrow results.

## Population

FTS tables are populated in a single pass after all content is imported:

```sql
INSERT INTO lexicon_fts(lexicon_fts) VALUES ('rebuild');
INSERT INTO text_fts(text_fts)      VALUES ('rebuild');
```

The `rebuild` command re-reads the content tables and reindexes everything. This is faster than row-by-row insertion and ensures the index is consistent with the content tables.

## Query Examples

### Headword lookup with stemming
```sql
SELECT e.key, e.orthography, e.full_text
FROM lexicon_fts f
JOIN lexicon_entries e ON e.id = f.rowid
WHERE lexicon_fts MATCH 'honour*'
ORDER BY rank;
```
Matches `honour`, `honourable`, `honourably`, `dishonour`, etc.

### Phrase search in play text
```sql
SELECT w.title, tl.act, tl.scene, tl.line_number,
       tl.char_name, tl.content
FROM text_fts f
JOIN text_lines tl ON tl.id = f.rowid
JOIN works w ON w.id = tl.work_id
WHERE text_fts MATCH '"to be or not to be"'
ORDER BY w.title, tl.act, tl.scene, tl.line_number;
```

### Search by speaker name
```sql
SELECT w.title, tl.act, tl.scene, tl.content
FROM text_fts f
JOIN text_lines tl ON tl.id = f.rowid
JOIN works w ON w.id = tl.work_id
WHERE text_fts MATCH 'char_name:Hamlet AND "my father"'
ORDER BY rank;
```

### Limit to one edition
```sql
SELECT tl.act, tl.scene, tl.line_number, tl.content
FROM text_fts f
JOIN text_lines tl ON tl.id = f.rowid
WHERE text_fts MATCH 'wherefore art thou'
  AND tl.edition_id = (SELECT id FROM editions WHERE short_code = 'first_folio')
ORDER BY rank;
```

### BM25 ranking
FTS5's `rank` column uses the BM25 algorithm. Lower rank values indicate better matches. Always `ORDER BY rank` when result relevance matters.

## Porter Stemming Notes

The Porter stemmer is English-specific and is appropriate for Shakespeare's language, which is Early Modern English. It handles most common inflections correctly. However:

- Archaic forms like `dost`, `hath`, `wouldst` are not stemmed to their modern equivalents — search for the archaic form directly.
- The First Folio's original spelling (`honour` vs `honor`, `musick` vs `music`) is indexed as-is. Cross-spelling searches require wildcards: `MATCH 'musick OR music'`.
