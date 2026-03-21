-- Lexicon entries (Schmidt)
-- base_key: the headword with trailing sense numbers stripped (e.g., "A" for A1-A7).
--           Used to group related entries in the UI.
-- sense_group: the trailing number from the key (1 for A1, 2 for A2, etc.), or NULL.
CREATE TABLE IF NOT EXISTS lexicon_entries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key TEXT NOT NULL UNIQUE,
    base_key TEXT NOT NULL,
    sense_group INTEGER,
    letter TEXT NOT NULL,
    orthography TEXT,
    entry_type TEXT DEFAULT 'main',
    full_text TEXT,
    source_file TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_lexicon_key ON lexicon_entries(key);
CREATE INDEX IF NOT EXISTS idx_lexicon_base_key ON lexicon_entries(base_key);
CREATE INDEX IF NOT EXISTS idx_lexicon_letter ON lexicon_entries(letter);
CREATE INDEX IF NOT EXISTS idx_lexicon_cursor ON lexicon_entries(letter, id);
