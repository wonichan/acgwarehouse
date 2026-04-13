DROP INDEX IF EXISTS idx_tags_parent_id;

PRAGMA foreign_keys = OFF;

ALTER TABLE tags RENAME TO tags_old;

DROP INDEX IF EXISTS idx_tags_slug;
DROP INDEX IF EXISTS idx_tags_category;
DROP INDEX IF EXISTS idx_tags_parent_id;

CREATE TABLE tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    preferred_label TEXT UNIQUE NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    primary_category TEXT,
    review_state TEXT DEFAULT 'pending',
    trust_score REAL DEFAULT 0.0,
    usage_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_tags_slug ON tags(slug);
CREATE INDEX idx_tags_category ON tags(primary_category);

INSERT INTO tags (id, preferred_label, slug, primary_category, review_state, trust_score, usage_count, created_at)
SELECT id, preferred_label, slug, primary_category, review_state, trust_score, usage_count, created_at
FROM tags_old;

DROP TABLE tags_old;

PRAGMA foreign_keys = ON;
