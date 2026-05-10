package repository

import (
	"context"
	"database/sql"

	"github.com/wonichan/acgwarehouse-backend/internal/d1client"
)

// UpdateImageTagsInFTS updates the tags text for an image in the FTS index.
// This should be called when an image's tags change.
func UpdateImageTagsInFTS(db *sql.DB, imageID int64, tagsText string) error {
	_, err := db.Exec(`
		UPDATE images_fts SET tags = ? WHERE image_id = ?
	`, tagsText, imageID)
	return err
}

// RebuildFTSIndex rebuilds the entire FTS index from the images and image_tags tables.
// This is useful for maintenance or recovery purposes.
func RebuildFTSIndex(db *sql.DB) error {
	// Clear existing FTS records
	_, err := db.Exec(`DELETE FROM images_fts`)
	if err != nil {
		return err
	}

	// Rebuild from images table with aggregated tags
	// Use COALESCE to handle images without tags
	query := `
		INSERT INTO images_fts(image_id, filename, tags)
		SELECT 
			i.id,
			i.filename,
			COALESCE(
				(SELECT GROUP_CONCAT(t.preferred_label, ' ')
				 FROM image_tags it
				 JOIN tags t ON it.tag_id = t.id
				 WHERE it.image_id = i.id),
				''
			) as tags
		FROM images i
	`
	_, err = db.Exec(query)
	return err
}

// RebuildD1FTSIndex rebuilds the D1 FTS index from D1 images and image_tags.
func RebuildD1FTSIndex(ctx context.Context, client *d1client.Client) error {
	return client.ExecBatch(ctx, []d1client.MutateStatement{
		{SQL: `DELETE FROM images_fts`},
		{SQL: `
			INSERT INTO images_fts(image_id, filename, tags)
			SELECT
				i.id,
				i.filename,
				COALESCE(
					(SELECT GROUP_CONCAT(t.preferred_label, ' ')
					 FROM image_tags it
					 JOIN tags t ON it.tag_id = t.id
					 WHERE it.image_id = i.id),
					''
				) as tags
			FROM images i
		`},
	})
}
