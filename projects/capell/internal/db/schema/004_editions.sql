-- Editions
CREATE TABLE IF NOT EXISTS editions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    short_code TEXT NOT NULL UNIQUE,
    source_id INTEGER REFERENCES sources(id),
    year INTEGER,
    editors TEXT,
    description TEXT,
    notes TEXT,
    source_key TEXT,    -- e.g. 'folger', 'ise', 'eebo' — used by --exclude flag
    license_tier TEXT   -- 'cc0', 'cc-by-sa', 'cc-by-nc'
);
