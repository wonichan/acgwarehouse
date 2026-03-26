ALTER TABLE image_tags ADD COLUMN source TEXT DEFAULT 'manual';

UPDATE image_tags
SET source = 'ai'
WHERE source_observation_id IS NOT NULL;
