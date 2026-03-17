-- Full-text search: lexicon
CREATE VIRTUAL TABLE IF NOT EXISTS lexicon_fts USING fts5(
    key, orthography, full_text,
    content='lexicon_entries',
    content_rowid='id',
    tokenize='porter unicode61'
);

-- Full-text search: text
CREATE VIRTUAL TABLE IF NOT EXISTS text_fts USING fts5(
    content, char_name,
    content='text_lines',
    content_rowid='id',
    tokenize='porter unicode61'
);
