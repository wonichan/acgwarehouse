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

const logsDirEnv = "ACG_LOGS_DIR"

type autoSchedulerLifecycle interface {
	Start(ctx context.Context)
	Stop()
}

// initRepositories initializes all repositories.
func (a *App) initRepositories() {
	a.imageRepo = repository.NewImageRepository(a.db)
	a.jobRepo = repository.NewJobRepository(a.db)
	a.tagRepo = repository.NewTagRepository(a.db)
	a.aliasRepo = repository.NewTagAliasRepository(a.db)
	a.obsRepo = repository.NewTagObservationRepository(a.db)
	a.imageTagRepo = repository.NewImageTagRepository(a.db)
	a.searchRepo = repository.NewSearchRepository(a.db)
	a.collectionRepo = repository.NewCollectionRepository(a.db)
}

// initServices initializes all services.
func (a *App) initServices() {
	a.governanceSvc = service.NewTagGovernanceService(a.tagRepo, a.aliasRepo, a.obsRepo, a.imageTagRepo)
	a.searchSvc = service.NewSearchService(a.imageRepo, a.tagRepo, a.searchRepo)
	a.collectionSvc = service.NewCollectionService(a.collectionRepo)
	a.batchSvc = service.NewBatchService(a.imageRepo, a.tagRepo, a.imageTagRepo, a.collectionRepo)
}

func (a *App) initAutoScheduler(cfg *config.Config) {
	if cfg == nil {
		return
	}
	if a.newAutoScheduler == nil {
		a.newAutoScheduler = func(cfg *config.Config) autoSchedulerLifecycle {
			scheduler := service.NewAITagAutoScheduler(a.imageRepo, a.newTaskPlatformService(), cfg)
			a.autoScheduler = scheduler
			return scheduler
		}
	}
	a.autoSchedulerMu.Lock()
	defer a.autoSchedulerMu.Unlock()
	a.autoSchedulerControl = a.newAutoScheduler(cfg)
	a.autoSchedulerStarted = false
	if scheduler, ok := a.autoSchedulerControl.(*service.AITagAutoScheduler); ok {
		a.autoScheduler = scheduler
	}
}

func (a *App) startAutoScheduler() {
	a.autoSchedulerMu.Lock()
	defer a.autoSchedulerMu.Unlock()
	if a.config == nil || !a.config.AI.AutoAITagOnImport || a.autoSchedulerControl == nil || a.autoSchedulerStarted {
		return
	}
	a.autoSchedulerControl.Start(context.Background())
	a.autoSchedulerStarted = true
	log.Printf("AI 标签自动调度服务已启动，扫描间隔: %d 分钟", a.config.AI.AutoScanIntervalMinutes)
}

func (a *App) stopAutoScheduler() {
	a.autoSchedulerMu.Lock()
	defer a.autoSchedulerMu.Unlock()
	if a.autoSchedulerControl == nil || !a.autoSchedulerStarted {
		return
	}
	a.autoSchedulerControl.Stop()
	a.autoSchedulerStarted = false
	log.Printf("AI 标签自动调度服务已停止")
}

func (a *App) handleAutoSchedulerConfigChange(old, new *config.Config) {
	a.config = new
	if old == nil || new == nil {
		return
	}
	oldAI := old.AI
	newAI := new.AI
	if oldAI.AutoAITagOnImport == newAI.AutoAITagOnImport &&
		oldAI.AutoScanIntervalMinutes == newAI.AutoScanIntervalMinutes &&
		oldAI.AutoScanBatchSize == newAI.AutoScanBatchSize {
		return
	}

	a.autoSchedulerMu.Lock()
	started := a.autoSchedulerStarted
	current := a.autoSchedulerControl
	a.autoSchedulerMu.Unlock()

	if !newAI.AutoAITagOnImport {
		if started && current != nil {
			a.stopAutoScheduler()
		}
		return
	}

	if started && current != nil {
		a.stopAutoScheduler()
	}
	a.initAutoScheduler(new)
	a.startAutoScheduler()
	if started {
		log.Printf("AI 标签自动调度服务已重启，新间隔: %d 分钟", newAI.AutoScanIntervalMinutes)
	}
}

// handleAIConfigChange 处理 AI 配置的热更新
func (a *App) handleAIConfigChange(old, new *config.Config) {
	if old == nil || new == nil {
		return
	}
	oldAI := old.AI
	newAI := new.AI

	if oldAI.MaxConcurrency != newAI.MaxConcurrency {
		worker.SetAITagConcurrencyLimiter(newAI.MaxConcurrency)
	}

	if oldAI.RequestsPerMinute != newAI.RequestsPerMinute {
		if a.aiRateLimitedClient != nil {
			a.aiRateLimitedClient.SetRequestsPerMinute(newAI.RequestsPerMinute)
			log.Printf("AI 请求限流已调整为: %d 请求/分钟", newAI.RequestsPerMinute)
		}
	}

	ai.SetAllowedLocalImageRoots(new.Storage.ScanRoots)
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

		thumbnailPayload, err := json.Marshal(map[string]any{
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
	ai.SetAllowedLocalImageRoots(a.config.Storage.ScanRoots)

	client := ai.NewRateLimitedClient(provider, a.config.AI.RequestsPerMinute)
	a.aiRateLimitedClient = client

	aiHandler := worker.NewBatchAITagJobHandler(a.jobRepo, client, a.obsRepo, a.governanceSvc, a.newTaskPlatformService(), repository.NewPlatformTaskRepository(a.db), a.imageTagRepo, a.config.AI.DoubaoBatchMode)
	aiRegenerationHandler := worker.NewAITagRegenerationJobHandler(client, a.obsRepo, a.governanceSvc)
	a.registerPlatformTaskHandler(domain.PlatformTaskTypeAITagGeneration, aiHandler)
	a.registerPlatformTaskHandler(domain.PlatformTaskTypeAITagRegeneration, aiRegenerationHandler)
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
