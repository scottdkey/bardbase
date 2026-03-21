-- Lexicon senses
-- sense_number: top-level sense (1, 2, 3, ...)
-- sub_sense: lettered sub-sense within a top-level sense ("a", "b", "c", ...) or NULL
CREATE TABLE IF NOT EXISTS lexicon_senses (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    entry_id INTEGER NOT NULL REFERENCES lexicon_entries(id),
    sense_number INTEGER NOT NULL,
    sub_sense TEXT,
    definition_text TEXT
);
CREATE INDEX IF NOT EXISTS idx_senses_entry ON lexicon_senses(entry_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_senses_unique ON lexicon_senses(entry_id, sense_number, sub_sense);
