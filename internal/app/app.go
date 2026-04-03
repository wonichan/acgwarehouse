package app

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/wonichan/acgwarehouse-backend/internal/config"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/handler"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
	"github.com/wonichan/acgwarehouse-backend/internal/service"
	"github.com/wonichan/acgwarehouse-backend/internal/sidecar"
	"github.com/wonichan/acgwarehouse-backend/internal/sqliteutil"
	"github.com/wonichan/acgwarehouse-backend/internal/worker"
)

type sidecarRuntimeLifecycle interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	State() sidecar.State
	Status() sidecar.Status
}

type appSidecarMode string

const (
	sidecarModeNotConfigured appSidecarMode = "not_configured"
	sidecarModeReady         appSidecarMode = "ready"
	sidecarModeDegraded      appSidecarMode = "degraded"
)

type appSidecarStatusProvider struct {
	app *App
}

func (p appSidecarStatusProvider) SidecarStatus(context.Context) service.SidecarStatusSnapshot {
	if p.app == nil {
		return service.SidecarStatusSnapshot{}
	}
	return p.app.currentSidecarStatusSnapshot()
}

// App represents the application with all its dependencies and lifecycle management.
type App struct {
	config      *config.Config
	cfgReloader *config.Reloader
	db          *sql.DB
	httpServer  *http.Server
	jobManager  *worker.Manager

	// Repositories
	imageRepo      repository.ImageRepository
	jobRepo        repository.JobRepository
	tagRepo        repository.TagRepository
	aliasRepo      repository.TagAliasRepository
	obsRepo        repository.TagObservationRepository
	imageTagRepo   repository.ImageTagRepository
	duplicateRepo  repository.DuplicateRepository
	searchRepo     repository.SearchRepository
	collectionRepo repository.CollectionRepository

	// Services
	governanceSvc *service.TagGovernanceService
	hashSvc       *service.HashService
	duplicateSvc  *service.DuplicateService
	searchSvc     *service.SearchService
	adminSvc      *service.AdminService
	autoScheduler *service.AITagAutoScheduler

	// Background task control
	refillStopMu         sync.Mutex
	refillStopCh         chan struct{}
	shutdownOnce         sync.Once
	autoSchedulerMu      sync.Mutex
	autoSchedulerControl autoSchedulerLifecycle
	newAutoScheduler     func(*config.Config) autoSchedulerLifecycle
	autoSchedulerStarted bool
	runtimeManifestPath  string
	sidecarRuntime       sidecarRuntimeLifecycle
	sidecarMode          appSidecarMode
	fullyReady           bool
}

// New creates a new App instance with all dependencies initialized.
func New(cfgPath string) (*App, error) {
	app := &App{
		refillStopCh:     make(chan struct{}),
		newAutoScheduler: nil,
		sidecarMode:      sidecarModeNotConfigured,
	}

	// Load configuration
	cfgReloader, err := config.NewReloader(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	app.cfgReloader = cfgReloader
	app.config = cfgReloader.Get()

	// Initialize database
	db, err := openDatabase(app.config)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	app.db = db

	// Ensure database schema
	if err := repository.EnsureScanSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ensure scan schema: %w", err)
	}

	// Initialize repositories
	app.initRepositories()

	// Initialize services
	app.initServices()
	app.initSidecarRuntime()
	app.initAutoScheduler(app.config)

	// Initialize worker manager
	if err := app.initWorkerManager(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize worker manager: %w", err)
	}

	app.adminSvc = service.NewAdminService(
		app.config,
		app.jobRepo,
		app.imageRepo,
		app.tagRepo,
		app.collectionRepo,
		app.jobManager,
		service.NewTaskReadService(repository.NewTaskBatchReadRepository(app.db)),
		repository.NewTaskBatchRepository(app.db),
		repository.NewPlatformTaskRepository(app.db),
		appSidecarStatusProvider{app: app},
	)

	return app, nil
}

