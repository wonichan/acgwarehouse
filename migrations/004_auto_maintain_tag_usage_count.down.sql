-- Migration 004: Remove auto-maintain triggers for tags.usage_count
-- Reverts to manual IncrementUsageCount/DecrementUsageCount mode

DROP TRIGGER IF EXISTS trg_image_tags_after_insert;
DROP TRIGGER IF EXISTS trg_image_tags_after_delete;
DROP TRIGGER IF EXISTS trg_image_tags_after_update;
