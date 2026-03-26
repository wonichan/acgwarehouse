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
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
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
}

func (a *App) initAutoScheduler(cfg *config.Config) {
	if cfg == nil {
		return
	}
	a.autoScheduler = service.NewAITagAutoScheduler(a.imageRepo, a.newTaskPlatformService(), cfg)
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
	taskPlatformSvc := a.newTaskPlatformService()
	cosSvc, err := service.NewCOSService(&a.config.COS)
	if err != nil {
		log.Printf("thumbnail job handler not registered: %v", err)
		return
	}

	thumbnailHandler := worker.NewThumbnailHandler(thumbnailSvc, cosSvc, a.imageRepo)
	a.registerPlatformTaskHandler(domain.PlatformTaskTypeThumbnailGenerate, thumbnailHandler.Handle)

	// Register image_imported handler - auto-triggers thumbnail generation
	a.jobManager.RegisterHandler(domain.PlatformTaskTypeImageImported, a.createImageImportedHandler(taskPlatformSvc))
	log.Printf("已注册 image_imported 处理器 - 将自动触发缩略图生成")
}

// createImageImportedHandler creates the handler for image_imported events.
func (a *App) createImageImportedHandler(taskPlatformSvc *service.TaskPlatformService) func(ctx context.Context, id int64, payload string) error {
	return func(ctx context.Context, id int64, payload string) error {
		var p struct {
			ImageID  int64  `json:"image_id"`
			Path     string `json:"path"`
			Filename string `json:"filename"`
		}
		if err := json.Unmarshal([]byte(payload), &p); err != nil {
			return fmt.Errorf("解析 image_imported payload 失败: %w", err)
		}
		if taskPlatformSvc == nil {
			return nil
		}
		image, err := a.imageRepo.FindByID(p.ImageID)
		if err != nil {
			return fmt.Errorf("查询导入图片失败: %w", err)
		}
		plan, err := taskPlatformSvc.PlanBatch(ctx, service.TaskPlatformPlanRequest{
			SourceType:   domain.TaskBatchSourceImportScan,
			SummaryLabel: service.BuildTaskBatchSummaryLabel(domain.TaskBatchSourceImportScan, []string{image.SourceRoot}, 1),
			SourceRoots:  []string{image.SourceRoot},
			TaskTypes:    []string{domain.PlatformTaskTypeThumbnailGenerate},
			Items: []service.TaskPlatformPlanItem{{
				ImageID:          image.ID,
				ImageVersionKey:  service.BuildImageVersionKey(image),
				SourceDescriptor: image.Path,
			}},
		})
		if err != nil {
			return fmt.Errorf("规划导入平台任务失败: %w", err)
		}

		thumbnailPayload, err := json.Marshal(map[string]interface{}{
			"image_id": p.ImageID,
			"path":     p.Path,
			"filename": p.Filename,
		})
		if err != nil {
			return err
		}

		createdJobs := 0
		for i := range plan.CreatedTasks {
			job, err := taskPlatformSvc.QueueTask(ctx, &plan.CreatedTasks[i], domain.PlatformTaskTypeThumbnailGenerate, string(thumbnailPayload))
			if err != nil {
				return fmt.Errorf("添加缩略图生成任务失败: %w", err)
			}
			a.jobManager.LoadExistingJob(job)
			createdJobs++
		}

		if createdJobs == 0 {
			log.Printf("图片 %d 的缩略图平台任务已存在，本次 image_imported 仅保留内部调度记录", p.ImageID)
			return nil
		}

		log.Printf("已为新导入的图片 %d 创建 %d 个缩略图平台任务", p.ImageID, createdJobs)
		return nil
	}
}

// registerScanHandler registers the manual scan handler.
func (a *App) registerScanHandler() {
	metadataSvc := service.NewMetadataService()
	scannerSvc := service.NewScannerService(metadataSvc, a.imageRepo, a.jobRepo, a.newTaskPlatformService(), 4)
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
	aiHandler := worker.NewAITagJobHandler(client, a.obsRepo, a.governanceSvc)
	a.registerPlatformTaskHandler(domain.PlatformTaskTypeAITagGeneration, aiHandler)
}

func (a *App) newTaskPlatformService() *service.TaskPlatformService {
	return service.NewTaskPlatformService(
		repository.NewTaskBatchRepository(a.db),
		repository.NewPlatformTaskRepository(a.db),
		a.jobRepo,
	)
}

func (a *App) registerPlatformTaskHandler(jobType string, handler worker.JobFunc) {
	taskPlatformSvc := a.newTaskPlatformService()
	a.jobManager.RegisterHandler(jobType, func(ctx context.Context, id int64, payload string) error {
		if err := taskPlatformSvc.MarkJobRunning(ctx, id); err != nil {
			return fmt.Errorf("mark platform task running: %w", err)
		}
		if err := handler(ctx, id, payload); err != nil {
			if markErr := taskPlatformSvc.MarkJobFailed(ctx, id, err.Error()); markErr != nil {
				log.Printf("同步平台任务失败状态失败: job=%d err=%v", id, markErr)
			}
			return err
		}
		if err := taskPlatformSvc.MarkJobCompleted(ctx, id); err != nil {
			return fmt.Errorf("mark platform task completed: %w", err)
		}
		return nil
	})
}

// SetupGinMode sets the Gin mode based on environment.
func SetupGinMode(cfg *config.Config) {
	if strings.EqualFold(cfg.Server.Env, "production") {
		gin.SetMode(gin.ReleaseMode)
	}
}
