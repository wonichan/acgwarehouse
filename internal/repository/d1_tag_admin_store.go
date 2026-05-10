package repository

import (
	"context"
	"database/sql"

	"github.com/wonichan/acgwarehouse-backend/internal/d1client"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

type d1TagAdminStore struct {
	client *d1client.Client
}

func NewD1TagAdminStore(client *d1client.Client) TagAdminStore {
	return &d1TagAdminStore{client: client}
}

func (s *d1TagAdminStore) ChangeTagLevel(ctx context.Context, tag *domain.Tag, childrenToDetach []*domain.Tag) (*domain.Tag, error) {
	statements := []d1client.MutateStatement{
		updateTagAdminStatement(tag),
	}
	for _, child := range childrenToDetach {
		child.ParentID = nil
		statements = append(statements, updateTagAdminStatement(child))
	}
	if err := s.client.ExecBatch(ctx, statements); err != nil {
		return nil, err
	}
	return tag, nil
}

func (s *d1TagAdminStore) CountDirectTagAssociations(ctx context.Context, tagID int64) (int64, error) {
	return s.client.QueryCount(ctx, `
		SELECT COUNT(DISTINCT image_id) as cnt
		FROM image_tags
		WHERE tag_id = ?
	`, tagID)
}

func (s *d1TagAdminStore) DeleteTag(ctx context.Context, tagID int64) (*TagAdminDeleteResult, error) {
	if _, err := s.queryTagByID(ctx, tagID); err != nil {
		return nil, err
	}

	children, err := s.queryChildrenByParent(ctx, tagID)
	if err != nil {
		return nil, err
	}
	affectedImageCount, err := s.CountDirectTagAssociations(ctx, tagID)
	if err != nil {
		return nil, err
	}
	imageIDs, err := s.listImageIDsByTag(ctx, tagID)
	if err != nil {
		return nil, err
	}

	statements := make([]d1client.MutateStatement, 0, len(children)+len(imageIDs)+3)
	for _, child := range children {
		statements = append(statements, d1client.MutateStatement{
			SQL:    `UPDATE tags SET parent_id = NULL WHERE id = ?`,
			Params: []any{child.ID},
		})
	}
	statements = append(statements, d1client.MutateStatement{
		SQL:    `DELETE FROM image_tags WHERE tag_id = ?`,
		Params: []any{tagID},
	})
	for _, imageID := range imageIDs {
		statements = append(statements, d1client.MutateStatement{
			SQL: `UPDATE images_fts SET tags = (
				SELECT COALESCE(GROUP_CONCAT(t.preferred_label, ' '), '')
				FROM image_tags it
				JOIN tags t ON t.id = it.tag_id
				WHERE it.image_id = ?
			) WHERE image_id = ?`,
			Params: []any{imageID, imageID},
		})
	}
	statements = append(statements,
		d1client.MutateStatement{SQL: `DELETE FROM tag_aliases WHERE tag_id = ?`, Params: []any{tagID}},
		d1client.MutateStatement{SQL: `DELETE FROM tags WHERE id = ?`, Params: []any{tagID}},
	)

	if err := s.client.ExecBatch(ctx, statements); err != nil {
		return nil, err
	}
	return &TagAdminDeleteResult{
		DeletedTagID:       tagID,
		AffectedImageCount: affectedImageCount,
		DetachedChildCount: int64(len(children)),
	}, nil
}

func (s *d1TagAdminStore) MergeTags(ctx context.Context, sourceTagID, targetTagID int64) (*TagAdminMergeResult, error) {
	sourceTag, err := s.queryTagByID(ctx, sourceTagID)
	if err != nil {
		return nil, err
	}
	targetTag, err := s.queryTagByID(ctx, targetTagID)
	if err != nil {
		return nil, err
	}

	mergeRows, err := s.queryImageTagsByTag(ctx, sourceTagID)
	if err != nil {
		return nil, err
	}
	targetAliases, err := s.queryAliasesByTagID(ctx, targetTagID)
	if err != nil {
		return nil, err
	}
	sourceAliases, err := s.queryAliasesByTagID(ctx, sourceTagID)
	if err != nil {
		return nil, err
	}

	result := &TagAdminMergeResult{SourceTagID: sourceTagID, TargetTagID: targetTagID}
	statements := make([]d1client.MutateStatement, 0, len(mergeRows)*3+len(sourceAliases)+4)
	for _, row := range mergeRows {
		source := row.Source
		if source == "" {
			source = domain.ImageTagSourceManual
		}
		statements = append(statements,
			d1client.MutateStatement{SQL: `DELETE FROM image_tags WHERE image_id = ? AND tag_id = ?`, Params: []any{row.ImageID, sourceTagID}},
			d1client.MutateStatement{
				SQL: `INSERT OR REPLACE INTO image_tags (image_id, tag_id, source, source_observation_id, confidence, review_state)
					VALUES (?, ?, ?, ?, ?, ?)`,
				Params: []any{row.ImageID, targetTagID, source, row.SourceObservationID, row.Confidence, row.ReviewState},
			},
			d1client.MutateStatement{
				SQL: `UPDATE images_fts SET tags = (
					SELECT COALESCE(GROUP_CONCAT(t.preferred_label, ' '), '')
					FROM image_tags it
					JOIN tags t ON t.id = it.tag_id
					WHERE it.image_id = ?
				) WHERE image_id = ?`,
				Params: []any{row.ImageID, row.ImageID},
			},
		)
		result.MigratedImageAssociations++
	}

	existingNormalized := map[string]struct{}{
		NormalizeLabel(targetTag.PreferredLabel): {},
	}
	for _, alias := range targetAliases {
		existingNormalized[NormalizeLabel(alias.Label)] = struct{}{}
	}
	appendAlias := func(alias *domain.TagAlias) {
		normalized := NormalizeLabel(alias.Label)
		if normalized == "" {
			return
		}
		if _, exists := existingNormalized[normalized]; exists {
			return
		}
		statements = append(statements, d1client.MutateStatement{
			SQL: `INSERT OR IGNORE INTO tag_aliases (tag_id, label, normalized_label, locale, alias_type, is_preferred)
				VALUES (?, ?, ?, ?, ?, ?)`,
			Params: []any{targetTagID, alias.Label, normalized, alias.Locale, alias.AliasType, alias.IsPreferred},
		})
		existingNormalized[normalized] = struct{}{}
		result.MigratedAliases++
	}
	for _, sourceAlias := range sourceAliases {
		appendAlias(sourceAlias)
	}
	if NormalizeLabel(sourceTag.PreferredLabel) != NormalizeLabel(targetTag.PreferredLabel) {
		appendAlias(&domain.TagAlias{Label: sourceTag.PreferredLabel, AliasType: "synonym"})
	}

	statements = append(statements,
		d1client.MutateStatement{SQL: `DELETE FROM tag_aliases WHERE tag_id = ?`, Params: []any{sourceTagID}},
		d1client.MutateStatement{SQL: `DELETE FROM tags WHERE id = ?`, Params: []any{sourceTagID}},
	)
	if err := s.client.ExecBatch(ctx, statements); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *d1TagAdminStore) ReparentTag(ctx context.Context, tag *domain.Tag) (*domain.Tag, error) {
	statement := updateTagAdminStatement(tag)
	if err := s.client.Exec(ctx, statement.SQL, statement.Params...); err != nil {
		return nil, err
	}
	return tag, nil
}

func updateTagAdminStatement(tag *domain.Tag) d1client.MutateStatement {
	return d1client.MutateStatement{
		SQL: `UPDATE tags
			SET preferred_label = ?, slug = ?, level = ?, parent_id = ?, primary_category = ?, review_state = ?, trust_score = ?, usage_count = ?
			WHERE id = ?`,
		Params: []any{tag.PreferredLabel, tag.Slug, tag.Level, tag.ParentID, tag.PrimaryCategory, tag.ReviewState, tag.TrustScore, tag.UsageCount, tag.ID},
	}
}

func (s *d1TagAdminStore) queryTagByID(ctx context.Context, id int64) (*domain.Tag, error) {
	row, err := s.client.QueryOne(ctx, `
		SELECT id, preferred_label, slug, level, parent_id, primary_category, review_state, trust_score, usage_count, created_at
		FROM tags
		WHERE id = ?
	`, id)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, sql.ErrNoRows
	}
	return mapTagFromD1(row)
}

