package handler

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wonichan/acgwarehouse-backend/internal/config"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
	"github.com/wonichan/acgwarehouse-backend/internal/service"
	"github.com/wonichan/acgwarehouse-backend/internal/sidecar"
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
	CollectionRepo repository.CollectionRepository
	GovernanceSvc  *service.TagGovernanceService
	DuplicateSvc   *service.DuplicateService
	SearchSvc      *service.SearchService
	SidecarRuntime *sidecar.Runtime
	CollectionSvc  *service.CollectionService
	BatchSvc       *service.BatchService
	AdminSvc       AdminServiceInterface
	JobManager     *worker.Manager
	AdminCfg       *config.Config
	ConfigReloader *config.Reloader // For hot-reloadable config access
	AITagProcessor gin.HandlerFunc
	DB             *sql.DB // for FTS rebuild and other direct DB operations
}

// SetupRoutes registers all HTTP routes.
func SetupRoutes(r *gin.Engine, depsOpt ...*Dependencies) {
	r.GET("/health", HealthCheck)
	r.GET("/ready", ReadyCheck)

	var deps *Dependencies
	if len(depsOpt) > 0 {
		deps = depsOpt[0]
	}
	configProvider := func() *config.Config {
		if deps == nil {
			return nil
		}
		if deps.ConfigReloader != nil {
			return deps.ConfigReloader.Get()
		}
		return deps.AdminCfg
	}

	api := r.Group("/api/v1")

	// Admin routes - protected with Basic Auth
	var adminHandler *AdminHandler
	if deps != nil && deps.AdminSvc != nil && deps.AdminCfg != nil {
		// Create admin handler; wire backfill service if image repo and DB are available
		if deps.ImageRepo != nil && deps.JobRepo != nil && deps.DB != nil {
			taskRepo := repository.NewPlatformTaskRepository(deps.DB)
			batchRepo := repository.NewTaskBatchRepository(deps.DB)
			taskPlatformSvc := service.NewTaskPlatformService(batchRepo, taskRepo, deps.JobRepo)
			backfillSvc := service.NewAIBackfillService(deps.ImageRepo, taskPlatformSvc, deps.JobManager, configProvider)
			adminHandler = NewAdminHandlerWithBackfill(deps.AdminCfg, deps.AdminSvc, backfillSvc)
		} else {
			adminHandler = NewAdminHandler(deps.AdminCfg, deps.AdminSvc)
		}
	}

	admin := r.Group("/admin/api")
	if adminHandler != nil {
		admin.Use(adminHandler.AuthMiddleware())
		{
			admin.GET("/summary", adminHandler.GetSummary)
			admin.GET("/task-platform/overview", adminHandler.GetTaskPlatformOverview)
			admin.GET("/jobs", adminHandler.GetJobs)
			admin.GET("/task-batches", adminHandler.GetTaskBatches)
			admin.GET("/tasks", adminHandler.GetTasks)
			admin.POST("/actions/scan", adminHandler.TriggerScan)
			admin.POST("/actions/jobs/pause", adminHandler.PauseBackgroundTasks)
			admin.POST("/actions/jobs/resume", adminHandler.ResumeBackgroundTasks)
			admin.POST("/actions/jobs/clear-queue", adminHandler.ClearTaskQueue)
			admin.POST("/actions/jobs/retry-failed", adminHandler.RetryFailedJobs)
			admin.POST("/actions/task-batches/:batch_id/retry-failed", adminHandler.RetryFailedBatchTasks)
			admin.POST("/actions/task-batches/:batch_id/cancel", adminHandler.CancelTaskBatch)
			admin.POST("/actions/tasks/:task_id/cancel", adminHandler.CancelTask)
			admin.POST("/actions/tasks/:task_id/retry-failed", adminHandler.RetryFailedTask)
			// Phase 14: Backfill preview and execute endpoints
			admin.POST("/actions/backfill/preview", adminHandler.BackfillPreview)
			admin.POST("/actions/backfill/execute", adminHandler.BackfillExecute)
			// FTS rebuild endpoint for fixing search index
			if deps != nil && deps.DB != nil {
				admin.POST("/actions/search/rebuild-fts", func(c *gin.Context) {
					if err := repository.RebuildFTSIndex(deps.DB); err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{
							"success": false,
							"error":   "failed to rebuild FTS index: " + err.Error(),
						})
						return
					}
					c.JSON(http.StatusOK, gin.H{
						"success": true,
						"message": "FTS index rebuilt successfully",
					})
				})
			}
		}
	}

	// Redirect /admin to /admin-ui for convenience
	r.GET("/admin", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/admin-ui/")
	})

	// Serve admin static files
	// Using /admin-ui instead of /admin to avoid route conflicts with /admin/api/* and /api/*
	r.Static("/admin-ui", "./web/admin")

	images := api.Group("/images")
	imageList := gin.HandlerFunc(placeholderHandler)
	imageGet := gin.HandlerFunc(placeholderHandler)
	imageScan := gin.HandlerFunc(placeholderHandler)
	if deps != nil && deps.ImageRepo != nil && deps.TagRepo != nil && deps.ImageTagRepo != nil {
		imageHandler := NewImageHandler(deps.ImageRepo, deps.TagRepo, deps.ImageTagRepo, deps.AdminSvc)
		imageList = imageHandler.ListImages
		imageGet = imageHandler.GetImage
		imageScan = imageHandler.TriggerImport
	}
	images.GET("", imageList)
	images.GET("/:id", imageGet)
	images.POST("/scan", imageScan)

	tagGet := gin.HandlerFunc(placeholderHandler)
	tagCreate := gin.HandlerFunc(placeholderHandler)
	tagUpdate := gin.HandlerFunc(placeholderHandler)
	tagDelete := gin.HandlerFunc(placeholderHandler)
	tagGetAliases := gin.HandlerFunc(placeholderHandler)
	tagAddAlias := gin.HandlerFunc(placeholderHandler)
	tagDeleteAlias := gin.HandlerFunc(placeholderHandler)
	tagGetStats := gin.HandlerFunc(placeholderHandler)
	tagGetGovernance := gin.HandlerFunc(placeholderHandler)
	tagGetDeletePreview := gin.HandlerFunc(placeholderHandler)
	tagMerge := gin.HandlerFunc(placeholderHandler)
	tagBatchCleanup := gin.HandlerFunc(placeholderHandler)
	if deps != nil && deps.TagRepo != nil && deps.AliasRepo != nil && deps.ImageTagRepo != nil {
		adminService := service.NewTagAdminService(deps.DB, deps.TagRepo, deps.AliasRepo, deps.ImageTagRepo)
		tagHandler := NewTagHandler(deps.TagRepo, deps.AliasRepo, deps.ImageTagRepo, adminService)
		tagGet = tagHandler.GetTags
		tagCreate = tagHandler.CreateTag
		tagUpdate = tagHandler.UpdateTag
		tagDelete = tagHandler.DeleteTag
		tagGetAliases = tagHandler.GetAliases
		tagAddAlias = tagHandler.AddAlias
		tagDeleteAlias = tagHandler.DeleteAlias
		tagGetStats = tagHandler.GetTagStats
		tagGetGovernance = tagHandler.GetGovernanceTags
		tagGetDeletePreview = tagHandler.GetDeletePreview
		tagMerge = tagHandler.MergeTag
		tagBatchCleanup = tagHandler.CleanUnusedTags
	}
	api.GET("/tags", tagGet)
	api.GET("/tags/governance", tagGetGovernance)
	api.POST("/tags", tagCreate)
	api.PUT("/tags/:id", tagUpdate)
	api.DELETE("/tags/:id", tagDelete)
	api.POST("/tags/:id/merge", tagMerge)
	api.GET("/tags/:id/delete-preview", tagGetDeletePreview)
	api.GET("/tags/:id/aliases", tagGetAliases)
	api.POST("/tags/:id/aliases", tagAddAlias)
	api.DELETE("/tags/:id/aliases/:alias_id", tagDeleteAlias)
	api.GET("/tags/stats", tagGetStats)
	api.POST("/tags/batch/cleanup", tagBatchCleanup)

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
	aiDefaultPrompt := gin.HandlerFunc(placeholderHandler)
	if deps != nil && deps.JobManager != nil && deps.ImageRepo != nil && deps.JobRepo != nil && deps.DB != nil {
		taskRepo := repository.NewPlatformTaskRepository(deps.DB)
		batchRepo := repository.NewTaskBatchRepository(deps.DB)
		taskPlatformSvc := service.NewTaskPlatformService(batchRepo, taskRepo, deps.JobRepo)
		aiTagHandler := NewAITagHandler(deps.JobManager, deps.ImageRepo, deps.JobRepo, taskRepo, taskPlatformSvc, configProvider)
		aiTrigger = aiTagHandler.TriggerAITags
		aiStatus = aiTagHandler.GetAITagStatus
		aiBatch = aiTagHandler.BatchTriggerAITags
		aiDefaultPrompt = aiTagHandler.GetDefaultPrompt
	}
	api.POST("/images/:id/ai-tags", aiTrigger)
	api.GET("/images/:id/ai-tags/status", aiStatus)
	api.POST("/images/batch-ai-tags", aiBatch)
	api.GET("/ai-tags/default-prompt", aiDefaultPrompt)

	// Duplicate detection routes
	duplicateDetect := gin.HandlerFunc(placeholderHandler)
	duplicateList := gin.HandlerFunc(placeholderHandler)
	duplicateGet := gin.HandlerFunc(placeholderHandler)
	duplicateDelete := gin.HandlerFunc(placeholderHandler)
	if deps != nil && deps.DuplicateSvc != nil {
		duplicateHandler := NewDuplicateHandler(deps.DuplicateSvc, deps.SidecarRuntime)
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

	// Collection routes
	collectionList := gin.HandlerFunc(placeholderHandler)
	collectionGet := gin.HandlerFunc(placeholderHandler)
	collectionCreate := gin.HandlerFunc(placeholderHandler)
	collectionUpdate := gin.HandlerFunc(placeholderHandler)
	collectionDelete := gin.HandlerFunc(placeholderHandler)
	collectionAddImage := gin.HandlerFunc(placeholderHandler)
	collectionRemoveImage := gin.HandlerFunc(placeholderHandler)
	collectionSetCover := gin.HandlerFunc(placeholderHandler)
	collectionGetImages := gin.HandlerFunc(placeholderHandler)
	if deps != nil && deps.CollectionSvc != nil {
		collectionHandler := NewCollectionHandler(deps.CollectionSvc)
		collectionList = collectionHandler.ListCollections
		collectionGet = collectionHandler.GetCollection
		collectionCreate = collectionHandler.CreateCollection
		collectionUpdate = collectionHandler.UpdateCollection
		collectionDelete = collectionHandler.DeleteCollection
		collectionAddImage = collectionHandler.AddImageToCollection
		collectionRemoveImage = collectionHandler.RemoveImageFromCollection
		collectionSetCover = collectionHandler.SetCoverImage
		collectionGetImages = collectionHandler.GetCollectionImages
	}
	api.GET("/collections", collectionList)
	api.GET("/collections/:id", collectionGet)
	api.POST("/collections", collectionCreate)
	api.PUT("/collections/:id", collectionUpdate)
	api.DELETE("/collections/:id", collectionDelete)
	api.POST("/collections/:id/images", collectionAddImage)
	api.DELETE("/collections/:id/images/:image_id", collectionRemoveImage)
	api.PUT("/collections/:id/cover", collectionSetCover)
	api.GET("/collections/:id/images", collectionGetImages)

	// Batch operation routes
	batchAddTags := gin.HandlerFunc(placeholderHandler)
	batchRemoveTags := gin.HandlerFunc(placeholderHandler)
	batchMoveToCollection := gin.HandlerFunc(placeholderHandler)
	batchRemoveFromCollection := gin.HandlerFunc(placeholderHandler)
	batchDeleteImages := gin.HandlerFunc(placeholderHandler)
	if deps != nil && deps.BatchSvc != nil {
		batchHandler := NewBatchHandler(deps.BatchSvc)
		batchAddTags = batchHandler.BatchAddTags
		batchRemoveTags = batchHandler.BatchRemoveTags
		batchMoveToCollection = batchHandler.BatchMoveToCollection
		batchRemoveFromCollection = batchHandler.BatchRemoveFromCollection
		batchDeleteImages = batchHandler.BatchDeleteImages
	}
	api.POST("/batch/tags/add", batchAddTags)
	api.POST("/batch/tags/remove", batchRemoveTags)
	api.POST("/batch/collections/move", batchMoveToCollection)
	api.POST("/batch/collections/remove", batchRemoveFromCollection)
	api.POST("/batch/images/delete", batchDeleteImages)
}

func placeholderHandler(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "not implemented",
		"hint":  "This endpoint will be implemented in a future phase",
	})
}
