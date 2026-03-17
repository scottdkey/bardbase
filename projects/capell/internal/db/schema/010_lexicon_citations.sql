-- Lexicon citations
CREATE TABLE IF NOT EXISTS lexicon_citations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    entry_id INTEGER NOT NULL REFERENCES lexicon_entries(id),
    sense_id INTEGER REFERENCES lexicon_senses(id),
    work_id INTEGER REFERENCES works(id),
    work_abbrev TEXT,
    perseus_ref TEXT,
    act INTEGER,
    scene INTEGER,
    line INTEGER,
    quote_text TEXT,
    display_text TEXT,
    raw_bibl TEXT
);
CREATE INDEX IF NOT EXISTS idx_citations_entry ON lexicon_citations(entry_id);
CREATE INDEX IF NOT EXISTS idx_citations_work ON lexicon_citations(work_id);
CREATE INDEX IF NOT EXISTS idx_citations_location ON lexicon_citations(work_abbrev, act, scene, line);
CREATE INDEX IF NOT EXISTS idx_citations_cursor ON lexicon_citations(entry_id, id);
