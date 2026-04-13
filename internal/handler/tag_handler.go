package handler

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
	"github.com/wonichan/acgwarehouse-backend/internal/service"
)

type tagAdminService interface {
	ListGovernanceTags(ctx context.Context, search string, limit, offset int) ([]service.TagGovernanceRow, int, error)
	MergeTags(ctx context.Context, sourceTagID, targetTagID int64) (*service.TagMergeResult, error)
	GetDeletePreview(ctx context.Context, tagID int64) (*service.TagDeletePreview, error)
	CleanupUnusedTags(ctx context.Context, tagIDs []int64) (*service.TagCleanupResult, error)
	GetParentCandidates(ctx context.Context, targetLevel string) ([]*domain.Tag, error)
	GetTagTree(ctx context.Context) ([]service.TagTreeNode, error)
	ChangeLevel(ctx context.Context, tagID int64, targetLevel string, parentID *int64) (*domain.Tag, error)
	ReparentTag(ctx context.Context, tagID int64, parentID *int64) (*domain.Tag, error)
}

const mergeOrReclassifyRequired = "merge_or_reclassify_required"

type TagHandler struct {
	tagRepo      repository.TagRepository
	aliasRepo    repository.TagAliasRepository
	imageTagRepo repository.ImageTagRepository
	adminSvc     tagAdminService
}

func NewTagHandler(tagRepo repository.TagRepository, aliasRepo repository.TagAliasRepository, imageTagRepo repository.ImageTagRepository, adminSvcOpt ...tagAdminService) *TagHandler {
	var adminSvc tagAdminService
	if len(adminSvcOpt) > 0 {
		adminSvc = adminSvcOpt[0]
	}

	return &TagHandler{
		tagRepo:      tagRepo,
		aliasRepo:    aliasRepo,
		imageTagRepo: imageTagRepo,
		adminSvc:     adminSvc,
	}
}

