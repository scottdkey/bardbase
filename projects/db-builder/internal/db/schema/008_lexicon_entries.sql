-- Lexicon entries (Schmidt)
CREATE TABLE IF NOT EXISTS lexicon_entries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key TEXT NOT NULL UNIQUE,
    letter TEXT NOT NULL,
    orthography TEXT,
    entry_type TEXT DEFAULT 'main',
    full_text TEXT,
    raw_xml TEXT,
    source_file TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_lexicon_key ON lexicon_entries(key);
CREATE INDEX IF NOT EXISTS idx_lexicon_letter ON lexicon_entries(letter);
CREATE INDEX IF NOT EXISTS idx_lexicon_cursor ON lexicon_entries(letter, id);
