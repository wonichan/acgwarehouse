package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
)

var (
	ErrTagNotFound           = errors.New("tag not found")
	ErrMergeSameSourceTarget = errors.New("source and target tags must be different")
)

type TagGovernanceRow struct {
	TagID              int64    `json:"tag_id"`
	PreferredLabel     string   `json:"preferred_label"`
	PrimaryCategory    string   `json:"primary_category"`
	Aliases            []string `json:"aliases"`
	UsageCount         int64    `json:"usage_count"`
	PendingCount       int64    `json:"pending_count"`
	ConfirmedCount     int64    `json:"confirmed_count"`
	RejectedCount      int64    `json:"rejected_count"`
	AICount            int64    `json:"ai_count"`
	ManualCount        int64    `json:"manual_count"`
	AffectedImageCount int64    `json:"affected_image_count"`
	CanDelete          bool     `json:"can_delete"`
}

type TagMergeResult struct {
	SourceTagID               int64 `json:"source_tag_id"`
	TargetTagID               int64 `json:"target_tag_id"`
	MigratedImageAssociations int   `json:"migrated_image_associations"`
	MigratedAliases           int   `json:"migrated_aliases"`
}

type TagAdminService struct {
	db           *sql.DB
	tagRepo      repository.TagRepository
	aliasRepo    repository.TagAliasRepository
	imageTagRepo repository.ImageTagRepository
}

func NewTagAdminService(db *sql.DB, tagRepo repository.TagRepository, aliasRepo repository.TagAliasRepository, imageTagRepo repository.ImageTagRepository) *TagAdminService {
	return &TagAdminService{
		db:           db,
		tagRepo:      tagRepo,
		aliasRepo:    aliasRepo,
		imageTagRepo: imageTagRepo,
	}
}

func (s *TagAdminService) ListGovernanceTags(ctx context.Context, search string, limit, offset int) ([]TagGovernanceRow, int, error) {
	search = strings.TrimSpace(search)
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	tags, total, err := s.resolveTagSlice(ctx, search, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	rows := make([]TagGovernanceRow, 0, len(tags))
	for _, tag := range tags {
		aliases, err := s.aliasRepo.FindByTagID(ctx, tag.ID)
		if err != nil {
			return nil, 0, err
		}

		stats, err := s.imageTagRepo.GetTagStats(ctx, tag.ID)
		if err != nil {
			return nil, 0, err
		}

		aliasLabels := make([]string, 0, len(aliases))
		for _, alias := range aliases {
			aliasLabels = append(aliasLabels, alias.Label)
		}

		row := TagGovernanceRow{
			TagID:              tag.ID,
			PreferredLabel:     tag.PreferredLabel,
			PrimaryCategory:    tag.PrimaryCategory,
			Aliases:            aliasLabels,
			UsageCount:         stats.UsageCount,
			PendingCount:       stats.PendingCount,
			ConfirmedCount:     stats.ConfirmedCount,
			RejectedCount:      stats.RejectedCount,
			AICount:            stats.AICount,
			ManualCount:        stats.ManualCount,
			AffectedImageCount: stats.UsageCount,
			CanDelete:          stats.UsageCount == 0,
		}
		rows = append(rows, row)
	}

	return rows, total, nil
}

func (s *TagAdminService) MergeTags(ctx context.Context, sourceTagID, targetTagID int64) (*TagMergeResult, error) {
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
		return nil, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	sourceTag, err := queryTagByIDTx(ctx, tx, sourceTagID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTagNotFound
		}
		return nil, err
	}
	targetTag, err := queryTagByIDTx(ctx, tx, targetTagID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTagNotFound
		}
		return nil, err
	}

	if err := s.mergeImageAssociationsTx(ctx, tx, sourceTagID, targetTagID, mergeResult); err != nil {
		return nil, err
	}

	if err := s.mergeAliasesTx(ctx, tx, sourceTag, targetTag, mergeResult); err != nil {
		return nil, err
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM tag_aliases WHERE tag_id = ?`, sourceTagID); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM tags WHERE id = ?`, sourceTagID); err != nil {
		return nil, err
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE tags
		SET usage_count = (SELECT COUNT(*) FROM image_tags WHERE tag_id = ?)
		WHERE id = ?
	`, targetTagID, targetTagID); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	tx = nil

	if err := s.imageTagRepo.SyncFTSForTag(ctx, targetTagID); err != nil {
		return nil, err
	}

	return mergeResult, nil
}

func (s *TagAdminService) resolveTagSlice(ctx context.Context, search string, limit, offset int) ([]*domain.Tag, int, error) {
	if search == "" {
		tags, err := s.tagRepo.FindAll(ctx, limit, offset)
		if err != nil {
			return nil, 0, err
		}
		total, err := s.tagRepo.Count(ctx)
		if err != nil {
			return nil, 0, err
		}
		return tags, total, nil
	}

	results := make(map[int64]*domain.Tag)

	byLabel, err := s.tagRepo.FindByLabelLike(ctx, search, limit+offset)
	if err != nil {
		return nil, 0, err
	}
	for _, tag := range byLabel {
		results[tag.ID] = tag
	}

	aliases, err := s.aliasRepo.FindByLabelLike(ctx, search)
	if err != nil {
		return nil, 0, err
	}
	for _, alias := range aliases {
		if _, exists := results[alias.TagID]; exists {
			continue
		}
		tag, err := s.tagRepo.FindByID(ctx, alias.TagID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			return nil, 0, err
		}
		results[tag.ID] = tag
	}

	merged := make([]*domain.Tag, 0, len(results))
	for _, tag := range results {
		merged = append(merged, tag)
	}
	sort.Slice(merged, func(i, j int) bool {
		if merged[i].UsageCount == merged[j].UsageCount {
			return merged[i].ID < merged[j].ID
		}
		return merged[i].UsageCount > merged[j].UsageCount
	})

	total := len(merged)
	if offset >= total {
		return []*domain.Tag{}, total, nil
	}
	end := total
	if offset+limit < end {
		end = offset + limit
	}
	return merged[offset:end], total, nil
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

func queryTagByIDTx(ctx context.Context, tx *sql.Tx, id int64) (*domain.Tag, error) {
	tag := &domain.Tag{}
	err := tx.QueryRowContext(ctx, `
		SELECT id, preferred_label, slug, primary_category, review_state, trust_score, usage_count, created_at
		FROM tags
		WHERE id = ?
	`, id).Scan(
		&tag.ID,
		&tag.PreferredLabel,
		&tag.Slug,
		&tag.PrimaryCategory,
		&tag.ReviewState,
		&tag.TrustScore,
		&tag.UsageCount,
		&tag.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return tag, nil
}

func queryAliasesByTagIDTx(ctx context.Context, tx *sql.Tx, tagID int64) ([]*domain.TagAlias, error) {
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return aliases, nil
}

func syncImageFTSForTx(ctx context.Context, tx *sql.Tx, imageID int64) error {
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
