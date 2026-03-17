-- Characters
CREATE TABLE IF NOT EXISTS characters (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    char_id TEXT UNIQUE,
    name TEXT NOT NULL,
    abbrev TEXT,
    work_id INTEGER REFERENCES works(id),
    oss_work_id TEXT,
    description TEXT,
    speech_count INTEGER
);
CREATE INDEX IF NOT EXISTS idx_characters_work ON characters(work_id);
