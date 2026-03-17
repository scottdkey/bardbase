-- Works: all 43 Shakespeare works
CREATE TABLE IF NOT EXISTS works (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    oss_id TEXT UNIQUE,
    title TEXT NOT NULL,
    full_title TEXT,
    short_title TEXT,
    schmidt_abbrev TEXT,
    work_type TEXT,
    date_composed INTEGER,
    genre_type TEXT,
    total_words INTEGER,
    total_paragraphs INTEGER,
    source_text TEXT,
    folger_url TEXT,
    perseus_id TEXT,
    notes TEXT
);
CREATE INDEX IF NOT EXISTS idx_works_oss_id ON works(oss_id);
CREATE INDEX IF NOT EXISTS idx_works_schmidt ON works(schmidt_abbrev);
CREATE INDEX IF NOT EXISTS idx_works_type ON works(work_type);
