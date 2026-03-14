# Shakespeare Database Schema

SQLite database designed for a Shakespeare dictionary app with multi-edition text comparison, lexicon lookup, and full-text search.

---

## Design Principles

1. **Every line belongs to an edition** — no ambiguity about which version you're reading
2. **Citations resolve to actual text** — lexicon entries link to retrievable passages
3. **Side-by-side comparison** — any two editions of the same work, aligned by act/scene/line
4. **Line mapping across editions** — Globe line 40 in one edition might be line 42 in another
5. **Attribution is first-class** — every source tracked with license and credit requirements
6. **Single SQLite file** — ships with the app, no server needed

---

## Tables

### `source_attributions`
Tracks every data source with its license and required attribution text. Drives the app's credits page.

```sql
CREATE TABLE source_attributions (
    source_id       TEXT PRIMARY KEY,       -- e.g. 'oss_moby', 'perseus', 'eebo_f1'
    source_name     TEXT NOT NULL,          -- display name
    source_url      TEXT,                   -- homepage URL
    license         TEXT NOT NULL,          -- 'PD', 'CC-BY-SA-3.0', 'CC0', etc.
    attribution_text TEXT,                  -- exact text to display in app credits
    attribution_required INTEGER DEFAULT 0, -- 1 if legally required
    notes           TEXT                    -- additional context
);
```

### `works`
Canonical list of Shakespeare works (plays, poems, sonnets). One row per work regardless of how many editions exist.

```sql
CREATE TABLE works (
    work_id         TEXT PRIMARY KEY,       -- e.g. 'hamlet', 'tempest', 'sonnet_18'
    title           TEXT NOT NULL,          -- 'Hamlet'
    long_title      TEXT,                   -- 'The Tragedy of Hamlet, Prince of Denmark'
    short_title     TEXT,                   -- 'Ham.'
    genre           TEXT,                   -- 'tragedy', 'comedy', 'history', 'poem', 'sonnet'
    date_composed   INTEGER,               -- approximate year
    schmidt_abbrev  TEXT                    -- Schmidt's abbreviation: 'ham', 'tmp', etc.
);
```

### `editions`
A specific version/text of Shakespeare's works from a specific source.

```sql
CREATE TABLE editions (
    edition_id      TEXT PRIMARY KEY,       -- e.g. 'globe_moby', 'f1_eebo', 'q2_hamlet'
    source_id       TEXT NOT NULL REFERENCES source_attributions(source_id),
    edition_name    TEXT NOT NULL,          -- 'Globe/Moby Modern Text'
    edition_type    TEXT,                   -- 'modern', 'f1_diplomatic', 'quarto', 'globe'
    year_published  INTEGER,               -- year of the edition (1623 for F1, etc.)
    spelling        TEXT DEFAULT 'modern',  -- 'modern' or 'original'
    notes           TEXT
);
```

### `edition_works`
Which works are available in which editions (many-to-many).

```sql
CREATE TABLE edition_works (
    edition_id      TEXT NOT NULL REFERENCES editions(edition_id),
    work_id         TEXT NOT NULL REFERENCES works(work_id),
    PRIMARY KEY (edition_id, work_id)
);
```

### `characters`
Dramatis personae per work.

```sql
CREATE TABLE characters (
    character_id    TEXT PRIMARY KEY,       -- e.g. 'hamlet_hamlet', 'hamlet_ophelia'
    work_id         TEXT NOT NULL REFERENCES works(work_id),
    character_name  TEXT NOT NULL,          -- 'Hamlet'
    abbreviation    TEXT,                   -- 'Ham.'
    description     TEXT,                   -- 'Prince of Denmark'
    speech_count    INTEGER
);
```

### `text_sections`
Act/scene structure per edition per work.

```sql
CREATE TABLE text_sections (
    section_id      INTEGER PRIMARY KEY AUTOINCREMENT,
    edition_id      TEXT NOT NULL REFERENCES editions(edition_id),
    work_id         TEXT NOT NULL REFERENCES works(work_id),
    act             INTEGER,               -- NULL for poems/sonnets
    scene           INTEGER,               -- NULL for poems/sonnets
    section_title   TEXT,                   -- 'Elsinore. A platform before the castle.'
    sort_order      INTEGER
);
```

### `text_lines`
Every line of text in every edition. This is the big table.

