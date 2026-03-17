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
