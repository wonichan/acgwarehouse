package service

import (
	"context"
	"database/sql"
	"errors"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/logger"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
)

func (s *TagAdminService) MergeTags(ctx context.Context, sourceTagID, targetTagID int64) (*TagMergeResult, error) {
	logger.Infof("[service] TagAdmin MergeTags started: source_id=%d target_id=%d", sourceTagID, targetTagID)
	if sourceTagID <= 0 || targetTagID <= 0 {
		return nil, ErrTagNotFound
	}
	if sourceTagID == targetTagID {
		return nil, ErrMergeSameSourceTarget
	}
	if s.db == nil {
		return nil, errors.New("database is required")
	}

	mergeResult := &TagMergeResult{
		SourceTagID: sourceTagID,
		TargetTagID: targetTagID,
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		logger.Errorf("[service] TagAdmin MergeTags failed: %v", err)
		return nil, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	sourceTag, err := queryTagByIDTx(ctx, tx, sourceTagID)
	if err != nil {
		logger.Errorf("[service] TagAdmin MergeTags failed: %v", err)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTagNotFound
		}
		return nil, err
	}
	targetTag, err := queryTagByIDTx(ctx, tx, targetTagID)
	if err != nil {
		logger.Errorf("[service] TagAdmin MergeTags failed: %v", err)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTagNotFound
		}
		return nil, err
	}
	if sourceTag.Level != targetTag.Level {
		return nil, ErrCrossLevelMerge
	}
	children, err := s.tagRepo.FindChildrenByParent(ctx, sourceTagID)
	if err != nil {
		return nil, err
	}
	if len(children) > 0 {
		return nil, ErrMergeSourceHasChildren
	}

	if err := s.mergeImageAssociationsTx(ctx, tx, sourceTagID, targetTagID, mergeResult); err != nil {
		logger.Errorf("[service] TagAdmin MergeTags failed: %v", err)
		return nil, err
	}

	if err := s.mergeAliasesTx(ctx, tx, sourceTag, targetTag, mergeResult); err != nil {
		logger.Errorf("[service] TagAdmin MergeTags failed: %v", err)
		return nil, err
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM tag_aliases WHERE tag_id = ?`, sourceTagID); err != nil {
		logger.Errorf("[service] TagAdmin MergeTags failed: %v", err)
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM tags WHERE id = ?`, sourceTagID); err != nil {
		logger.Errorf("[service] TagAdmin MergeTags failed: %v", err)
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		logger.Errorf("[service] TagAdmin MergeTags failed: %v", err)
		return nil, err
	}
	tx = nil

	if err := s.imageTagRepo.SyncFTSForTag(ctx, targetTagID); err != nil {
		logger.Errorf("[service] TagAdmin MergeTags failed: %v", err)
		return nil, err
	}

	logger.Infof("[service] TagAdmin MergeTags completed: source_id=%d target_id=%d migrated_images=%d migrated_aliases=%d", sourceTagID, targetTagID, mergeResult.MigratedImageAssociations, mergeResult.MigratedAliases)
	return mergeResult, nil
}

func (s *TagAdminService) mergeImageAssociationsTx(ctx context.Context, tx *sql.Tx, sourceTagID, targetTagID int64, result *TagMergeResult) error {
	rows, err := tx.QueryContext(ctx, `
		SELECT image_id, source, source_observation_id, confidence, review_state
		FROM image_tags
		WHERE tag_id = ?
		ORDER BY image_id ASC
	`, sourceTagID)
	if err != nil {
		return err
	}
	defer rows.Close()

	type mergeRow struct {
		imageID             int64
		source              string
		sourceObservationID *int64
		confidence          float64
		reviewState         string
	}
	mergeRows := make([]mergeRow, 0)
	for rows.Next() {
		var row mergeRow
		if err := rows.Scan(&row.imageID, &row.source, &row.sourceObservationID, &row.confidence, &row.reviewState); err != nil {
			return err
		}
		if row.source == "" {
			row.source = domain.ImageTagSourceManual
		}
		mergeRows = append(mergeRows, row)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, row := range mergeRows {
		if _, err := tx.ExecContext(ctx, `DELETE FROM image_tags WHERE image_id = ? AND tag_id = ?`, row.imageID, sourceTagID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT OR REPLACE INTO image_tags (image_id, tag_id, source, source_observation_id, confidence, review_state)
			VALUES (?, ?, ?, ?, ?, ?)
		`, row.imageID, targetTagID, row.source, row.sourceObservationID, row.confidence, row.reviewState); err != nil {
			return err
		}
		if err := syncImageFTSForTx(ctx, tx, row.imageID); err != nil {
			return err
		}
		result.MigratedImageAssociations++
	}

	return nil
}

func (s *TagAdminService) mergeAliasesTx(ctx context.Context, tx *sql.Tx, sourceTag, targetTag *domain.Tag, result *TagMergeResult) error {
	targetAliases, err := queryAliasesByTagIDTx(ctx, tx, targetTag.ID)
	if err != nil {
		return err
	}
	sourceAliases, err := queryAliasesByTagIDTx(ctx, tx, sourceTag.ID)
	if err != nil {
		return err
	}

	existingNormalized := map[string]struct{}{
		repository.NormalizeLabel(targetTag.PreferredLabel): {},
	}
	for _, alias := range targetAliases {
		existingNormalized[repository.NormalizeLabel(alias.Label)] = struct{}{}
	}

	appendAlias := func(alias *domain.TagAlias) error {
		normalized := repository.NormalizeLabel(alias.Label)
		if normalized == "" {
			return nil
		}
		if _, exists := existingNormalized[normalized]; exists {
			return nil
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT OR IGNORE INTO tag_aliases (tag_id, label, normalized_label, locale, alias_type, is_preferred)
			VALUES (?, ?, ?, ?, ?, ?)
		`, targetTag.ID, alias.Label, normalized, alias.Locale, alias.AliasType, alias.IsPreferred); err != nil {
			return err
		}
		existingNormalized[normalized] = struct{}{}
		result.MigratedAliases++
		return nil
	}

	for _, sourceAlias := range sourceAliases {
		if err := appendAlias(sourceAlias); err != nil {
			return err
		}
	}

	if repository.NormalizeLabel(sourceTag.PreferredLabel) != repository.NormalizeLabel(targetTag.PreferredLabel) {
		if err := appendAlias(&domain.TagAlias{Label: sourceTag.PreferredLabel, AliasType: "synonym"}); err != nil {
			return err
		}
	}

	return nil
}
