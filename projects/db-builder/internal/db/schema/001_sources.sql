-- Sources: where data comes from
CREATE TABLE IF NOT EXISTS sources (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    short_code TEXT NOT NULL UNIQUE,
    url TEXT,
    license TEXT,
    license_url TEXT,
    attribution_text TEXT,
    attribution_required BOOLEAN DEFAULT 0,
    notes TEXT,
    imported_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
