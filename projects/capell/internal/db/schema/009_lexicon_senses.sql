-- Lexicon senses
CREATE TABLE IF NOT EXISTS lexicon_senses (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    entry_id INTEGER NOT NULL REFERENCES lexicon_entries(id),
    sense_number INTEGER NOT NULL,
    definition_text TEXT,
    UNIQUE(entry_id, sense_number)
);
CREATE INDEX IF NOT EXISTS idx_senses_entry ON lexicon_senses(entry_id);