func (h *TagHandler) GetTags(c *gin.Context) {
	limit := parsePositiveInt(c.DefaultQuery("limit", "20"), 20)
	offset := parsePositiveInt(c.DefaultQuery("offset", "0"), 0)
	search := strings.TrimSpace(c.Query("search"))

	var (
		tags []*domain.Tag
		err  error
	)
	if search != "" {
		tags, err = h.SearchTags(c.Request.Context(), search, limit+offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		total := len(tags)
		tags = paginateTags(tags, limit, offset)
		c.JSON(http.StatusOK, gin.H{"tags": tags, "total": total})
		return
	}

	tags, err = h.tagRepo.FindAll(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	total, err := h.tagRepo.Count(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tags": tags, "total": total})
}

func (h *TagHandler) SearchTags(ctx context.Context, query string, limit int) ([]*domain.Tag, error) {
	results := make(map[int64]*domain.Tag)

	tags, err := h.tagRepo.FindByLabelLike(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	for _, tag := range tags {
		results[tag.ID] = tag
	}

	aliases, err := h.aliasRepo.FindByLabelLike(ctx, query)
	if err != nil {
		return nil, err
	}
	for _, alias := range aliases {
		if _, ok := results[alias.TagID]; ok {
			continue
		}
		tag, err := h.tagRepo.FindByID(ctx, alias.TagID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			return nil, err
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

	if limit > 0 && len(merged) > limit {
		return merged[:limit], nil
	}
	return merged, nil
}

func (h *TagHandler) CreateTag(c *gin.Context) {
	var req struct {
		PreferredLabel  string `json:"preferred_label"`
		PrimaryCategory string `json:"primary_category"`
		Level           string `json:"level"`
		ParentID        *int64 `json:"parent_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if strings.TrimSpace(req.PreferredLabel) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "preferred_label is required"})
		return
	}
	tag, reused, actualLevel, err := resolveOrCreateManualTag(c.Request.Context(), h.tagRepo, h.aliasRepo, manualTagCreateInput{
		PreferredLabel:  req.PreferredLabel,
		PrimaryCategory: req.PrimaryCategory,
		Level:           req.Level,
		ParentID:        req.ParentID,
		ReviewState:     "confirmed",
	})
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, service.ErrInvalidHierarchy) {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	if reused {
		response := gin.H{"id": tag.ID, "reused": true, "tag": tag}
		if req.Level != "" && actualLevel != req.Level {
			response["requested_level"] = req.Level
			response["actual_level"] = actualLevel
		}
		c.JSON(http.StatusOK, response)
		return
	}

	c.JSON(http.StatusCreated, tag)
}

func (h *TagHandler) UpdateTag(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	tag, err := h.tagRepo.FindByID(c.Request.Context(), id)
	if err != nil {
		respondRepoError(c, err)
		return
	}

	var req struct {
		PreferredLabel  string `json:"preferred_label"`
		PrimaryCategory string `json:"primary_category"`
		ReviewState     string `json:"review_state"`
		Level           string `json:"level"`
		ParentID        *int64 `json:"parent_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if strings.TrimSpace(req.Level) != "" || req.ParentID != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "use hierarchy endpoints for level or parent changes"})
		return
	}

	labelChanged := false
	if label := strings.TrimSpace(req.PreferredLabel); label != "" && label != tag.PreferredLabel {
		// Check for duplicate label (excluding current tag)
		existing, err := h.tagRepo.FindByLabel(c.Request.Context(), label)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if existing != nil && existing.ID != tag.ID {
			c.JSON(http.StatusConflict, gin.H{"error": "标签名已存在"})
			return
		}
		tag.PreferredLabel = label
		tag.Slug = makeSlug(label)
		labelChanged = true
	}
	if category := strings.TrimSpace(req.PrimaryCategory); category != "" {
		tag.PrimaryCategory = category
	}
	if state := strings.TrimSpace(req.ReviewState); state != "" {
		tag.ReviewState = state
	}
	if err := h.tagRepo.Update(c.Request.Context(), tag); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Sync FTS index if label changed
	if labelChanged {
		if err := h.imageTagRepo.SyncFTSForTag(c.Request.Context(), id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "标签更新成功，但FTS索引同步失败: " + err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, tag)
}

func (h *TagHandler) DeleteTag(c *gin.Context) {
	if h.adminSvc == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "tag governance service unavailable"})
		return
	}

	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	preview, err := h.adminSvc.GetDeletePreview(c.Request.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTagNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "resource not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	if !preview.CanDelete {
		blockingReason := preview.BlockingReason
		if strings.TrimSpace(blockingReason) == "" {
			blockingReason = mergeOrReclassifyRequired
		}
		c.JSON(http.StatusConflict, gin.H{
			"error":                "tag is still in use",
			"affected_image_count": preview.AffectedImageCount,
			"blocking_reason":      blockingReason,
		})
		return
	}

	aliases, err := h.aliasRepo.FindByTagID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	for _, alias := range aliases {
		if err := h.aliasRepo.Delete(c.Request.Context(), alias.ID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	if err := h.tagRepo.Delete(c.Request.Context(), id); err != nil {
		respondRepoError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":              true,
		"deleted_tag_id":       id,
		"affected_image_count": int64(0),
	})
}

func (h *TagHandler) GetAliases(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	aliases, err := h.aliasRepo.FindByTagID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"aliases": aliases})
}

func (h *TagHandler) AddAlias(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	if _, err := h.tagRepo.FindByID(c.Request.Context(), id); err != nil {
		respondRepoError(c, err)
		return
	}

	var req struct {
		Label     string `json:"label"`
		AliasType string `json:"alias_type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	alias := &domain.TagAlias{
		TagID:     id,
		Label:     strings.TrimSpace(req.Label),
		AliasType: strings.TrimSpace(req.AliasType),
	}
	if alias.Label == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "label is required"})
		return
	}
	if alias.AliasType == "" {
		alias.AliasType = "synonym"
	}

	if err := h.aliasRepo.Save(c.Request.Context(), alias); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, alias)
}

func (h *TagHandler) DeleteAlias(c *gin.Context) {
	aliasID, ok := parseIDParam(c, "alias_id")
	if !ok {
		return
	}

	if err := h.aliasRepo.Delete(c.Request.Context(), aliasID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetTagStats handles GET /api/v1/tags/stats
// It returns governance statistics for tags including usage, source, and pending-review counts.
func (h *TagHandler) GetTagStats(c *gin.Context) {
	ctx := c.Request.Context()
	if h.adminSvc != nil {
		total, err := h.tagRepo.Count(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		rows, _, err := h.adminSvc.ListGovernanceTags(ctx, "", total, 0)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		stats := make([]gin.H, 0, len(rows))
		for _, row := range rows {
			stats = append(stats, gin.H{
				"tag_id":                 row.TagID,
				"preferred_label":        row.PreferredLabel,
				"level":                  row.Level,
				"parent_id":              row.ParentID,
				"usage_count":            row.UsageCount,
				"direct_usage_count":     row.DirectUsageCount,
				"tree_usage_count":       row.TreeUsageCount,
				"pending_count":          row.PendingCount,
				"direct_pending_count":   row.DirectPendingCount,
				"tree_pending_count":     row.TreePendingCount,
				"confirmed_count":        row.ConfirmedCount,
				"direct_confirmed_count": row.DirectConfirmedCount,
				"tree_confirmed_count":   row.TreeConfirmedCount,
				"rejected_count":         row.RejectedCount,
				"ai_count":               row.AICount,
				"direct_ai_count":        row.DirectAICount,
				"tree_ai_count":          row.TreeAICount,
				"manual_count":           row.ManualCount,
				"direct_manual_count":    row.DirectManualCount,
				"tree_manual_count":      row.TreeManualCount,
			})
		}
		c.JSON(http.StatusOK, gin.H{"stats": stats})
		return
	}

	// Get all tags
	tags, err := h.tagRepo.FindAll(ctx, 100, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	stats := make([]gin.H, 0, len(tags))
	for _, tag := range tags {
		tagStats, err := h.imageTagRepo.GetTagStats(ctx, tag.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		stats = append(stats, gin.H{
			"tag_id":          tag.ID,
			"preferred_label": tag.PreferredLabel,
			"level":           tag.Level,
			"parent_id":       tag.ParentID,
			"usage_count":     tagStats.UsageCount,
			"pending_count":   tagStats.PendingCount,
			"confirmed_count": tagStats.ConfirmedCount,
			"rejected_count":  tagStats.RejectedCount,
			"ai_count":        tagStats.AICount,
			"manual_count":    tagStats.ManualCount,
		})
	}

	c.JSON(http.StatusOK, gin.H{"stats": stats})
}

func (h *TagHandler) GetGovernanceTags(c *gin.Context) {
	if h.adminSvc == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "tag governance service unavailable"})
		return
	}

	limit := parsePositiveInt(c.DefaultQuery("limit", "20"), 20)
	offset := parsePositiveInt(c.DefaultQuery("offset", "0"), 0)
	search := strings.TrimSpace(c.Query("search"))

	rows, total, err := h.adminSvc.ListGovernanceTags(c.Request.Context(), search, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"rows": rows, "total": total})
}

func (h *TagHandler) MergeTag(c *gin.Context) {
	if h.adminSvc == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "tag governance service unavailable"})
		return
	}

	sourceTagID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	var req struct {
		TargetTagID int64 `json:"target_tag_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if req.TargetTagID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "target_tag_id is required"})
		return
	}

	result, err := h.adminSvc.MergeTags(c.Request.Context(), sourceTagID, req.TargetTagID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrMergeSameSourceTarget):
			c.JSON(http.StatusBadRequest, gin.H{"error": "source and target tags must be different"})
		case errors.Is(err, service.ErrCrossLevelMerge):
			c.JSON(http.StatusBadRequest, gin.H{"error": "source and target tags must be at the same level"})
		case errors.Is(err, service.ErrMergeSourceHasChildren):
			c.JSON(http.StatusBadRequest, gin.H{"error": "source tag has child tags"})
		case errors.Is(err, service.ErrTagNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "resource not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":                     true,
		"source_tag_id":               result.SourceTagID,
		"target_tag_id":               result.TargetTagID,
		"migrated_image_associations": result.MigratedImageAssociations,
		"migrated_aliases":            result.MigratedAliases,
	})
}

func (h *TagHandler) GetDeletePreview(c *gin.Context) {
	if h.adminSvc == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "tag governance service unavailable"})
		return
	}

	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	preview, err := h.adminSvc.GetDeletePreview(c.Request.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTagNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "resource not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, preview)
}

