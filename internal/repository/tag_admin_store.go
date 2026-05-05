package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

type TagAdminStore interface {
	ChangeTagLevel(ctx context.Context, tag *domain.Tag, childrenToDetach []*domain.Tag) (*domain.Tag, error)
	CountDirectTagAssociations(ctx context.Context, tagID int64) (int64, error)
	DeleteTag(ctx context.Context, tagID int64) (*TagAdminDeleteResult, error)
	MergeTags(ctx context.Context, sourceTagID, targetTagID int64) (*TagAdminMergeResult, error)
	ReparentTag(ctx context.Context, tag *domain.Tag) (*domain.Tag, error)
}

type TagAdminDeleteResult struct {
	DeletedTagID       int64
	AffectedImageCount int64
	DetachedChildCount int64
}

type TagAdminMergeResult struct {
	SourceTagID               int64
	TargetTagID               int64
	MigratedImageAssociations int
	MigratedAliases           int
}

type TxRunner interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context, tx *sql.Tx) error) error
}

type sqlTxRunner struct {
	db *sql.DB
}

func (r sqlTxRunner) WithinTx(ctx context.Context, fn func(ctx context.Context, tx *sql.Tx) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if err := fn(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}

type sqliteTagAdminStore struct {
	db       *sql.DB
	txRunner TxRunner
}

func NewTagAdminStore(db *sql.DB) TagAdminStore {
	return &sqliteTagAdminStore{db: db, txRunner: sqlTxRunner{db: db}}
}

func (s *sqliteTagAdminStore) ChangeTagLevel(ctx context.Context, tag *domain.Tag, childrenToDetach []*domain.Tag) (*domain.Tag, error) {
	if err := s.txRunner.WithinTx(ctx, func(ctx context.Context, tx *sql.Tx) error {
		if err := updateTagForAdminTx(ctx, tx, tag); err != nil {
			return err
		}
		for _, child := range childrenToDetach {
			child.ParentID = nil
			if err := updateTagForAdminTx(ctx, tx, child); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return tag, nil
}

func (s *sqliteTagAdminStore) CountDirectTagAssociations(ctx context.Context, tagID int64) (int64, error) {
	var count int64
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT image_id)
		FROM image_tags
		WHERE tag_id = ?
	`, tagID).Scan(&count)
	return count, err
}

func (s *sqliteTagAdminStore) DeleteTag(ctx context.Context, tagID int64) (*TagAdminDeleteResult, error) {
	var children []*domain.Tag
	var affectedImageCount int64
	if err := s.txRunner.WithinTx(ctx, func(ctx context.Context, tx *sql.Tx) error {
		if _, err := queryTagByIDForAdminTx(ctx, tx, tagID); err != nil {
			return err
		}
		var err error
		children, err = queryChildrenByParentForAdminTx(ctx, tx, tagID)
		if err != nil {
			return err
		}
		affectedImageCount, err = countDirectAssociationsForAdminTx(ctx, tx, tagID)
		if err != nil {
			return err
		}
		imageIDs, err := listImageIDsByTagForAdminTx(ctx, tx, tagID)
		if err != nil {
			return err
		}

		for _, child := range children {
			if _, err := tx.ExecContext(ctx, `UPDATE tags SET parent_id = NULL WHERE id = ?`, child.ID); err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM image_tags WHERE tag_id = ?`, tagID); err != nil {
			return err
		}
		for _, imageID := range imageIDs {
			if err := syncImageFTSForAdminTx(ctx, tx, imageID); err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM tag_aliases WHERE tag_id = ?`, tagID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM tags WHERE id = ?`, tagID); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return &TagAdminDeleteResult{
		DeletedTagID:       tagID,
		AffectedImageCount: affectedImageCount,
		DetachedChildCount: int64(len(children)),
	}, nil
}

func (s *sqliteTagAdminStore) MergeTags(ctx context.Context, sourceTagID, targetTagID int64) (*TagAdminMergeResult, error) {
	result := &TagAdminMergeResult{
		SourceTagID: sourceTagID,
		TargetTagID: targetTagID,
	}

	if err := s.txRunner.WithinTx(ctx, func(ctx context.Context, tx *sql.Tx) error {
		sourceTag, err := queryTagByIDForAdminTx(ctx, tx, sourceTagID)
		if err != nil {
			return err
		}
		targetTag, err := queryTagByIDForAdminTx(ctx, tx, targetTagID)
		if err != nil {
			return err
		}

		if err := mergeImageAssociationsForAdminTx(ctx, tx, sourceTagID, targetTagID, result); err != nil {
			return err
		}
		if err := mergeAliasesForAdminTx(ctx, tx, sourceTag, targetTag, result); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM tag_aliases WHERE tag_id = ?`, sourceTagID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM tags WHERE id = ?`, sourceTagID); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *sqliteTagAdminStore) ReparentTag(ctx context.Context, tag *domain.Tag) (*domain.Tag, error) {
	if err := s.txRunner.WithinTx(ctx, func(ctx context.Context, tx *sql.Tx) error {
		return updateTagForAdminTx(ctx, tx, tag)
	}); err != nil {
		return nil, err
	}
	return tag, nil
}

func queryTagByIDForAdminTx(ctx context.Context, tx *sql.Tx, id int64) (*domain.Tag, error) {
	tag := &domain.Tag{}
	err := scanRepositoryTag(tx.QueryRowContext(ctx, `
		SELECT id, preferred_label, slug, level, parent_id, primary_category, review_state, trust_score, usage_count, created_at
		FROM tags
		WHERE id = ?
	`, id), tag)
	if err != nil {
		return nil, err
	}
	return tag, nil
}

func updateTagForAdminTx(ctx context.Context, tx *sql.Tx, tag *domain.Tag) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE tags
		SET preferred_label = ?, slug = ?, level = ?, parent_id = ?, primary_category = ?, review_state = ?, trust_score = ?, usage_count = ?
		WHERE id = ?
	`, tag.PreferredLabel, tag.Slug, tag.Level, tag.ParentID, tag.PrimaryCategory, tag.ReviewState, tag.TrustScore, tag.UsageCount, tag.ID)
	return err
}

func queryChildrenByParentForAdminTx(ctx context.Context, tx *sql.Tx, parentID int64) ([]*domain.Tag, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT id, preferred_label, slug, level, parent_id, primary_category, review_state, trust_score, usage_count, created_at
		FROM tags
		WHERE parent_id = ?
		ORDER BY usage_count DESC, id ASC
	`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	children := make([]*domain.Tag, 0)
	for rows.Next() {
		child := &domain.Tag{}
		if err := scanRepositoryTag(rows, child); err != nil {
			return nil, err
		}
		children = append(children, child)
	}
	return children, rows.Err()
}

func scanRepositoryTag(scanner interface{ Scan(dest ...any) error }, tag *domain.Tag) error {
	var primaryCategory sql.NullString
	if err := scanner.Scan(
		&tag.ID,
		&tag.PreferredLabel,
		&tag.Slug,
		&tag.Level,
		&tag.ParentID,
		&primaryCategory,
		&tag.ReviewState,
		&tag.TrustScore,
		&tag.UsageCount,
		&tag.CreatedAt,
	); err != nil {
		return err
	}
	if primaryCategory.Valid {
		tag.PrimaryCategory = primaryCategory.String
	}
	return nil
}

func countDirectAssociationsForAdminTx(ctx context.Context, tx *sql.Tx, tagID int64) (int64, error) {
	var count int64
	err := tx.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT image_id)
		FROM image_tags
		WHERE tag_id = ?
	`, tagID).Scan(&count)
	return count, err
}

func listImageIDsByTagForAdminTx(ctx context.Context, tx *sql.Tx, tagID int64) ([]int64, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT DISTINCT image_id
		FROM image_tags
		WHERE tag_id = ?
		ORDER BY image_id ASC
	`, tagID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	imageIDs := make([]int64, 0)
	for rows.Next() {
		var imageID int64
		if err := rows.Scan(&imageID); err != nil {
			return nil, err
		}
		imageIDs = append(imageIDs, imageID)
	}
	return imageIDs, rows.Err()
}

func syncImageFTSForAdminTx(ctx context.Context, tx *sql.Tx, imageID int64) error {
	var tagsText string
	err := tx.QueryRowContext(ctx, `
		SELECT COALESCE(GROUP_CONCAT(t.preferred_label, ' '), '')
		FROM image_tags it
		JOIN tags t ON t.id = it.tag_id
		WHERE it.image_id = ?
	`, imageID).Scan(&tagsText)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE images_fts SET tags = ? WHERE image_id = ?`, tagsText, imageID); err != nil {
		return fmt.Errorf("sync image fts: %w", err)
	}
	return nil
}

func mergeImageAssociationsForAdminTx(ctx context.Context, tx *sql.Tx, sourceTagID, targetTagID int64, result *TagAdminMergeResult) error {
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
		if err := syncImageFTSForAdminTx(ctx, tx, row.imageID); err != nil {
			return err
		}
		result.MigratedImageAssociations++
	}

	return nil
}

func mergeAliasesForAdminTx(ctx context.Context, tx *sql.Tx, sourceTag, targetTag *domain.Tag, result *TagAdminMergeResult) error {
	targetAliases, err := queryAliasesByTagIDForAdminTx(ctx, tx, targetTag.ID)
	if err != nil {
		return err
	}
	sourceAliases, err := queryAliasesByTagIDForAdminTx(ctx, tx, sourceTag.ID)
	if err != nil {
		return err
	}

	existingNormalized := map[string]struct{}{
		NormalizeLabel(targetTag.PreferredLabel): {},
	}
	for _, alias := range targetAliases {
		existingNormalized[NormalizeLabel(alias.Label)] = struct{}{}
	}

	appendAlias := func(alias *domain.TagAlias) error {
		normalized := NormalizeLabel(alias.Label)
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

	if NormalizeLabel(sourceTag.PreferredLabel) != NormalizeLabel(targetTag.PreferredLabel) {
		if err := appendAlias(&domain.TagAlias{Label: sourceTag.PreferredLabel, AliasType: "synonym"}); err != nil {
			return err
		}
	}

	return nil
}

func queryAliasesByTagIDForAdminTx(ctx context.Context, tx *sql.Tx, tagID int64) ([]*domain.TagAlias, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT id, tag_id, label, normalized_label, locale, alias_type, is_preferred
		FROM tag_aliases
		WHERE tag_id = ?
		ORDER BY id ASC
	`, tagID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	aliases := make([]*domain.TagAlias, 0)
	for rows.Next() {
		alias := &domain.TagAlias{}
		if err := rows.Scan(&alias.ID, &alias.TagID, &alias.Label, &alias.NormalizedLabel, &alias.Locale, &alias.AliasType, &alias.IsPreferred); err != nil {
			return nil, err
		}
		aliases = append(aliases, alias)
	}
	return aliases, rows.Err()
}
