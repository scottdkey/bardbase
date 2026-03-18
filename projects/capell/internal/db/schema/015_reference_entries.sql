-- Reference entries: headword-based reference works (glossaries, grammars, dictionaries)
-- Used by: Onions Shakespeare Glossary, Abbott Shakespearian Grammar, Henley-Farmer Slang
CREATE TABLE IF NOT EXISTS reference_entries (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    source_id  INTEGER NOT NULL REFERENCES sources(id),
    headword   TEXT    NOT NULL,
    letter     TEXT    NOT NULL,   -- first letter, for browsing
    raw_text   TEXT    NOT NULL,   -- full entry text as extracted (OCR as-is)
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_ref_entries_source   ON reference_entries(source_id);
CREATE INDEX IF NOT EXISTS idx_ref_entries_headword ON reference_entries(headword);
CREATE INDEX IF NOT EXISTS idx_ref_entries_cursor   ON reference_entries(source_id, id);
