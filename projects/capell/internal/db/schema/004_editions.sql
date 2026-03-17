-- Editions
CREATE TABLE IF NOT EXISTS editions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    short_code TEXT NOT NULL UNIQUE,
    source_id INTEGER REFERENCES sources(id),
    year INTEGER,
    editors TEXT,
    description TEXT,
    notes TEXT
);
