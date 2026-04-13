ALTER TABLE tags ADD COLUMN level TEXT NOT NULL DEFAULT 'child';

ALTER TABLE tags ADD COLUMN parent_id INTEGER REFERENCES tags(id) ON DELETE SET NULL;

UPDATE tags
SET level = 'child'
WHERE level IS NULL OR TRIM(level) = '';

UPDATE tags
SET parent_id = NULL
WHERE parent_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_tags_parent_id ON tags(parent_id);
