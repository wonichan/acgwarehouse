package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wonichan/acgwarehouse-backend/internal/ai"
	"github.com/wonichan/acgwarehouse-backend/internal/config"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
	"github.com/wonichan/acgwarehouse-backend/internal/service"
	"github.com/wonichan/acgwarehouse-backend/internal/sidecar"
	"github.com/wonichan/acgwarehouse-backend/internal/worker"
)

const (
	sidecarExecutableEnv = "ACG_SIDECAR_EXECUTABLE"
	sidecarPortEnv       = "ACG_SIDECAR_PORT"
	diagnosticsDirEnv    = "ACG_DIAGNOSTICS_DIR"
	logsDirEnv           = "ACG_LOGS_DIR"
	defaultSidecarHost   = "127.0.0.1"
	defaultSidecarPort   = "8000"
)

var (
	newSidecarRuntime = func(cfg sidecar.RuntimeConfig) sidecarRuntimeLifecycle {
		return sidecar.NewRuntime(cfg)
	}
	sidecarHTTPDo = func(req *http.Request) (*http.Response, error) {
		return http.DefaultClient.Do(req)
	}
	startSidecarProcess = func(ctx context.Context, executable string, args []string, logPath string) (sidecar.Process, error) {
		cmd := exec.CommandContext(ctx, executable, args...)
		if logPath != "" {
			if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
				return nil, fmt.Errorf("create sidecar log directory: %w", err)
			}
			file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
			if err != nil {
				return nil, fmt.Errorf("open sidecar log file: %w", err)
			}
			cmd.Stdout = file
			cmd.Stderr = file
		}
		if err := cmd.Start(); err != nil {
			if file, ok := cmd.Stdout.(*os.File); ok {
				_ = file.Close()
			}
			return nil, err
		}
		return &sidecarCmdProcess{cmd: cmd}, nil
	}
)

type sidecarBootstrapSettings struct {
	executable     string
	args           []string
	baseURL        string
	diagnosticPath string
	logPaths       []string
	sidecarLogPath string
}

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
	a.duplicateRepo = repository.NewDuplicateRepository(a.db)
	a.searchRepo = repository.NewSearchRepository(a.db)
	a.collectionRepo = repository.NewCollectionRepository(a.db)
}

// initServices initializes all services.
func (a *App) initServices() {
	a.governanceSvc = service.NewTagGovernanceService(a.tagRepo, a.aliasRepo, a.obsRepo, a.imageTagRepo)
	a.sidecarClient = sidecar.NewSidecarClient(a.sidecarBaseURL)
	a.duplicateSvc = service.NewDuplicateService(a.imageRepo, a.duplicateRepo, a.sidecarClient, unwrapRuntime(a.sidecarRuntime))
	a.searchSvc = service.NewSearchService(a.imageRepo, a.tagRepo, a.searchRepo)
}

func (a *App) initSidecarRuntime() {
	if a.sidecarRuntime != nil {
		return
	}
	settings, err := resolveSidecarBootstrapSettings()
	if err != nil {
		log.Printf("sidecar bootstrap configuration invalid: %v", err)
		settings = fallbackSidecarBootstrapSettings()
	}

	a.sidecarBaseURL = settings.baseURL
	a.sidecarRuntime = newSidecarRuntime(sidecar.RuntimeConfig{
		StartupTimeout: 2 * time.Second,
		ProbeInterval:  100 * time.Millisecond,
		CommandFactory: func(ctx context.Context) (sidecar.Process, error) {
			proc, err := startSidecarProcess(ctx, settings.executable, settings.args, settings.sidecarLogPath)
			if err != nil {
				if settings.diagnosticPath != "" {
					if writeErr := WriteStartupDiagnostic(settings.diagnosticPath, "python", err.Error(), settings.logPaths); writeErr != nil {
						log.Printf("write startup diagnostic failed: %v", writeErr)
					}
				}
				return nil, err
			}
			return proc, nil
		},
		Probe: func(ctx context.Context) error {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.sidecarBaseURL+"/health", nil)
			if err != nil {
				return err
			}
			resp, err := sidecarHTTPDo(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("sidecar probe status: %d", resp.StatusCode)
			}
			return nil
		},
		ShutdownProbe: func(context.Context) error {
			return nil
		},
	})
}

