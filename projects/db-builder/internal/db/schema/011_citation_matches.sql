-- Citation-to-text resolved links
-- Maps lexicon citations to actual text_lines rows in each edition.
-- confidence: 1.0 = exact quote match, 0.9 = line number match,
--             0.7 = fuzzy text match, 0.5 = positional guess
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
