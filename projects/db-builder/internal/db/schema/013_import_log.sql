-- Build tracking
CREATE TABLE IF NOT EXISTS import_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    phase TEXT NOT NULL,
    action TEXT NOT NULL,
    details TEXT,
    count INTEGER DEFAULT 0,
    duration_secs REAL,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
