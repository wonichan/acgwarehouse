package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/wonichan/acgwarehouse-backend/internal/d1client"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

type d1ImageTagRepository struct {
	client *d1client.Client
}

func NewD1ImageTagRepository(client *d1client.Client) ImageTagRepository {
	return &d1ImageTagRepository{client: client}
}

func (r *d1ImageTagRepository) FindByImageID(ctx context.Context, imageID int64) ([]*domain.ImageTag, error) {
	rows, err := r.client.Query(ctx, `
		SELECT image_id, tag_id, source, source_observation_id, confidence, review_state
		FROM image_tags WHERE image_id = ? ORDER BY tag_id ASC
	`, imageID)
	if err != nil {
		return nil, err
	}
	return mapImageTagsFromD1(rows)
}

func (r *d1ImageTagRepository) FindByTagID(ctx context.Context, tagID int64, limit, offset int) ([]*domain.ImageTag, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.client.Query(ctx, `
		SELECT image_id, tag_id, source, source_observation_id, confidence, review_state
		FROM image_tags WHERE tag_id = ?
		ORDER BY image_id ASC
		LIMIT ? OFFSET ?
	`, tagID, int64(limit), int64(offset))
	if err != nil {
		return nil, err
	}
	return mapImageTagsFromD1(rows)
}

func (r *d1ImageTagRepository) HasAITags(ctx context.Context, imageID int64) (bool, error) {
	cnt, err := r.client.QueryCount(ctx, `
		SELECT COUNT(*) as cnt
		FROM image_tags
		WHERE image_id = ? AND source = ? AND review_state != ?
	`, imageID, domain.ImageTagSourceAI, domain.ReviewStateRejected)
	if err != nil {
		return false, err
	}
	return cnt > 0, nil
}

func (r *d1ImageTagRepository) Exists(ctx context.Context, imageID, tagID int64) (bool, error) {
	cnt, err := r.client.QueryCount(ctx, `
		SELECT COUNT(*) as cnt FROM image_tags WHERE image_id = ? AND tag_id = ?
	`, imageID, tagID)
	if err != nil {
		return false, err
	}
	return cnt > 0, nil
}

func (r *d1ImageTagRepository) GetTagStats(ctx context.Context, tagID int64) (*TagStats, error) {
	row, err := r.client.QueryOne(ctx, `
		SELECT
			COUNT(*) as usage_count,
			COALESCE(SUM(CASE WHEN review_state = 'confirmed' THEN 1 ELSE 0 END), 0) as confirmed_count,
			COALESCE(SUM(CASE WHEN review_state = 'pending' THEN 1 ELSE 0 END), 0) as pending_count,
			COALESCE(SUM(CASE WHEN review_state = 'rejected' THEN 1 ELSE 0 END), 0) as rejected_count,
			COALESCE(SUM(CASE WHEN source = 'ai' THEN 1 ELSE 0 END), 0) as ai_count,
			COALESCE(SUM(CASE WHEN COALESCE(source, 'manual') != 'ai' THEN 1 ELSE 0 END), 0) as manual_count
		FROM image_tags WHERE tag_id = ?
	`, tagID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return &TagStats{TagID: tagID}, nil
	}
	usageCount, _ := toInt64(row["usage_count"])
	confirmedCount, _ := toInt64(row["confirmed_count"])
	pendingCount, _ := toInt64(row["pending_count"])
	rejectedCount, _ := toInt64(row["rejected_count"])
	aiCount, _ := toInt64(row["ai_count"])
	manualCount, _ := toInt64(row["manual_count"])

	return &TagStats{
		TagID:          tagID,
		UsageCount:     usageCount,
		ConfirmedCount: confirmedCount,
		PendingCount:   pendingCount,
		RejectedCount:  rejectedCount,
		AICount:        aiCount,
		ManualCount:    manualCount,
	}, nil
}

func (r *d1ImageTagRepository) Save(ctx context.Context, imageTag *domain.ImageTag) error {
	source := imageTag.Source
	if source == "" {
		source = domain.ImageTagSourceManual
	}
	err := r.client.Exec(ctx, `
		INSERT OR REPLACE INTO image_tags (image_id, tag_id, source, source_observation_id, confidence, review_state)
		VALUES (?, ?, ?, ?, ?, ?)
	`, imageTag.ImageID, imageTag.TagID, source, imageTag.SourceObservationID, imageTag.Confidence, imageTag.ReviewState)
	if err != nil {
		return err
	}
	return r.syncImageFTS(ctx, imageTag.ImageID)
}