// CleanUnusedTags handles POST /api/v1/tags/batch/cleanup.
func (h *TagHandler) CleanUnusedTags(c *gin.Context) {
	if h.adminSvc == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "tag governance service unavailable"})
		return
	}

	var req struct {
		TagIDs []int64 `json:"tag_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	result, err := h.adminSvc.CleanupUnusedTags(c.Request.Context(), req.TagIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"deleted": result.Deleted,
		"blocked": result.Blocked,
		"failed":  result.Failed,
	})
}

func (h *TagHandler) GetTagTree(c *gin.Context) {
	if h.adminSvc == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "tag governance service unavailable"})
		return
	}
	tree, err := h.adminSvc.GetTagTree(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"tree": tree})
}

func (h *TagHandler) GetParentCandidates(c *gin.Context) {
	if h.adminSvc == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "tag governance service unavailable"})
		return
	}
	level := strings.TrimSpace(c.Query("level"))
	candidates, err := h.adminSvc.GetParentCandidates(c.Request.Context(), level)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"candidates": candidates})
}

func (h *TagHandler) ChangeTagLevel(c *gin.Context) {
	if h.adminSvc == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "tag governance service unavailable"})
		return
	}
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var req struct {
		Level    string `json:"level"`
		ParentID *int64 `json:"parent_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	tag, err := h.adminSvc.ChangeLevel(c.Request.Context(), id, req.Level, req.ParentID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTagNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "resource not found"})
		case errors.Is(err, service.ErrInvalidHierarchy):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, tag)
}

func (h *TagHandler) ReparentTag(c *gin.Context) {
	if h.adminSvc == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "tag governance service unavailable"})
		return
	}
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var req struct {
		ParentID *int64 `json:"parent_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	tag, err := h.adminSvc.ReparentTag(c.Request.Context(), id, req.ParentID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTagNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "resource not found"})
		case errors.Is(err, service.ErrInvalidHierarchy):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, tag)
}

func parseIDParam(c *gin.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid " + name})
		return 0, false
	}
	return id, true
}

func parsePositiveInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return fallback
	}
	if parsed == 0 && fallback > 0 {
		return fallback
	}
	return parsed
}

func paginateTags(tags []*domain.Tag, limit, offset int) []*domain.Tag {
	if offset >= len(tags) {
		return []*domain.Tag{}
	}
	end := len(tags)
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}
	return tags[offset:end]
}

func makeSlug(label string) string {
	label = strings.TrimSpace(strings.ToLower(label))
	label = strings.ReplaceAll(label, " ", "-")
	return label
}

func respondRepoError(c *gin.Context, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "resource not found"})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}