```sql
CREATE TABLE text_lines (
    line_id         INTEGER PRIMARY KEY AUTOINCREMENT,
    section_id      INTEGER NOT NULL REFERENCES text_sections(section_id),
    edition_id      TEXT NOT NULL REFERENCES editions(edition_id),
    work_id         TEXT NOT NULL REFERENCES works(work_id),
    act             INTEGER,
    scene           INTEGER,
    line_number     INTEGER,               -- line number within the scene (edition-specific)
    globe_line      INTEGER,               -- Globe edition line number (for Schmidt references)
    tln             INTEGER,               -- Through Line Number (continuous numbering)
    character_id    TEXT REFERENCES characters(character_id),
    line_text       TEXT NOT NULL,          -- the actual text
    line_type       TEXT DEFAULT 'dialogue', -- 'dialogue', 'stage_direction', 'prologue', 'epilogue', 'song'
    is_prose        INTEGER DEFAULT 0,     -- 1 if prose, 0 if verse
    original_spelling TEXT                  -- original spelling variant (if different from line_text)
);

CREATE INDEX idx_lines_edition_work ON text_lines(edition_id, work_id);
CREATE INDEX idx_lines_act_scene ON text_lines(work_id, act, scene);
CREATE INDEX idx_lines_globe ON text_lines(work_id, globe_line);
CREATE INDEX idx_lines_tln ON text_lines(work_id, tln);
```

### `spelling_normalizations`
Maps original spelling (F1/Quartos) to modern equivalents for cross-edition search.

```sql
CREATE TABLE spelling_normalizations (
    original        TEXT NOT NULL,          -- 'heauie'
    modern          TEXT NOT NULL,          -- 'heavy'
    source          TEXT,                   -- where this mapping came from
    PRIMARY KEY (original, modern)
);
```

### `line_mappings`
Aligns lines across editions for side-by-side comparison.

```sql
CREATE TABLE line_mappings (
    mapping_id      INTEGER PRIMARY KEY AUTOINCREMENT,
    work_id         TEXT NOT NULL REFERENCES works(work_id),
    act             INTEGER,
    scene           INTEGER,
    edition_a       TEXT NOT NULL REFERENCES editions(edition_id),
    line_a          INTEGER,               -- line number in edition A
    edition_b       TEXT NOT NULL REFERENCES editions(edition_id),
    line_b          INTEGER,               -- corresponding line in edition B
    confidence      REAL DEFAULT 1.0,      -- 0.0-1.0, how sure the mapping is
    notes           TEXT                    -- e.g. 'line only in Q2', 'F1 splits into two lines'
);

CREATE INDEX idx_mapping_lookup ON line_mappings(work_id, act, scene, edition_a, edition_b);
```

### `folger_references`
URLs to Folger Shakespeare Library pages. Reference only — no content stored.

```sql
CREATE TABLE folger_references (
    work_id         TEXT NOT NULL REFERENCES works(work_id),
    act             INTEGER,
    scene           INTEGER,
    url             TEXT NOT NULL,
    page_title      TEXT,
    PRIMARY KEY (work_id, act, scene)
);
```

### `lexicon_entries`
Schmidt's Shakespeare Lexicon — one row per headword.

```sql
CREATE TABLE lexicon_entries (
    entry_id        INTEGER PRIMARY KEY AUTOINCREMENT,
    key             TEXT NOT NULL UNIQUE,   -- 'Abandon' (normalized headword)
    headword        TEXT NOT NULL,          -- display form with punctuation
    entry_type      TEXT DEFAULT 'main',    -- 'main' or 'cross_ref'
    raw_xml         TEXT,                   -- original TEI XML for re-parsing
    definition_text TEXT,                   -- plain text definition (all senses combined)
    source_id       TEXT DEFAULT 'perseus' REFERENCES source_attributions(source_id)
);

CREATE INDEX idx_lexicon_key ON lexicon_entries(key);
```

### `lexicon_senses`
Individual numbered senses within a lexicon entry.

```sql
CREATE TABLE lexicon_senses (
    sense_id        INTEGER PRIMARY KEY AUTOINCREMENT,
    entry_id        INTEGER NOT NULL REFERENCES lexicon_entries(entry_id),
    sense_number    INTEGER,               -- 1, 2, 3... (NULL if entry has only one sense)
    definition      TEXT NOT NULL,          -- the definition text for this sense
    sort_order      INTEGER
);
```

### `lexicon_citations`
Individual citations within lexicon senses — links to specific play/act/scene/line.

