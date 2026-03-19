package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/wonichan/acgwarehouse-backend/internal/ai"
	"github.com/wonichan/acgwarehouse-backend/internal/config"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
	"github.com/wonichan/acgwarehouse-backend/internal/service"
	"github.com/wonichan/acgwarehouse-backend/internal/worker"
)

// initRepositories initializes all repositories.
func (a *App) initRepositories() {
	a.imageRepo = repository.NewImageRepository(a.db)
	a.jobRepo = repository.NewJobRepository(a.db)
	a.tagRepo = repository.NewTagRepository(a.db)
	a.aliasRepo = repository.NewTagAliasRepository(a.db)
	a.obsRepo = repository.NewTagObservationRepository(a.db)
	a.imageTagRepo = repository.NewImageTagRepository(a.db)
	a.duplicateRepo = repository.NewDuplicateRepository(a.db)
	a.searchRepo = repository.NewSearchRepository(a.db)
	a.collectionRepo = repository.NewCollectionRepository(a.db)
}

// initServices initializes all services.
func (a *App) initServices() {
	a.governanceSvc = service.NewTagGovernanceService(a.tagRepo, a.aliasRepo, a.obsRepo, a.imageTagRepo)
	a.hashSvc = service.NewHashService()
	a.duplicateSvc = service.NewDuplicateService(a.imageRepo, a.duplicateRepo, a.hashSvc)
	a.searchSvc = service.NewSearchService(a.imageRepo, a.tagRepo, a.searchRepo)
	a.adminSvc = service.NewAdminService(
		a.config,
		a.jobRepo,
		a.imageRepo,
		a.tagRepo,
		a.collectionRepo,
		a.jobManager,
	)
}

// initWorkerManager initializes the worker manager and registers all handlers.
func (a *App) initWorkerManager() error {
	// Create job manager with config
	a.jobManager = worker.NewManagerWithConfig(
		a.jobRepo,
		a.config.WorkerPool.WorkerCount,
		a.config.WorkerPool.QueueSize,
	)
	log.Printf("任务管理器配置: workers=%d, queue_size=%d", a.config.WorkerPool.WorkerCount, a.config.WorkerPool.QueueSize)
	a.jobManager.Start(context.Background())

	// Register config change callback
	a.cfgReloader.OnChange(func(old, new *config.Config) {
		if old.WorkerPool.WorkerCount != new.WorkerPool.WorkerCount {
			a.jobManager.SetWorkerCount(context.Background(), new.WorkerPool.WorkerCount)
		}
	})

	// Register handlers
	a.registerThumbnailHandler()
	a.registerScanHandler()
	a.registerAIHandlers()

	return nil
}

// registerThumbnailHandler registers the thumbnail generation handler.
func (a *App) registerThumbnailHandler() {
	thumbnailSvc := service.NewThumbnailService()
	cosSvc, err := service.NewCOSService(&a.config.COS)
	if err != nil {
		log.Printf("thumbnail job handler not registered: %v", err)
		return
	}

	thumbnailHandler := worker.NewThumbnailHandler(thumbnailSvc, cosSvc, a.imageRepo)
	a.jobManager.RegisterHandler("thumbnail_generate", thumbnailHandler.Handle)

	// Register image_imported handler - auto-triggers thumbnail generation
	a.jobManager.RegisterHandler("image_imported", a.createImageImportedHandler())
	log.Printf("已注册 image_imported 处理器 - 将自动触发缩略图生成")
}

// createImageImportedHandler creates the handler for image_imported events.
func (a *App) createImageImportedHandler() func(ctx context.Context, id int64, payload string) error {
	return func(ctx context.Context, id int64, payload string) error {
		var p struct {
			ImageID  int64  `json:"image_id"`
			Path     string `json:"path"`
			Filename string `json:"filename"`
		}
		if err := json.Unmarshal([]byte(payload), &p); err != nil {
			return fmt.Errorf("解析 image_imported payload 失败: %w", err)
		}

		thumbnailPayload, err := json.Marshal(map[string]interface{}{
			"image_id": p.ImageID,
			"path":     p.Path,
			"filename": p.Filename,
		})
		if err != nil {
			return err
		}

		_, err = a.jobManager.AddJob(ctx, "thumbnail_generate", string(thumbnailPayload))
		if err != nil {
			return fmt.Errorf("添加缩略图生成任务失败: %w", err)
		}

		log.Printf("已为新导入的图片 %d 创建缩略图生成任务", p.ImageID)
		return nil
	}
}

// registerScanHandler registers the manual scan handler.
func (a *App) registerScanHandler() {
	metadataSvc := service.NewMetadataService()
	scannerSvc := service.NewScannerService(metadataSvc, a.imageRepo, a.jobRepo, 4)
	scanHandler := worker.NewScanHandler(scannerSvc, a.config.Storage.ScanRoots)
	a.jobManager.RegisterHandler("manual_scan", scanHandler.Handle)
	log.Printf("已注册 manual_scan 处理器 - 支持手动触发扫描任务")
}

// registerAIHandlers registers AI-related handlers if configured.
func (a *App) registerAIHandlers() {
	provider, err := ai.NewProvider(&a.config.AI)
	if err != nil {
		log.Printf("AI provider not configured for background processing: %v", err)
		return
	}

	// 初始化 AI 标签生成并发控制器
	worker.InitAITagConcurrencyLimiter(a.config.AI.MaxConcurrency)

	client := ai.NewRateLimitedClient(provider, a.config.AI.RequestsPerMinute)
	worker.RegisterAITagHandler(a.jobManager, client, a.obsRepo, a.governanceSvc)
}

// SetupGinMode sets the Gin mode based on environment.
func SetupGinMode(cfg *config.Config) {
	if strings.EqualFold(cfg.Server.Env, "production") {
		gin.SetMode(gin.ReleaseMode)
	}
}
