-- Attributions: display rules for source credits
-- Tracks attribution requirements for ALL sources, whether
-- legally required or voluntary. Display rules define how
-- and where attribution should appear in consuming applications.
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