func fallbackSidecarBootstrapSettings() sidecarBootstrapSettings {
	return sidecarBootstrapSettings{
		executable: "python",
		args:       []string{"services/python-sidecar/main.py", "--host", defaultSidecarHost, "--port", defaultSidecarPort},
		baseURL:    fmt.Sprintf("http://%s:%s", defaultSidecarHost, defaultSidecarPort),
	}
}

func resolveSidecarBootstrapSettings() (sidecarBootstrapSettings, error) {
	settings := fallbackSidecarBootstrapSettings()
	port := strings.TrimSpace(os.Getenv(sidecarPortEnv))
	if port == "" {
		port = defaultSidecarPort
	}
	settings.baseURL = fmt.Sprintf("http://%s:%s", defaultSidecarHost, port)

	paths := ResolveLogSourcePaths()
	runtimeRoot := strings.TrimSpace(os.Getenv(portableRuntimeRootEnv))
	diagnosticsDir := strings.TrimSpace(os.Getenv(diagnosticsDirEnv))
	if runtimeRoot != "" {
		layout := resolvePortableRuntimeLayoutRoot(runtimeRoot)
		if diagnosticsDir == "" {
			diagnosticsDir = layout.DiagnosticsDir
		}
	}
	if diagnosticsDir != "" {
		settings.diagnosticPath = filepath.Join(diagnosticsDir, startupDiagnosticFileName)
	}
	if paths.GoLogPath != "" || paths.SidecarLogPath != "" {
		settings.sidecarLogPath = paths.SidecarLogPath
		settings.logPaths = []string{paths.GoLogPath, paths.SidecarLogPath}
	}

	configuredExecutable := strings.TrimSpace(os.Getenv(sidecarExecutableEnv))
	if configuredExecutable == "" {
		settings.args = []string{"services/python-sidecar/main.py", "--host", defaultSidecarHost, "--port", port}
		return settings, nil
	}

	if !filepath.IsAbs(configuredExecutable) && runtimeRoot != "" {
		configuredExecutable = filepath.Join(runtimeRoot, configuredExecutable)
	}
	configuredExecutable = filepath.Clean(configuredExecutable)
	if _, err := os.Stat(configuredExecutable); err != nil {
		return sidecarBootstrapSettings{}, fmt.Errorf("stat sidecar executable: %w", err)
	}

	settings.executable = configuredExecutable
	settings.args = []string{"--host", defaultSidecarHost, "--port", port}
	return settings, nil
}

type sidecarCmdProcess struct {
	cmd *exec.Cmd
}

func (p *sidecarCmdProcess) Kill() error {
	if p == nil || p.cmd == nil || p.cmd.Process == nil {
		return nil
	}
	return p.cmd.Process.Kill()
}

func (p *sidecarCmdProcess) Wait() error {
	if p == nil || p.cmd == nil {
		return nil
	}
	if p.cmd.Stdout != nil {
		defer closeSidecarLogWriter(p.cmd.Stdout)
	}
	if p.cmd.Stderr != nil && p.cmd.Stderr != p.cmd.Stdout {
		defer closeSidecarLogWriter(p.cmd.Stderr)
	}
	return p.cmd.Wait()
}

func closeSidecarLogWriter(writer io.Writer) {
	closer, ok := writer.(io.Closer)
	if !ok {
		return
	}
	_ = closer.Close()
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

	client := ai.NewRateLimitedClient(provider, a.config.AI.RequestsPerMinute)
	aiHandler := worker.NewAITagJobHandler(client, a.obsRepo, a.governanceSvc, a.imageTagRepo)
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