```sql
CREATE TABLE lexicon_citations (
    citation_id     INTEGER PRIMARY KEY AUTOINCREMENT,
    sense_id        INTEGER NOT NULL REFERENCES lexicon_senses(sense_id),
    entry_id        INTEGER NOT NULL REFERENCES lexicon_entries(entry_id),
    work_id         TEXT REFERENCES works(work_id),
    raw_ref         TEXT NOT NULL,          -- original reference text: 'Ham. III, 2, 47'
    perseus_ref     TEXT,                   -- Perseus bibl n="" value: 'shak. ham 3.2'
    act             INTEGER,
    scene           INTEGER,
    line            INTEGER,
    quote_text      TEXT,                   -- quoted text from Schmidt (if present)
    sort_order      INTEGER
);

CREATE INDEX idx_citations_work ON lexicon_citations(work_id);
CREATE INDEX idx_citations_entry ON lexicon_citations(entry_id);
CREATE INDEX idx_citations_location ON lexicon_citations(work_id, act, scene, line);
```

### `lexicon_reference_resolutions`
Tracks whether each citation actually resolves to a real text line in each edition.

```sql
CREATE TABLE lexicon_reference_resolutions (
    resolution_id   INTEGER PRIMARY KEY AUTOINCREMENT,
    citation_id     INTEGER NOT NULL REFERENCES lexicon_citations(citation_id),
    edition_id      TEXT NOT NULL REFERENCES editions(edition_id),
    line_id         INTEGER REFERENCES text_lines(line_id),
    resolved        INTEGER DEFAULT 0,     -- 1 if matched, 0 if not
    confidence      REAL DEFAULT 0.0,      -- 0.0-1.0
    corrected_line  INTEGER,               -- if the line number needed adjustment
    notes           TEXT,
    UNIQUE(citation_id, edition_id)
);
```

### Full-Text Search (FTS5)

```sql
CREATE VIRTUAL TABLE lexicon_fts USING fts5(
    key,
    definition_text,
    content=lexicon_entries,
    content_rowid=entry_id,
    tokenize='porter unicode61'
);

CREATE VIRTUAL TABLE text_fts USING fts5(
    line_text,
    content=text_lines,
    content_rowid=line_id,
    tokenize='porter unicode61'
);
```

---

## Key Queries the App Needs

### 1. Dictionary Lookup (Full-Text Search)
```sql
-- User searches "abandoned"
SELECT e.key, e.headword, e.definition_text
FROM lexicon_fts f
JOIN lexicon_entries e ON e.entry_id = f.rowid
WHERE lexicon_fts MATCH 'abandon*'
ORDER BY rank;
```

### 2. Get All Citations for a Lexicon Entry
```sql
SELECT s.sense_number, s.definition, c.raw_ref, c.work_id, c.act, c.scene, c.line, c.quote_text, w.title
FROM lexicon_senses s
JOIN lexicon_citations c ON c.sense_id = s.sense_id
LEFT JOIN works w ON w.work_id = c.work_id
WHERE s.entry_id = ?
ORDER BY s.sort_order, c.sort_order;
```

### 3. Retrieve the Actual Text for a Citation
```sql
-- Get the passage around a citation (e.g., Hamlet Act 3, Scene 2, line 47 ± 5 lines)
SELECT tl.line_number, tl.line_text, tl.character_id, c.character_name
FROM text_lines tl
LEFT JOIN characters c ON c.character_id = tl.character_id
WHERE tl.edition_id = ? AND tl.work_id = 'hamlet' AND tl.act = 3 AND tl.scene = 2
  AND tl.line_number BETWEEN 42 AND 52
ORDER BY tl.line_number;
```

### 4. Side-by-Side Edition Comparison
```sql
-- Compare Globe and First Folio for Hamlet Act 3, Scene 1
SELECT
    g.line_number AS globe_line,
    g.line_text AS globe_text,
    f.line_number AS f1_line,
    f.line_text AS f1_text
FROM text_lines g
LEFT JOIN line_mappings m ON m.work_id = g.work_id AND m.act = g.act AND m.scene = g.scene
    AND m.edition_a = g.edition_id AND m.line_a = g.line_number AND m.edition_b = 'f1_eebo'
LEFT JOIN text_lines f ON f.edition_id = 'f1_eebo' AND f.work_id = g.work_id
    AND f.act = g.act AND f.scene = g.scene AND f.line_number = m.line_b
WHERE g.edition_id = 'globe_moby' AND g.work_id = 'hamlet' AND g.act = 3 AND g.scene = 1
ORDER BY g.line_number;
```

