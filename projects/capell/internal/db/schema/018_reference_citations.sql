-- Parsed citations extracted from reference_entries.raw_text
-- One row per play/poem reference found in an entry's text block.
-- Primarily populated from Onions (1911), using Globe act/scene/line numbers.
--
-- work_abbrev holds the raw abbreviation as found in the source text (e.g. 'AYL.', '2H4').
-- work_id is resolved via the Onions→Schmidt abbreviation map + works.schmidt_abbrev.
CREATE TABLE IF NOT EXISTS reference_citations (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    entry_id    INTEGER NOT NULL REFERENCES reference_entries(id),
    source_id   INTEGER NOT NULL REFERENCES sources(id),
    work_id     INTEGER REFERENCES works(id),
    work_abbrev TEXT    NOT NULL,  -- raw abbreviation from source text
    act         INTEGER,           -- NULL for poems (Lucr., Ven.)
    scene       INTEGER,           -- NULL for poems; sonnet number for Sonn.
    line        INTEGER            -- Globe line number
);
CREATE INDEX IF NOT EXISTS idx_ref_cit_entry ON reference_citations(entry_id);
CREATE INDEX IF NOT EXISTS idx_ref_cit_work  ON reference_citations(work_id);
CREATE INDEX IF NOT EXISTS idx_ref_cit_loc   ON reference_citations(work_id, act, scene);
