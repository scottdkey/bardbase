package constants

// SchemaSQL is the complete DDL for the Shakespeare database.
const SchemaSQL = `
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

-- Works: all 43 Shakespeare works
CREATE TABLE IF NOT EXISTS works (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    oss_id TEXT UNIQUE,
    title TEXT NOT NULL,
    full_title TEXT,
    short_title TEXT,
    schmidt_abbrev TEXT,
    work_type TEXT,
    date_composed INTEGER,
    genre_type TEXT,
    total_words INTEGER,
    total_paragraphs INTEGER,
    source_text TEXT,
    folger_url TEXT,
    perseus_id TEXT,
    notes TEXT
);
CREATE INDEX IF NOT EXISTS idx_works_oss_id ON works(oss_id);
CREATE INDEX IF NOT EXISTS idx_works_schmidt ON works(schmidt_abbrev);

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

-- Text lines: the actual play/poem text
CREATE TABLE IF NOT EXISTS text_lines (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    work_id INTEGER NOT NULL REFERENCES works(id),
    edition_id INTEGER NOT NULL REFERENCES editions(id),
    act INTEGER,
    scene INTEGER,
    paragraph_num INTEGER,
    character_id INTEGER REFERENCES characters(id),
    char_name TEXT,
    content TEXT NOT NULL,
    content_type TEXT DEFAULT 'speech',
    word_count INTEGER DEFAULT 0,
    oss_paragraph_id INTEGER,
    sonnet_number INTEGER,
    stanza INTEGER
);
CREATE INDEX IF NOT EXISTS idx_text_work_edition ON text_lines(work_id, edition_id);
CREATE INDEX IF NOT EXISTS idx_text_location ON text_lines(work_id, act, scene);

-- Structural divisions (acts/scenes)
CREATE TABLE IF NOT EXISTS text_divisions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    work_id INTEGER NOT NULL REFERENCES works(id),
    edition_id INTEGER NOT NULL REFERENCES editions(id),
    act INTEGER NOT NULL,
    scene INTEGER NOT NULL,
    description TEXT,
    line_count INTEGER DEFAULT 0,
    UNIQUE(work_id, edition_id, act, scene)
);

-- Lexicon entries (Schmidt)
CREATE TABLE IF NOT EXISTS lexicon_entries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key TEXT NOT NULL UNIQUE,
    letter TEXT NOT NULL,
    orthography TEXT,
    entry_type TEXT DEFAULT 'main',
    full_text TEXT,
    raw_xml TEXT,
    source_file TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_lexicon_key ON lexicon_entries(key);
CREATE INDEX IF NOT EXISTS idx_lexicon_letter ON lexicon_entries(letter);

-- Lexicon senses
CREATE TABLE IF NOT EXISTS lexicon_senses (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    entry_id INTEGER NOT NULL REFERENCES lexicon_entries(id),
    sense_number INTEGER NOT NULL,
    definition_text TEXT,
    UNIQUE(entry_id, sense_number)
);
CREATE INDEX IF NOT EXISTS idx_senses_entry ON lexicon_senses(entry_id);

-- Lexicon citations
CREATE TABLE IF NOT EXISTS lexicon_citations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    entry_id INTEGER NOT NULL REFERENCES lexicon_entries(id),
    sense_id INTEGER REFERENCES lexicon_senses(id),
    work_id INTEGER REFERENCES works(id),
    work_abbrev TEXT,
    perseus_ref TEXT,
    act INTEGER,
    scene INTEGER,
    line INTEGER,
    quote_text TEXT,
    display_text TEXT,
    raw_bibl TEXT
);
CREATE INDEX IF NOT EXISTS idx_citations_entry ON lexicon_citations(entry_id);
CREATE INDEX IF NOT EXISTS idx_citations_work ON lexicon_citations(work_id);
CREATE INDEX IF NOT EXISTS idx_citations_location ON lexicon_citations(work_abbrev, act, scene, line);

-- Citation-to-text resolved links
CREATE TABLE IF NOT EXISTS citation_matches (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    citation_id INTEGER NOT NULL REFERENCES lexicon_citations(id),
    text_line_id INTEGER NOT NULL REFERENCES text_lines(id),
    edition_id INTEGER NOT NULL REFERENCES editions(id),
    match_type TEXT DEFAULT 'exact',
    confidence REAL DEFAULT 1.0,
    matched_text TEXT,
    notes TEXT
);

-- Cross-edition line concordance
CREATE TABLE IF NOT EXISTS line_mappings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    work_id INTEGER NOT NULL REFERENCES works(id),
    act INTEGER,
    scene INTEGER,
    globe_line INTEGER,
    f1_tln INTEGER,
    f1_line INTEGER,
    notes TEXT
);

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
`
