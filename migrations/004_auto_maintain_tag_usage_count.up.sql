-- Migration 004: Auto-maintain tags.usage_count via triggers
-- Only counts non-rejected associations (confirmed + pending contribute to usage_count)

-- Drop existing triggers first (in case migration is re-run)
DROP TRIGGER IF EXISTS trg_image_tags_after_insert;
DROP TRIGGER IF EXISTS trg_image_tags_after_delete;
DROP TRIGGER IF EXISTS trg_image_tags_after_update;

-- First, recalculate all usage_count values to correct any drift
UPDATE tags
SET usage_count = (
    SELECT COUNT(*) FROM image_tags WHERE tag_id = tags.id AND review_state != 'rejected'
);

-- Trigger: INSERT new image-tag association (only non-rejected counts)
CREATE TRIGGER trg_image_tags_after_insert
AFTER INSERT ON image_tags
FOR EACH ROW
WHEN NEW.review_state != 'rejected'
BEGIN
    UPDATE tags SET usage_count = usage_count + 1 WHERE id = NEW.tag_id;
END;

-- Trigger: DELETE image-tag association (only non-rejected counts)
CREATE TRIGGER trg_image_tags_after_delete
AFTER DELETE ON image_tags
FOR EACH ROW
WHEN OLD.review_state != 'rejected'
BEGIN
    UPDATE tags SET usage_count = MAX(usage_count - 1, 0) WHERE id = OLD.tag_id;
END;

-- Trigger: UPDATE image-tag association
-- Split into two triggers since SQLite only allows one WHEN per trigger

-- Handle tag_id changes (merge/replace scenario)
CREATE TRIGGER trg_image_tags_after_update_tagid
AFTER UPDATE ON image_tags
FOR EACH ROW
WHEN OLD.tag_id != NEW.tag_id
BEGIN
    -- Decrement old tag if it was counted
    UPDATE tags SET usage_count = MAX(usage_count - 1, 0)
    WHERE id = OLD.tag_id AND OLD.review_state != 'rejected';
    -- Increment new tag if it should be counted
    UPDATE tags SET usage_count = usage_count + 1
    WHERE id = NEW.tag_id AND NEW.review_state != 'rejected';
END;

-- Handle review_state transitions (pending/confirmed <-> rejected)
CREATE TRIGGER trg_image_tags_after_update_review
AFTER UPDATE ON image_tags
FOR EACH ROW
WHEN OLD.review_state != NEW.review_state
BEGIN
    -- Transition: non-rejected -> rejected (decrement)
    UPDATE tags SET usage_count = MAX(usage_count - 1, 0)
    WHERE id = NEW.tag_id
      AND OLD.review_state != 'rejected'
      AND NEW.review_state = 'rejected';
    -- Transition: rejected -> non-rejected (increment)
    UPDATE tags SET usage_count = usage_count + 1
    WHERE id = NEW.tag_id
      AND OLD.review_state = 'rejected'
      AND NEW.review_state != 'rejected';
END;