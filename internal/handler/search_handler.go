package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/wonichan/acgwarehouse-backend/internal/service"
)

// SearchHandler handles search API endpoints.
type SearchHandler struct {
	searchService *service.SearchService
}

// NewSearchHandler creates a new search handler.
func NewSearchHandler(ss *service.SearchService) *SearchHandler {
	return &SearchHandler{
		searchService: ss,
	}
}

// Search handles GET /api/v1/search
// Query params:
//   - q: search query (optional)
//   - tag_ids: comma-separated tag IDs (optional)
//   - sort_by: relevance | created_at | filename | file_size (default: relevance)
//   - sort_order: asc | desc (default: desc)
//   - limit: results per page (default: 20, max: 100)
//   - offset: pagination offset (default: 0)
func (h *SearchHandler) Search(c *gin.Context) {
	// Parse query parameters
	query := strings.TrimSpace(c.Query("q"))
	tagIDsStr := c.Query("tag_ids")
	sortBy := c.DefaultQuery("sort_by", "relevance")
	sortOrder := c.DefaultQuery("sort_order", "desc")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	// Validate and cap limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	// Validate sort_by
	validSortFields := map[string]bool{
		"relevance":  true,
		"created_at": true,
		"filename":   true,
		"file_size":  true,
	}
	if !validSortFields[sortBy] {
		sortBy = "relevance"
	}

	// Validate sort_order
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	// Parse tag IDs
	var tagIDs []int64
	if tagIDsStr != "" {
		parts := strings.Split(tagIDsStr, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			id, err := strconv.ParseInt(p, 10, 64)
			if err == nil {
				tagIDs = append(tagIDs, id)
			}
		}
	}

	// Perform search
	result, err := h.searchService.Search(c.Request.Context(), service.SearchOptions{
		Query:     query,
		TagIDs:    tagIDs,
		SortBy:    sortBy,
		SortOrder: sortOrder,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Search failed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"images":   result.Images,
		"total":    result.Total,
		"has_more": result.HasMore,
	})
}

// SearchByFilename handles GET /api/v1/search/filename
// Query params:
//   - pattern: filename pattern to search
//   - limit: results per page (default: 20)
//   - offset: pagination offset (default: 0)
func (h *SearchHandler) SearchByFilename(c *gin.Context) {
	pattern := strings.TrimSpace(c.Query("pattern"))
	if pattern == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "pattern parameter is required",
		})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	result, err := h.searchService.SearchByFilename(c.Request.Context(), pattern, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Search failed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"images":   result.Images,
		"total":    result.Total,
		"has_more": result.HasMore,
	})
}

// SetupSearchRoutes registers search routes on the given router group.
func (h *SearchHandler) SetupSearchRoutes(rg *gin.RouterGroup) {
	rg.GET("/search", h.Search)
	rg.GET("/search/filename", h.SearchByFilename)
}