func (s *d1TagAdminStore) queryChildrenByParent(ctx context.Context, parentID int64) ([]*domain.Tag, error) {
	rows, err := s.client.Query(ctx, `
		SELECT id, preferred_label, slug, level, parent_id, primary_category, review_state, trust_score, usage_count, created_at
		FROM tags
		WHERE parent_id = ?
		ORDER BY usage_count DESC, id ASC
	`, parentID)
	if err != nil {
		return nil, err
	}
	return mapTagsFromD1(rows)
}

func (s *d1TagAdminStore) listImageIDsByTag(ctx context.Context, tagID int64) ([]int64, error) {
	rows, err := s.client.Query(ctx, `SELECT DISTINCT image_id FROM image_tags WHERE tag_id = ? ORDER BY image_id ASC`, tagID)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		id, err := toInt64(row["image_id"])
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

type d1TagAdminImageTagRow struct {
	ImageID             int64
	Source              string
	SourceObservationID *int64
	Confidence          float64
	ReviewState         string
}

func (s *d1TagAdminStore) queryImageTagsByTag(ctx context.Context, tagID int64) ([]d1TagAdminImageTagRow, error) {
	rows, err := s.client.Query(ctx, `
		SELECT image_id, source, source_observation_id, confidence, review_state
		FROM image_tags
		WHERE tag_id = ?
		ORDER BY image_id ASC
	`, tagID)
	if err != nil {
		return nil, err
	}
	result := make([]d1TagAdminImageTagRow, 0, len(rows))
	for _, raw := range rows {
		imageID, err := toInt64(raw["image_id"])
		if err != nil {
			return nil, err
		}
		sourceObservationID := (*int64)(nil)
		if id, err := toInt64(raw["source_observation_id"]); err == nil && id != 0 {
			sourceObservationID = &id
		}
		confidence, _ := toFloat64(raw["confidence"])
		result = append(result, d1TagAdminImageTagRow{
			ImageID:             imageID,
			Source:              toStringDefault(raw["source"], ""),
			SourceObservationID: sourceObservationID,
			Confidence:          confidence,
			ReviewState:         toStringDefault(raw["review_state"], ""),
		})
	}
	return result, nil
}

func (s *d1TagAdminStore) queryAliasesByTagID(ctx context.Context, tagID int64) ([]*domain.TagAlias, error) {
	rows, err := s.client.Query(ctx, `
		SELECT id, tag_id, label, normalized_label, locale, alias_type, is_preferred
		FROM tag_aliases
		WHERE tag_id = ?
		ORDER BY id ASC
	`, tagID)
	if err != nil {
		return nil, err
	}
	return mapTagAliasesFromD1(rows)
}