### 5. Full-Text Search Across All Editions
```sql
-- Search for "to be or not to be" across all editions
SELECT e.edition_name, w.title, tl.act, tl.scene, tl.line_number, tl.line_text
FROM text_fts f
JOIN text_lines tl ON tl.line_id = f.rowid
JOIN editions e ON e.edition_id = tl.edition_id
JOIN works w ON w.work_id = tl.work_id
WHERE text_fts MATCH '"to be or not to be"'
ORDER BY w.title, e.edition_id, tl.act, tl.scene, tl.line_number;
```

### 6. Validate Citation Resolution
```sql
-- Check which citations in the lexicon haven't been resolved for a given edition
SELECT le.key, lc.raw_ref, lc.work_id, lc.act, lc.scene, lc.line
FROM lexicon_citations lc
JOIN lexicon_entries le ON le.entry_id = lc.entry_id
LEFT JOIN lexicon_reference_resolutions r ON r.citation_id = lc.citation_id AND r.edition_id = ?
WHERE r.resolved IS NULL OR r.resolved = 0
ORDER BY lc.work_id, lc.act, lc.scene, lc.line;
```

### 7. Get Full Act or Scene
```sql
-- Get entire Act 1, Scene 2 of Hamlet in the Globe edition
SELECT tl.line_number, tl.line_text, tl.line_type, c.character_name
FROM text_lines tl
LEFT JOIN characters c ON c.character_id = tl.character_id
WHERE tl.edition_id = 'globe_moby' AND tl.work_id = 'hamlet' AND tl.act = 1 AND tl.scene = 2
ORDER BY tl.line_number;

-- Get entire Act 1 (all scenes)
SELECT tl.scene, tl.line_number, tl.line_text, tl.line_type, c.character_name
FROM text_lines tl
LEFT JOIN characters c ON c.character_id = tl.character_id
WHERE tl.edition_id = 'globe_moby' AND tl.work_id = 'hamlet' AND tl.act = 1
ORDER BY tl.scene, tl.line_number;
```

---

## Schmidt Abbreviation → Work ID Mapping

| Schmidt | Work ID | Title |
|---------|---------|-------|
| tp / tmp | tempest | The Tempest |
| gent | twogents | Two Gentlemen of Verona |
| wiv | merrywives | Merry Wives of Windsor |
| meas | measure | Measure for Measure |
| err | comedyerrors | Comedy of Errors |
| ado | muchado | Much Ado about Nothing |
| lll | — | Love's Labour's Lost |
| mid | — | A Midsummer Night's Dream |
| merch | merchantvenice | Merchant of Venice |
| ayl / as | asyoulikeit | As You Like It |
| shr | tamingshrew | Taming of the Shrew |
| aww / all's | allswell | All's Well That Ends Well |
| tw / tn | 12night | Twelfth Night |
| wt / wint | — | The Winter's Tale |
| john | kingjohn | King John |
| r2 | richard2 | Richard II |
| h4a / h4 1 | henry4p1 | Henry IV Part 1 |
| h4b / h4 2 | henry4p2 | Henry IV Part 2 |
| h5 | henry5 | Henry V |
| h6a / h6 1 | henry6p1 | Henry VI Part 1 |
| h6b / h6 2 | henry6p2 | Henry VI Part 2 |
| h6c / h6 3 | henry6p3 | Henry VI Part 3 |
| r3 | richard3 | Richard III |
| h8 | henry8 | Henry VIII |
| tro / troil | troilus | Troilus and Cressida |
| cor | coriolanus | Coriolanus |
| tit | titus | Titus Andronicus |
| rom | romeojuliet | Romeo and Juliet |
| tim | timonathens | Timon of Athens |
| caes | juliuscaesar | Julius Caesar |
| mac / mcb | macbeth | Macbeth |
| ham | hamlet | Hamlet |
| lr / lear | kinglear | King Lear |
| oth | othello | Othello |
| ant | antonycleo | Antony and Cleopatra |
| cym / cymb | cymbeline | Cymbeline |
| per | pericles | Pericles |
| sonn | sonnets | Sonnets |
| ven | venusadonis | Venus and Adonis |
| lucr | rapelucrece | Rape of Lucrece |
| pilgr | passionatepilgrim | Passionate Pilgrim |
| phoen | phoenixturtle | Phoenix and the Turtle |
| lc / compl | — | A Lover's Complaint |

**Note**: Entries marked `—` are works that exist in Schmidt but don't have matching IDs in the OSS data. These need to be added to the `works` table manually.
