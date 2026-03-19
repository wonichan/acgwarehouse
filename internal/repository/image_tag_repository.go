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
	MergeImageTag(ctx context.Context, imageID, sourceTagID, targetTagID int64) error
	GetTagStats(ctx context.Context, tagID int64) (*TagStats, error)
}

type TagStats struct {
	TagID          int64
	UsageCount     int64
	ConfirmedCount int64
	PendingCount   int64
	RejectedCount  int64
	AICount        int64
	ManualCount    int64
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

// MergeImageTag reassigns an image's tag from sourceTagID to targetTagID.
// It removes the old image-tag association and creates a new one with the target tag.
func (r *imageTagRepository) MergeImageTag(ctx context.Context, imageID, sourceTagID, targetTagID int64) error {
	// Get the existing image-tag to preserve confidence and review_state
	var confidence float64
	var reviewState string
	var sourceObsID *int64
	err := r.db.QueryRowContext(ctx, `
		SELECT confidence, review_state, source_observation_id
		FROM image_tags WHERE image_id = ? AND tag_id = ?
	`, imageID, sourceTagID).Scan(&confidence, &reviewState, &sourceObsID)
	if err != nil {
		return err
	}

	// Delete the old association
	if _, err := r.db.ExecContext(ctx, `DELETE FROM image_tags WHERE image_id = ? AND tag_id = ?`, imageID, sourceTagID); err != nil {
		return err
	}

	// Create new association with target tag
	_, err = r.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO image_tags (image_id, tag_id, source_observation_id, confidence, review_state)
		VALUES (?, ?, ?, ?, ?)
	`, imageID, targetTagID, sourceObsID, confidence, reviewState)
	return err
}

// GetTagStats returns usage statistics for a tag including counts by review state and source.
func (r *imageTagRepository) GetTagStats(ctx context.Context, tagID int64) (*TagStats, error) {
	stats := &TagStats{TagID: tagID}

	// Get total usage count and counts by review state
	err := r.db.QueryRowContext(ctx, `
		SELECT 
			COUNT(*) as usage_count,
			COALESCE(SUM(CASE WHEN review_state = 'confirmed' THEN 1 ELSE 0 END), 0) as confirmed_count,
			COALESCE(SUM(CASE WHEN review_state = 'pending' THEN 1 ELSE 0 END), 0) as pending_count,
			COALESCE(SUM(CASE WHEN review_state = 'rejected' THEN 1 ELSE 0 END), 0) as rejected_count
		FROM image_tags WHERE tag_id = ?
	`, tagID).Scan(&stats.UsageCount, &stats.ConfirmedCount, &stats.PendingCount, &stats.RejectedCount)
	if err != nil {
		return nil, err
	}

	// Get AI count (associations with source_observation_id set)
	err = r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM image_tags WHERE tag_id = ? AND source_observation_id IS NOT NULL
	`, tagID).Scan(&stats.AICount)
	if err != nil {
		return nil, err
	}

	// Manual count = total - AI count
	stats.ManualCount = stats.UsageCount - stats.AICount

	return stats, nil
}