// Run starts the HTTP server and blocks until the server stops.
func (a *App) Run() error {
	// Setup config hot reload
	if err := a.cfgReloader.Start(); err != nil {
		log.Printf("配置热重载启动失败: %v", err)
	}
	a.cfgReloader.OnChange(func(old, new *config.Config) {
		a.handleAutoSchedulerConfigChange(old, new)
	})
	// Start job recovery in background
	go a.recoverJobs()

	// Start refill loop in background
	go a.runRefillLoop()
	a.startAutoScheduler()
	if err := a.prepareSidecarStartup(context.Background()); err != nil {
		return err
	}
	// Setup HTTP routes
	r := gin.New()
	r.Use(gin.Recovery())

	handler.SetupRoutes(r, &handler.Dependencies{
		ImageRepo:      a.imageRepo,
		JobRepo:        a.jobRepo,
		TagRepo:        a.tagRepo,
		AliasRepo:      a.aliasRepo,
		ObsRepo:        a.obsRepo,
		ImageTagRepo:   a.imageTagRepo,
		DuplicateRepo:  a.duplicateRepo,
		SearchRepo:     a.searchRepo,
		CollectionRepo: a.collectionRepo,
		GovernanceSvc:  a.governanceSvc,
		DuplicateSvc:   a.duplicateSvc,
		SearchSvc:      a.searchSvc,
		HashSvc:        a.hashSvc,
		JobManager:     a.jobManager,
		AdminSvc:       a.adminSvc,
		AdminCfg:       a.cfgReloader.Get(),
		ConfigReloader: a.cfgReloader,
		DB:             a.db,
	})

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", a.config.Server.Host, a.config.Server.Port)
	a.httpServer = &http.Server{
		Addr:    addr,
		Handler: r,
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen server address: %w", err)
	}

	manifestBaseURL, err := ResolveRuntimeManifestBaseURL(listener.Addr(), a.config.Server.Host)
	if err != nil {
		_ = listener.Close()
		return err
	}
	payload, err := BuildRuntimeManifestPayload(manifestBaseURL, time.Now().UTC())
	if err != nil {
		_ = listener.Close()
		return err
	}

	a.runtimeManifestPath = ResolveRuntimeManifestPath()
	if err := WriteRuntimeManifestAtomic(a.runtimeManifestPath, payload); err != nil {
		_ = listener.Close()
		return err
	}

	log.Printf("ACGWarehouse server starting on %s", manifestBaseURL)
	log.Printf("runtime manifest generated at %s", a.runtimeManifestPath)
	return a.httpServer.Serve(listener)
}

// Shutdown gracefully stops the application.
func (a *App) Shutdown(ctx context.Context) error {
	var shutdownErr error
	a.shutdownOnce.Do(func() {
		a.stopAutoScheduler()

		// Stop refill loop
		a.refillStopMu.Lock()
		close(a.refillStopCh)
		a.refillStopMu.Unlock()

		// Stop config reloader
		if a.cfgReloader != nil {
			a.cfgReloader.Stop()
		}

		// Stop job manager
		if a.jobManager != nil {
			a.jobManager.Stop()
		}

		if a.sidecarRuntime != nil {
			if err := a.sidecarRuntime.Stop(ctx); err != nil {
				shutdownErr = err
				return
			}
		}

		if err := RemoveRuntimeManifest(a.runtimeManifestPath); err != nil {
			log.Printf("runtime manifest cleanup failed: %v", err)
		}

		// Shutdown HTTP server
		if a.httpServer != nil {
			if err := a.httpServer.Shutdown(ctx); err != nil {
				shutdownErr = err
				return
			}
		}

		// Close database
		if a.db != nil {
			shutdownErr = a.db.Close()
		}
	})
	return shutdownErr
}

func (a *App) prepareSidecarStartup(ctx context.Context) error {
	if a.sidecarRuntime == nil {
		a.sidecarMode = sidecarModeNotConfigured
		a.fullyReady = true
		return nil
	}

	if err := a.sidecarRuntime.Start(ctx); err != nil {
		a.sidecarMode = sidecarModeDegraded
		a.fullyReady = false
		log.Printf("sidecar startup degraded: %v", err)
		return nil
	}

	if a.sidecarRuntime.State() == sidecar.StateReady {
		a.sidecarMode = sidecarModeReady
		a.fullyReady = true
		return nil
	}

	a.sidecarMode = sidecarModeDegraded
	a.fullyReady = false
	return nil
}

