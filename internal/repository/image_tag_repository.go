package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

type ImageTagRepository interface {
	Save(ctx context.Context, imageTag *domain.ImageTag) error
	FindByImageID(ctx context.Context, imageID int64) ([]*domain.ImageTag, error)
	FindByTagID(ctx context.Context, tagID int64, limit, offset int) ([]*domain.ImageTag, error)
	UpdateReviewState(ctx context.Context, imageID, tagID int64, state string) error
	Delete(ctx context.Context, imageID, tagID int64) error
	BatchUpdateReviewState(ctx context.Context, imageID int64, tagIDs []int64, state string) error
}

type imageTagRepository struct {
	db *sql.DB
}

func NewImageTagRepository(db *sql.DB) ImageTagRepository {
	return &imageTagRepository{db: db}
}

func (r *imageTagRepository) Save(ctx context.Context, imageTag *domain.ImageTag) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO image_tags (image_id, tag_id, source_observation_id, confidence, review_state)
		VALUES (?, ?, ?, ?, ?)
	`, imageTag.ImageID, imageTag.TagID, imageTag.SourceObservationID, imageTag.Confidence, imageTag.ReviewState)
	return err
}

func (r *imageTagRepository) FindByImageID(ctx context.Context, imageID int64) ([]*domain.ImageTag, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT it.image_id, it.tag_id, it.source_observation_id, it.confidence, it.review_state
		FROM image_tags it
		INNER JOIN tags t ON t.id = it.tag_id
		WHERE it.image_id = ?
		ORDER BY t.usage_count DESC, it.tag_id ASC
	`, imageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanImageTags(rows)
}

func (r *imageTagRepository) FindByTagID(ctx context.Context, tagID int64, limit, offset int) ([]*domain.ImageTag, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT image_id, tag_id, source_observation_id, confidence, review_state
		FROM image_tags WHERE tag_id = ?
		ORDER BY image_id ASC
		LIMIT ? OFFSET ?
	`, tagID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanImageTags(rows)
}

func (r *imageTagRepository) UpdateReviewState(ctx context.Context, imageID, tagID int64, state string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE image_tags SET review_state = ? WHERE image_id = ? AND tag_id = ?`, state, imageID, tagID)
	return err
}

func (r *imageTagRepository) Delete(ctx context.Context, imageID, tagID int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM image_tags WHERE image_id = ? AND tag_id = ?`, imageID, tagID)
	return err
}

func (r *imageTagRepository) BatchUpdateReviewState(ctx context.Context, imageID int64, tagIDs []int64, state string) error {
	if len(tagIDs) == 0 {
		return nil
	}

	placeholders := make([]string, len(tagIDs))
	args := make([]any, 0, len(tagIDs)+2)
	args = append(args, state, imageID)
	for i, tagID := range tagIDs {
		placeholders[i] = "?"
		args = append(args, tagID)
	}

	query := fmt.Sprintf(`UPDATE image_tags SET review_state = ? WHERE image_id = ? AND tag_id IN (%s)`, strings.Join(placeholders, ", "))
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func scanImageTags(rows *sql.Rows) ([]*domain.ImageTag, error) {
	imageTags := make([]*domain.ImageTag, 0)
	for rows.Next() {
		imageTag := &domain.ImageTag{}
		if err := rows.Scan(&imageTag.ImageID, &imageTag.TagID, &imageTag.SourceObservationID, &imageTag.Confidence, &imageTag.ReviewState); err != nil {
			return nil, err
		}
		imageTags = append(imageTags, imageTag)
	}

	return imageTags, rows.Err()
}