func (r *d1ImageTagRepository) UpdateReviewState(ctx context.Context, imageID, tagID int64, state string) error {
	return r.client.Exec(ctx, `UPDATE image_tags SET review_state = ? WHERE image_id = ? AND tag_id = ?`, state, imageID, tagID)
}

func (r *d1ImageTagRepository) Delete(ctx context.Context, imageID, tagID int64) (int64, error) {
	// D1 doesn't support RowsAffected directly, so we check existence first
	exists, err := r.Exists(ctx, imageID, tagID)
	if err != nil {
		return 0, err
	}
	if !exists {
		return 0, nil
	}
	if err := r.client.Exec(ctx, `DELETE FROM image_tags WHERE image_id = ? AND tag_id = ?`, imageID, tagID); err != nil {
		return 0, err
	}
	if err := r.syncImageFTS(ctx, imageID); err != nil {
		return 1, err
	}
	return 1, nil
}

func (r *d1ImageTagRepository) BatchUpdateReviewState(ctx context.Context, imageID int64, tagIDs []int64, state string) error {
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
	return r.client.Exec(ctx, query, args...)
}

func (r *d1ImageTagRepository) MergeImageTag(ctx context.Context, imageID, sourceTagID, targetTagID int64) error {
	row, err := r.client.QueryOne(ctx, `
		SELECT confidence, review_state, source, source_observation_id
		FROM image_tags WHERE image_id = ? AND tag_id = ?
	`, imageID, sourceTagID)
	if err != nil {
		return err
	}
	if row == nil {
		return nil
	}
	confidence, _ := toFloat64(row["confidence"])
	reviewState := toStringDefault(row["review_state"], "")
	source := toStringDefault(row["source"], "")
	if source == "" {
		source = domain.ImageTagSourceManual
	}
	var sourceObsID *int64
	if sid, err := toInt64(row["source_observation_id"]); err == nil && sid != 0 {
		sourceObsID = &sid
	}

	if err := r.client.Exec(ctx, `DELETE FROM image_tags WHERE image_id = ? AND tag_id = ?`, imageID, sourceTagID); err != nil {
		return err
	}
	if err := r.client.Exec(ctx, `
		INSERT OR REPLACE INTO image_tags (image_id, tag_id, source, source_observation_id, confidence, review_state)
		VALUES (?, ?, ?, ?, ?, ?)
	`, imageID, targetTagID, source, sourceObsID, confidence, reviewState); err != nil {
		return err
	}
	return r.syncImageFTS(ctx, imageID)
}

func (r *d1ImageTagRepository) SyncFTSForTag(ctx context.Context, tagID int64) error {
	rows, err := r.client.Query(ctx, `SELECT image_id FROM image_tags WHERE tag_id = ?`, tagID)
	if err != nil {
		return err
	}
	for _, row := range rows {
		imageID, err := toInt64(row["image_id"])
		if err != nil {
			return err
		}
		if err := r.syncImageFTS(ctx, imageID); err != nil {
			return err
		}
	}
	return nil
}

func (r *d1ImageTagRepository) syncImageFTS(ctx context.Context, imageID int64) error {
	row, err := r.client.QueryOne(ctx, `
		SELECT COALESCE(GROUP_CONCAT(t.preferred_label, ' '), '') as tags
		FROM image_tags it
		JOIN tags t ON it.tag_id = t.id
		WHERE it.image_id = ?
	`, imageID)
	if err != nil {
		return err
	}
	tagsText := ""
	if row != nil {
		tagsText, _ = toString(row["tags"])
	}
	return r.client.Exec(ctx, `UPDATE images_fts SET tags = ? WHERE image_id = ?`, tagsText, imageID)
}

func mapImageTagsFromD1(rows []map[string]any) ([]*domain.ImageTag, error) {
	tags := make([]*domain.ImageTag, 0, len(rows))
	for _, row := range rows {
		tag, err := mapImageTagFromD1(row)
		if err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, nil
}