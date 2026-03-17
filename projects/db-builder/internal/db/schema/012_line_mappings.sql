-- Cross-edition line mappings (for side-by-side comparison)
-- Each row aligns one display position across two editions.
-- align_order is the sequential position in the comparison view.
-- match_type: 'aligned' (both present, similar), 'modified' (both present, different),
--             'only_a' (only in edition A), 'only_b' (only in edition B)
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
