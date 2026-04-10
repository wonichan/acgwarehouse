DELETE FROM collection_images
WHERE rowid NOT IN (
    SELECT MIN(rowid)
    FROM collection_images
    GROUP BY image_id
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_collection_images_image_id_unique
ON collection_images(image_id);
