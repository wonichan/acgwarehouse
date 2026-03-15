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
)

type TagHandler struct {
	tagRepo      repository.TagRepository
	aliasRepo    repository.TagAliasRepository
	imageTagRepo repository.ImageTagRepository
}

func NewTagHandler(tagRepo repository.TagRepository, aliasRepo repository.TagAliasRepository, imageTagRepo repository.ImageTagRepository) *TagHandler {
	return &TagHandler{
		tagRepo:      tagRepo,
		aliasRepo:    aliasRepo,
		imageTagRepo: imageTagRepo,
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
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	tag := &domain.Tag{
		PreferredLabel:  strings.TrimSpace(req.PreferredLabel),
		PrimaryCategory: strings.TrimSpace(req.PrimaryCategory),
		Slug:            makeSlug(req.PreferredLabel),
		ReviewState:     "confirmed",
	}
	if tag.PreferredLabel == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "preferred_label is required"})
		return
	}

	if err := h.tagRepo.Save(c.Request.Context(), tag); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if label := strings.TrimSpace(req.PreferredLabel); label != "" {
		tag.PreferredLabel = label
		tag.Slug = makeSlug(label)
	}
	if category := strings.TrimSpace(req.PrimaryCategory); category != "" {
		tag.PrimaryCategory = category
	}
	if state := strings.TrimSpace(req.ReviewState); state != "" {
		tag.ReviewState = state
	}

	if err := h.tagRepo.Save(c.Request.Context(), tag); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tag)
}

func (h *TagHandler) DeleteTag(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	imageTags, err := h.imageTagRepo.FindByTagID(c.Request.Context(), id, 1000, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	for _, imageTag := range imageTags {
		if err := h.imageTagRepo.Delete(c.Request.Context(), imageTag.ImageID, imageTag.TagID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
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

	c.JSON(http.StatusOK, gin.H{"success": true})
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
