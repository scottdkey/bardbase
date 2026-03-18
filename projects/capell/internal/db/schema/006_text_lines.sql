-- Text lines: the actual play/poem text
-- line_number is a scene-relative sequential line number
-- used for consistent cross-edition referencing and
-- citation matching (Globe-style numbering).
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
    stanza INTEGER,
    line_type  TEXT,   -- 'verse', 'prose', or NULL (unknown)
    -- Stage direction metadata (Folger TEIsimple only; NULL in all other editions)
    stage_type TEXT,   -- value of <stage type="…"> attr (e.g. 'entrance', 'exit')
    stage_who  TEXT    -- space-separated character IDs from <stage who="…"> attr
);
CREATE INDEX IF NOT EXISTS idx_text_work_edition ON text_lines(work_id, edition_id);
CREATE INDEX IF NOT EXISTS idx_text_location ON text_lines(work_id, act, scene);
CREATE INDEX IF NOT EXISTS idx_text_cursor ON text_lines(work_id, edition_id, id);
CREATE INDEX IF NOT EXISTS idx_text_line_number ON text_lines(work_id, edition_id, act, scene, line_number);