func (a *App) currentSidecarStatusSnapshot() service.SidecarStatusSnapshot {
	now := time.Now().UTC()
	if a.sidecarRuntime == nil {
		return service.SidecarStatusSnapshot{
			State:            string(sidecarModeNotConfigured),
			LastProbeAt:      now,
			LastProbeResult:  "unknown",
			LastErrorSummary: "not configured",
		}
	}

	status := a.sidecarRuntime.Status()
	state := string(status.State)
	if state == "" {
		state = string(a.sidecarMode)
	}

	probeResult := "unknown"
	switch status.State {
	case sidecar.StateReady:
		probeResult = "ok"
	case sidecar.StateDegraded:
		probeResult = "failed"
	case sidecar.StateStopped:
		probeResult = "failed"
	case sidecar.StateStarting:
		probeResult = "starting"
	}

	lastError := status.LastError
	if probeResult == "failed" && lastError == "" {
		lastError = "sidecar unavailable"
	}

	return service.SidecarStatusSnapshot{
		State:            state,
		LastProbeAt:      now,
		LastProbeResult:  probeResult,
		LastErrorSummary: lastError,
	}
}

// runRefillLoop periodically checks for ready jobs and loads them into the queue.
func (a *App) runRefillLoop() {
	for {
		cfg := a.cfgReloader.Get()
		refillInterval := time.Duration(cfg.WorkerPool.RefillIntervalSeconds) * time.Second
		refillThreshold := int(float64(cfg.WorkerPool.QueueSize) * cfg.WorkerPool.RefillThreshold)

		select {
		case <-a.refillStopCh:
			return
		case <-time.After(refillInterval):
			if a.jobManager.QueueSize() < refillThreshold {
				a.refillReadyJobs()
			}
		}
	}
}

func (a *App) refillReadyJobs() int {
	if a == nil || a.jobRepo == nil || a.jobManager == nil || a.cfgReloader == nil {
		return 0
	}

	jobs, err := a.jobRepo.FindByStatus("ready")
	if err != nil {
		return 0
	}

	cfg := a.cfgReloader.Get()
	maxToLoad := 0
	if cfg != nil {
		maxToLoad = cfg.WorkerPool.RefillBatchSize
		if maxToLoad <= 0 {
			maxToLoad = cfg.WorkerPool.QueueSize
		}
	}
	aiQueueLimit := service.ResolveAITagQueueLimit(cfg)

	loaded := 0
	for i := range jobs {
		if maxToLoad > 0 && loaded >= maxToLoad {
			break
		}
		if jobs[i].Type == domain.PlatformTaskTypeAITagGeneration && a.jobManager.QueuedByType(domain.PlatformTaskTypeAITagGeneration) >= aiQueueLimit {
			continue
		}
		if a.jobManager.LoadExistingJob(&jobs[i]) {
			loaded++
		} else {
			break
		}
	}
	if loaded > 0 {
		log.Printf("后台补充加载了 %d 个任务", loaded)
	}
	return loaded
}

// recoverJobs recovers jobs from database on startup.
func (a *App) recoverJobs() {
	// Reset running jobs to ready (handle abnormal restart)
	if count, err := a.jobRepo.ResetRunningToReady(); err != nil {
		log.Printf("重置 running 任务失败: %v", err)
	} else if count > 0 {
		log.Printf("已重置 %d 个 running 状态的任务为 ready", count)
	}

	// Load ready jobs into queue
	jobs, err := a.jobRepo.FindByStatus("ready")
	if err != nil {
		log.Printf("加载待处理任务失败: %v", err)
		return
	}
	if len(jobs) > 0 {
		log.Printf("发现 %d 个待处理任务，正在加载到队列...", len(jobs))
		loadedCount := 0
		skippedCount := 0
		for i := range jobs {
			if a.jobManager.LoadExistingJob(&jobs[i]) {
				loadedCount++
			} else {
				skippedCount++
			}
		}
		log.Printf("任务加载完成，已加载 %d 个，跳过 %d 个（将在队列空闲时自动加载）", loadedCount, skippedCount)
	}
}

func openDatabase(cfg *config.Config) (*sql.DB, error) {
	return sqliteutil.Open(cfg)
}
