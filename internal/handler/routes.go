package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
	"github.com/wonichan/acgwarehouse-backend/internal/service"
	"github.com/wonichan/acgwarehouse-backend/internal/worker"
)

type Dependencies struct {
	ImageRepo      repository.ImageRepository
	JobRepo        repository.JobRepository
	TagRepo        repository.TagRepository
	AliasRepo      repository.TagAliasRepository
	ObsRepo        repository.TagObservationRepository
	ImageTagRepo   repository.ImageTagRepository
	DuplicateRepo  repository.DuplicateRepository
	SearchRepo     repository.SearchRepository
	GovernanceSvc  *service.TagGovernanceService
	DuplicateSvc   *service.DuplicateService
	SearchSvc      *service.SearchService
	HashSvc        *service.HashService
	JobManager     *worker.Manager
	AITagProcessor gin.HandlerFunc
}

// SetupRoutes registers all HTTP routes.
func SetupRoutes(r *gin.Engine, depsOpt ...*Dependencies) {
	r.GET("/health", HealthCheck)
	r.GET("/ready", ReadyCheck)

	var deps *Dependencies
	if len(depsOpt) > 0 {
		deps = depsOpt[0]
	}

	api := r.Group("/api/v1")

	images := api.Group("/images")
	imageList := gin.HandlerFunc(placeholderHandler)
	imageGet := gin.HandlerFunc(placeholderHandler)
	if deps != nil && deps.ImageRepo != nil && deps.TagRepo != nil && deps.ImageTagRepo != nil {
		imageHandler := NewImageHandler(deps.ImageRepo, deps.TagRepo, deps.ImageTagRepo)
		imageList = imageHandler.ListImages
		imageGet = imageHandler.GetImage
	}
	images.GET("", imageList)
	images.GET("/:id", imageGet)
	images.POST("/scan", placeholderHandler)

	tagGet := gin.HandlerFunc(placeholderHandler)
	tagCreate := gin.HandlerFunc(placeholderHandler)
	tagUpdate := gin.HandlerFunc(placeholderHandler)
	tagDelete := gin.HandlerFunc(placeholderHandler)
	tagGetAliases := gin.HandlerFunc(placeholderHandler)
	tagAddAlias := gin.HandlerFunc(placeholderHandler)
	tagDeleteAlias := gin.HandlerFunc(placeholderHandler)
	tagGetStats := gin.HandlerFunc(placeholderHandler)
	if deps != nil && deps.TagRepo != nil && deps.AliasRepo != nil && deps.ImageTagRepo != nil {
		tagHandler := NewTagHandler(deps.TagRepo, deps.AliasRepo, deps.ImageTagRepo)
		tagGet = tagHandler.GetTags
		tagCreate = tagHandler.CreateTag
		tagUpdate = tagHandler.UpdateTag
		tagDelete = tagHandler.DeleteTag
		tagGetAliases = tagHandler.GetAliases
		tagAddAlias = tagHandler.AddAlias
		tagDeleteAlias = tagHandler.DeleteAlias
		tagGetStats = tagHandler.GetTagStats
	}
	api.GET("/tags", tagGet)
	api.POST("/tags", tagCreate)
	api.PUT("/tags/:id", tagUpdate)
	api.DELETE("/tags/:id", tagDelete)
	api.GET("/tags/:id/aliases", tagGetAliases)
	api.POST("/tags/:id/aliases", tagAddAlias)
	api.DELETE("/tags/:id/aliases/:alias_id", tagDeleteAlias)
	api.GET("/tags/stats", tagGetStats)

	imageTagGet := gin.HandlerFunc(placeholderHandler)
	imageTagAdd := gin.HandlerFunc(placeholderHandler)
	imageTagRemove := gin.HandlerFunc(placeholderHandler)
	imageTagReview := gin.HandlerFunc(placeholderHandler)
	imageTagBatchReview := gin.HandlerFunc(placeholderHandler)
	imageTagMerge := gin.HandlerFunc(placeholderHandler)
	if deps != nil && deps.ImageTagRepo != nil && deps.TagRepo != nil && deps.ImageRepo != nil && deps.GovernanceSvc != nil {
		imageTagHandler := NewImageTagHandler(deps.ImageTagRepo, deps.TagRepo, deps.ImageRepo, deps.GovernanceSvc)
		imageTagGet = imageTagHandler.GetImageTags
		imageTagAdd = imageTagHandler.AddImageTag
		imageTagRemove = imageTagHandler.RemoveImageTag
		imageTagReview = imageTagHandler.ReviewTag
		imageTagBatchReview = imageTagHandler.BatchReview
		imageTagMerge = imageTagHandler.MergeImageTag
	}
	api.GET("/images/:id/tags", imageTagGet)
	api.POST("/images/:id/tags", imageTagAdd)
	api.DELETE("/images/:id/tags/:tag_id", imageTagRemove)
	api.POST("/images/:id/tags/:tag_id/review", imageTagReview)
	api.POST("/images/:id/tags/batch-review", imageTagBatchReview)
	api.POST("/images/:id/tags/:tag_id/merge", imageTagMerge)

	aiTrigger := gin.HandlerFunc(placeholderHandler)
	aiStatus := gin.HandlerFunc(placeholderHandler)
	aiBatch := gin.HandlerFunc(placeholderHandler)
	if deps != nil && deps.JobManager != nil && deps.ImageRepo != nil && deps.JobRepo != nil {
		aiTagHandler := NewAITagHandler(deps.JobManager, deps.ImageRepo, deps.JobRepo)
		aiTrigger = aiTagHandler.TriggerAITags
		aiStatus = aiTagHandler.GetAITagStatus
		aiBatch = aiTagHandler.BatchTriggerAITags
	}
	api.POST("/images/:id/ai-tags", aiTrigger)
	api.GET("/images/:id/ai-tags/status", aiStatus)
	api.POST("/images/batch-ai-tags", aiBatch)

	// Duplicate detection routes
	duplicateDetect := gin.HandlerFunc(placeholderHandler)
	duplicateList := gin.HandlerFunc(placeholderHandler)
	duplicateGet := gin.HandlerFunc(placeholderHandler)
	duplicateDelete := gin.HandlerFunc(placeholderHandler)
	if deps != nil && deps.DuplicateSvc != nil {
		duplicateHandler := NewDuplicateHandler(deps.DuplicateSvc)
		duplicateDetect = duplicateHandler.DetectDuplicates
		duplicateList = duplicateHandler.ListDuplicates
		duplicateGet = duplicateHandler.GetDuplicate
		duplicateDelete = duplicateHandler.DeleteDuplicate
	}
	api.POST("/duplicates/detect", duplicateDetect)
	api.GET("/duplicates", duplicateList)
	api.GET("/duplicates/:id", duplicateGet)
	api.DELETE("/duplicates/:id", duplicateDelete)

	// Search routes
	searchHandler := gin.HandlerFunc(placeholderHandler)
	searchByFilename := gin.HandlerFunc(placeholderHandler)
	if deps != nil && deps.SearchSvc != nil {
		searchHdlr := NewSearchHandler(deps.SearchSvc)
		searchHandler = searchHdlr.Search
		searchByFilename = searchHdlr.SearchByFilename
	}
	api.GET("/search", searchHandler)
	api.GET("/search/filename", searchByFilename)

	collections := api.Group("/collections")
	collections.GET("", placeholderHandler)
	collections.POST("", placeholderHandler)
}

func placeholderHandler(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "not implemented",
		"hint":  "This endpoint will be implemented in a future phase",
	})
}
