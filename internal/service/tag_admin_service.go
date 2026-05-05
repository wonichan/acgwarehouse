package service

import (
	"database/sql"
	"errors"

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
	adminStore   repository.TagAdminStore
	govQuery     repository.TagGovernanceQuery
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
	return newTagAdminServiceWithStore(db, tagRepo, aliasRepo, imageTagRepo, nil, nil)
}

func newTagAdminServiceWithStore(db *sql.DB, tagRepo repository.TagRepository, aliasRepo repository.TagAliasRepository, imageTagRepo repository.ImageTagRepository, adminStore repository.TagAdminStore, govQuery repository.TagGovernanceQuery) *TagAdminService {
	if adminStore == nil {
		adminStore = repository.NewTagAdminStore(db)
	}
	if govQuery == nil {
		govQuery = repository.NewTagGovernanceQuery(db)
	}
	return &TagAdminService{
		db:           db,
		tagRepo:      tagRepo,
		aliasRepo:    aliasRepo,
		imageTagRepo: imageTagRepo,
		adminStore:   adminStore,
		govQuery:     govQuery,
	}
}

func chunkInt64IDs(ids []int64, chunkSize int) [][]int64 {
	return repository.ChunkInt64IDs(ids, chunkSize)
}
