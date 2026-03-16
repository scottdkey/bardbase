// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Scott Key

package constants

// SchemaSQL is the complete DDL for the Shakespeare database.
//
// ## Cursor Pagination
//
// All primary tables support cursor-based pagination using their auto-increment `id`
// as the cursor. This is more efficient than OFFSET for large datasets and infinite scroll.
//
// Pattern:
//
//	SELECT * FROM lexicon_entries WHERE letter = ? AND id > :cursor ORDER BY id ASC LIMIT :limit
//	SELECT * FROM text_lines WHERE work_id = ? AND edition_id = ? AND id > :cursor ORDER BY id ASC LIMIT :limit
//
// Composite indexes (table_name, id) are provided for efficient filtered cursor queries.
const SchemaSQL = `
-- ============================================================
-- Sources: where data comes from
-- ============================================================
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

-- ============================================================
-- Works: all 43 Shakespeare works
-- ============================================================
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
CREATE INDEX IF NOT EXISTS idx_works_type ON works(work_type);

-- ============================================================
-- Characters
-- ============================================================
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

-- ============================================================
-- Editions
-- ============================================================
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

-- ============================================================
-- Attributions: display rules for source credits
-- Tracks attribution requirements for ALL sources, whether
-- legally required or voluntary. Display rules define how
-- and where attribution should appear in consuming applications.
-- ============================================================
CREATE TABLE IF NOT EXISTS attributions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_id INTEGER NOT NULL REFERENCES sources(id) UNIQUE,
    required BOOLEAN DEFAULT 0,
    attribution_text TEXT NOT NULL,
    attribution_html TEXT,
    display_format TEXT DEFAULT 'footer',
    display_context TEXT DEFAULT 'always',
    display_priority INTEGER DEFAULT 0,
    requires_link_back BOOLEAN DEFAULT 0,
    link_back_url TEXT,
    requires_license_notice BOOLEAN DEFAULT 0,
    license_notice_text TEXT,
    requires_author_credit BOOLEAN DEFAULT 0,
    author_credit_text TEXT,
    share_alike_required BOOLEAN DEFAULT 0,
    commercial_use_allowed BOOLEAN DEFAULT 1,
    modification_allowed BOOLEAN DEFAULT 1,
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Text lines: the actual play/poem text
-- line_number is a scene-relative sequential line number
-- used for consistent cross-edition referencing and
-- citation matching (Globe-style numbering).
-- ============================================================
CREATE TABLE IF NOT EXISTS text_lines (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    work_id INTEGER NOT NULL REFERENCES works(id),
    edition_id INTEGER NOT NULL REFERENCES editions(id),
    act INTEGER,
    scene INTEGER,
    paragraph_num INTEGER,
    line_number INTEGER,
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
CREATE INDEX IF NOT EXISTS idx_text_cursor ON text_lines(work_id, edition_id, id);
CREATE INDEX IF NOT EXISTS idx_text_line_number ON text_lines(work_id, edition_id, act, scene, line_number);

-- ============================================================
-- Structural divisions (acts/scenes)
-- ============================================================
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

-- ============================================================
-- Lexicon entries (Schmidt)
-- ============================================================
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
CREATE INDEX IF NOT EXISTS idx_lexicon_cursor ON lexicon_entries(letter, id);

-- ============================================================
-- Lexicon senses
-- ============================================================
CREATE TABLE IF NOT EXISTS lexicon_senses (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    entry_id INTEGER NOT NULL REFERENCES lexicon_entries(id),
    sense_number INTEGER NOT NULL,
    definition_text TEXT,
    UNIQUE(entry_id, sense_number)
);
CREATE INDEX IF NOT EXISTS idx_senses_entry ON lexicon_senses(entry_id);

-- ============================================================
-- Lexicon citations
-- ============================================================
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
CREATE INDEX IF NOT EXISTS idx_citations_cursor ON lexicon_citations(entry_id, id);

-- ============================================================
-- Citation-to-text resolved links
-- Maps lexicon citations to actual text_lines rows in each edition.
-- confidence: 1.0 = exact quote match, 0.9 = line number match,
--             0.7 = fuzzy text match, 0.5 = positional guess
-- ============================================================
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
CREATE INDEX IF NOT EXISTS idx_citation_matches_citation ON citation_matches(citation_id);
CREATE INDEX IF NOT EXISTS idx_citation_matches_line ON citation_matches(text_line_id);
CREATE INDEX IF NOT EXISTS idx_citation_matches_cursor ON citation_matches(citation_id, id);

-- ============================================================
-- Cross-edition line mappings (for side-by-side comparison)
-- Each row aligns one display position across two editions.
-- align_order is the sequential position in the comparison view.
-- match_type: 'aligned' (both present, similar), 'modified' (both present, different),
--             'only_a' (only in edition A), 'only_b' (only in edition B)
-- ============================================================
CREATE TABLE IF NOT EXISTS line_mappings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    work_id INTEGER NOT NULL REFERENCES works(id),
    act INTEGER NOT NULL,
    scene INTEGER NOT NULL,
    align_order INTEGER NOT NULL,
    edition_a_id INTEGER NOT NULL REFERENCES editions(id),
    edition_b_id INTEGER NOT NULL REFERENCES editions(id),
    line_a_id INTEGER REFERENCES text_lines(id),
    line_b_id INTEGER REFERENCES text_lines(id),
    match_type TEXT DEFAULT 'aligned',
    similarity REAL DEFAULT 0.0,
    UNIQUE(work_id, act, scene, align_order, edition_a_id, edition_b_id)
);
CREATE INDEX IF NOT EXISTS idx_line_mappings_scene ON line_mappings(work_id, act, scene);
CREATE INDEX IF NOT EXISTS idx_line_mappings_cursor ON line_mappings(work_id, act, scene, id);

-- ============================================================
-- Build tracking
-- ============================================================
CREATE TABLE IF NOT EXISTS import_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    phase TEXT NOT NULL,
    action TEXT NOT NULL,
    details TEXT,
    count INTEGER DEFAULT 0,
    duration_secs REAL,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Full-text search: lexicon
-- ============================================================
CREATE VIRTUAL TABLE IF NOT EXISTS lexicon_fts USING fts5(
    key, orthography, full_text,
    content='lexicon_entries',
    content_rowid='id',
    tokenize='porter unicode61'
);

-- ============================================================
-- Full-text search: text
-- ============================================================
CREATE VIRTUAL TABLE IF NOT EXISTS text_fts USING fts5(
    content, char_name,
    content='text_lines',
    content_rowid='id',
    tokenize='porter unicode61'
);
`
