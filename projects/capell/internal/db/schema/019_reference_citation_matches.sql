-- Links reference_citations to actual text_lines rows, per edition.
-- Parallel to citation_matches but for reference sources (Onions, Abbott, etc.)
-- rather than lexicon_citations (Schmidt).
--
-- Primary match target is the OSS edition (Globe line numbers), then propagated
-- to other editions via line_mappings.
CREATE TABLE IF NOT EXISTS reference_citation_matches (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    ref_citation_id INTEGER NOT NULL REFERENCES reference_citations(id),
    text_line_id    INTEGER NOT NULL REFERENCES text_lines(id),
    edition_id      INTEGER NOT NULL REFERENCES editions(id),
    match_type      TEXT    DEFAULT 'line_number',  -- 'line_number', 'propagated'
    confidence      REAL    DEFAULT 1.0,
    matched_text    TEXT
);
CREATE INDEX IF NOT EXISTS idx_ref_cit_match_cit  ON reference_citation_matches(ref_citation_id);
CREATE INDEX IF NOT EXISTS idx_ref_cit_match_line ON reference_citation_matches(text_line_id);
