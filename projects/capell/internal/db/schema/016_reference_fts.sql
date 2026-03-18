-- Full-text search: reference entries (Onions, Abbott, Henley-Farmer, etc.)
CREATE VIRTUAL TABLE IF NOT EXISTS reference_fts USING fts5(
    headword, raw_text,
    content='reference_entries',
    content_rowid='id',
    tokenize='porter unicode61'
);
