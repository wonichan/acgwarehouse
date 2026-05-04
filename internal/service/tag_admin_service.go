package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
)

var (
	ErrTagNotFound            = errors.New("tag not found")
	ErrMergeSameSourceTarget  = errors.New("source and target tags must be different")
	ErrCrossLevelMerge        = errors.New("merge requires tags at the same level")
	ErrMergeSourceHasChildren = errors.New("merge source has child tags")
	ErrInvalidHierarchy       = errors.New("invalid tag hierarchy")
)

const sqliteBulkQueryChunkSize = 900

type TagGovernanceRow struct {
	TagID                int64    `json:"tag_id"`
	PreferredLabel       string   `json:"preferred_label"`
	Level                string   `json:"level"`
	ParentID             *int64   `json:"parent_id,omitempty"`
	PrimaryCategory      string   `json:"primary_category"`
	Aliases              []string `json:"aliases"`
	UsageCount           int64    `json:"usage_count"`
	DirectUsageCount     int64    `json:"direct_usage_count"`
	TreeUsageCount       int64    `json:"tree_usage_count"`
	PendingCount         int64    `json:"pending_count"`
	DirectPendingCount   int64    `json:"direct_pending_count"`
	TreePendingCount     int64    `json:"tree_pending_count"`
	ConfirmedCount       int64    `json:"confirmed_count"`
	DirectConfirmedCount int64    `json:"direct_confirmed_count"`
	TreeConfirmedCount   int64    `json:"tree_confirmed_count"`
	RejectedCount        int64    `json:"rejected_count"`
	AICount              int64    `json:"ai_count"`
	DirectAICount        int64    `json:"direct_ai_count"`
	TreeAICount          int64    `json:"tree_ai_count"`
	ManualCount          int64    `json:"manual_count"`
	DirectManualCount    int64    `json:"direct_manual_count"`
	TreeManualCount      int64    `json:"tree_manual_count"`
	AffectedImageCount   int64    `json:"affected_image_count"`
	CanDelete            bool     `json:"can_delete"`
}

type TagMergeResult struct {
	SourceTagID               int64 `json:"source_tag_id"`
	TargetTagID               int64 `json:"target_tag_id"`
	MigratedImageAssociations int   `json:"migrated_image_associations"`
	MigratedAliases           int   `json:"migrated_aliases"`
}

type TagDeletePreview struct {
	TagID              int64  `json:"tag_id"`
	PreferredLabel     string `json:"preferred_label"`
	AffectedImageCount int64  `json:"affected_image_count"`
	ChildCount         int64  `json:"child_count"`
	CanDelete          bool   `json:"can_delete"`
	BlockingReason     string `json:"blocking_reason"`
}

type TagDeleteResult struct {
	DeletedTagID       int64 `json:"deleted_tag_id"`
	AffectedImageCount int64 `json:"affected_image_count"`
	DetachedChildCount int64 `json:"detached_child_count"`
}

type TagCleanupEntry struct {
	TagID              int64  `json:"tag_id"`
	PreferredLabel     string `json:"preferred_label"`
	AffectedImageCount int64  `json:"affected_image_count,omitempty"`
	BlockingReason     string `json:"blocking_reason,omitempty"`
	Error              string `json:"error,omitempty"`
}

type TagCleanupResult struct {
	Deleted []TagCleanupEntry `json:"deleted"`
	Blocked []TagCleanupEntry `json:"blocked"`
	Failed  []TagCleanupEntry `json:"failed"`
}

type TagTreeNode struct {
	TagID          int64         `json:"tag_id"`
	PreferredLabel string        `json:"preferred_label"`
	Level          string        `json:"level"`
	ParentID       *int64        `json:"parent_id,omitempty"`
	UsageCount     int64         `json:"usage_count"`
	TreeUsageCount int64         `json:"tree_usage_count"`
	Children       []TagTreeNode `json:"children"`
}

type TagAdminService struct {
	db           *sql.DB
	tagRepo      repository.TagRepository
	aliasRepo    repository.TagAliasRepository
	imageTagRepo repository.ImageTagRepository
}

type hierarchyStats struct {
	UsageCount     int64
	PendingCount   int64
	ConfirmedCount int64
	RejectedCount  int64
	AICount        int64
	ManualCount    int64
}

type hierarchyStatsResult struct {
	DirectUsageCount     int64
	DirectPendingCount   int64
	DirectConfirmedCount int64
	DirectAICount        int64
	DirectManualCount    int64
	TreeUsageCount       int64
	TreePendingCount     int64
	TreeConfirmedCount   int64
	TreeAICount          int64
	TreeManualCount      int64
	DirectRejectedCount  int64
}

func NewTagAdminService(db *sql.DB, tagRepo repository.TagRepository, aliasRepo repository.TagAliasRepository, imageTagRepo repository.ImageTagRepository) *TagAdminService {
	return &TagAdminService{
		db:           db,
		tagRepo:      tagRepo,
		aliasRepo:    aliasRepo,
		imageTagRepo: imageTagRepo,
	}
}

func chunkInt64IDs(ids []int64, chunkSize int) [][]int64 {
	if len(ids) == 0 {
		return nil
	}
	if chunkSize <= 0 || len(ids) <= chunkSize {
		return [][]int64{ids}
	}

	chunks := make([][]int64, 0, (len(ids)+chunkSize-1)/chunkSize)
	for start := 0; start < len(ids); start += chunkSize {
		end := start + chunkSize
		if end > len(ids) {
			end = len(ids)
		}
		chunks = append(chunks, ids[start:end])
	}
	return chunks
}

func queryTagByIDTx(ctx context.Context, tx *sql.Tx, id int64) (*domain.Tag, error) {
	tag := &domain.Tag{}
	err := scanServiceTag(tx.QueryRowContext(ctx, `
		SELECT id, preferred_label, slug, level, parent_id, primary_category, review_state, trust_score, usage_count, created_at
		FROM tags
		WHERE id = ?
	`, id), tag)
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

func queryChildrenByParentTx(ctx context.Context, tx *sql.Tx, parentID int64) ([]*domain.Tag, error) {
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
		if err := scanServiceTag(rows, child); err != nil {
			return nil, err
		}
		children = append(children, child)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return children, nil
}

func scanServiceTag(scanner interface{ Scan(dest ...any) error }, tag *domain.Tag) error {
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
	} else {
		tag.PrimaryCategory = ""
	}

	return nil
}

func countDirectAssociationsTx(ctx context.Context, tx *sql.Tx, tagID int64) (int64, error) {
	var count int64
	err := tx.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT image_id)
		FROM image_tags
		WHERE tag_id = ?
	`, tagID).Scan(&count)
	return count, err
}

func listImageIDsByTagTx(ctx context.Context, tx *sql.Tx, tagID int64) ([]int64, error) {
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return imageIDs, nil
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
