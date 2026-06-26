package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	pkgerrors "github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/yachiyo/acgwarehouse/internal/conf"
	"github.com/yachiyo/acgwarehouse/internal/handler/router"
	"github.com/yachiyo/acgwarehouse/internal/infra/db"
	"github.com/yachiyo/acgwarehouse/internal/infra/search"
	"github.com/yachiyo/acgwarehouse/internal/repository"
	"github.com/yachiyo/acgwarehouse/internal/service"
	jwtpkg "github.com/yachiyo/acgwarehouse/pkg/jwt"
	"github.com/yachiyo/acgwarehouse/pkg/logger"
)

// shutdownHooks 保存服务优雅关闭阶段的资源钩子。
type shutdownHooks struct {
	flushers []func(context.Context) error
	stoppers []func(context.Context) error
	closers  []func(context.Context) error
}

// main 启动 HTTP 服务并处理进程退出码。
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		logger.Error(ctx, "web server stopped with error", zap.Error(err))
		os.Exit(1)
	}
}

// run 加载配置、初始化资源并启动 Hertz 服务。
func run(ctx context.Context) error {
	cfg, err := conf.Load()
	if err != nil {
		return pkgerrors.WithMessage(err, "load config")
	}

	hooks := newShutdownHooks()
	if err := setupLogger(cfg, hooks); err != nil {
		return err
	}

	sqliteDB, err := db.NewSQLite(cfg.Database)
	if err != nil {
		return pkgerrors.WithMessage(err, "init sqlite")
	}
	addSQLiteClose(hooks, sqliteDB)

	searchIndex, err := search.Open(cfg.Search.BlevePath)
	if err != nil {
		return pkgerrors.WithMessage(err, "open search index")
	}
	addSearchClose(hooks, searchIndex)

	userRepo := repository.NewUserRepository(sqliteDB.Read, sqliteDB.Write)
	userService := service.NewUserService(
		userRepo,
		jwtpkg.NewManager(cfg.Security.JWTSecret, cfg.Security.JWTDuration),
	)
	if err := bootstrapAdmin(ctx, cfg.Admin, userService); err != nil {
		return pkgerrors.WithMessage(err, "bootstrap admin")
	}

	imageRepo := repository.NewImageRepository(sqliteDB.Read, sqliteDB.Write)
	tagRepo := repository.NewTagRepository(sqliteDB.Read, sqliteDB.Write)
	searcher := search.NewSearcher(searchIndex)
	tagService := service.NewTagService(tagRepo, searcher)
	viewBuffer := service.NewViewBuffer(imageRepo, cfg.View.FlushInterval)
	viewBuffer.Start(ctx)
	addViewBufferFlush(hooks, viewBuffer)
	imageService := service.NewImageServiceWithTags(
		imageRepo,
		searcher,
		viewBuffer,
		tagService,
		cfg.COS.Domain,
	)

	engine := router.New(cfg, userService, imageService, tagService)
	engine.SetCustomSignalWaiter(newSignalWaiter(ctx))
	engine.OnShutdown = append(engine.OnShutdown, runShutdownHooks(hooks))

	logger.Info(ctx, "web server starting", zap.String("addr", cfg.Server.Address()))
	engine.Spin()
	return nil
}

// bootstrapAdmin 按环境配置幂等引导首个管理员。
func bootstrapAdmin(ctx context.Context, cfg conf.AdminConfig, userService *service.UserService) error {
	if cfg.Username == "" && cfg.Password == "" {
		return nil
	}
	if cfg.Username == "" || cfg.Password == "" {
		return pkgerrors.New("admin username and password must be configured together")
	}
	if err := userService.EnsureAdmin(ctx, cfg.Username, cfg.Password); err != nil {
		return pkgerrors.WithMessage(err, "ensure admin")
	}
	logger.Info(ctx, "admin bootstrap checked", zap.String("username", cfg.Username))
	return nil
}

// setupLogger 初始化全局日志并注册刷新钩子。
func setupLogger(cfg conf.Config, hooks *shutdownHooks) error {
	zapLogger, err := logger.New(cfg.Log.Level)
	if err != nil {
		return pkgerrors.WithMessage(err, "create logger")
	}
	logger.ReplaceGlobal(zapLogger)
	addLoggerSync(hooks)
	return nil
}

// newShutdownHooks 创建空的生命周期钩子集合。
func newShutdownHooks() *shutdownHooks {
	return &shutdownHooks{}
}

// addCloser 保存退出阶段需要释放的资源处理函数。
func (h *shutdownHooks) addCloser(fn func(context.Context) error) {
	h.closers = append(h.closers, fn)
}

// run 按 flush、stop、close 顺序执行生命周期钩子。
func (h *shutdownHooks) run(ctx context.Context) error {
	if err := runHookGroup(ctx, h.flushers); err != nil {
		return pkgerrors.WithMessage(err, "run flush hooks")
	}
	if err := runHookGroup(ctx, h.stoppers); err != nil {
		return pkgerrors.WithMessage(err, "run stop hooks")
	}
	if err := runHookGroup(ctx, h.closers); err != nil {
		return pkgerrors.WithMessage(err, "run close hooks")
	}
	return nil
}

// newSignalWaiter 将外部 signal.NotifyContext 接入 Hertz 优雅关闭流程。
func newSignalWaiter(ctx context.Context) func(chan error) error {
	return func(errCh chan error) error {
		select {
		case <-ctx.Done():
			return nil
		case err := <-errCh:
			if err != nil {
				return pkgerrors.WithMessage(err, "hertz server")
			}
			return nil
		}
	}
}

// runShutdownHooks 执行服务优雅关闭钩子。
func runShutdownHooks(hooks *shutdownHooks) func(context.Context) {
	return func(ctx context.Context) {
		if err := hooks.run(ctx); err != nil {
			logger.Error(ctx, "shutdown hooks failed", zap.Error(err))
			return
		}
		logger.Info(ctx, "shutdown hooks completed")
	}
}

// addSQLiteClose 将 SQLite 连接池纳入退出释放流程。
func addSQLiteClose(hooks *shutdownHooks, sqliteDB *db.SQLite) {
	hooks.addCloser(func(ctx context.Context) error {
		if err := sqliteDB.Close(); err != nil {
			return pkgerrors.WithMessage(err, "close sqlite")
		}
		logger.Info(ctx, "sqlite closed")
		return nil
	})
}

// addSearchClose 注册搜索索引关闭钩子。
func addSearchClose(hooks *shutdownHooks, index *search.Index) {
	hooks.addCloser(func(ctx context.Context) error {
		if err := index.Close(); err != nil {
			return pkgerrors.WithMessage(err, "close search index")
		}
		logger.Info(ctx, "search index closed")
		return nil
	})
}

// addViewBufferFlush 注册浏览事件缓冲刷新钩子。
func addViewBufferFlush(hooks *shutdownHooks, buffer *service.ViewBuffer) {
	hooks.flushers = append(hooks.flushers, func(ctx context.Context) error {
		if err := buffer.Stop(ctx); err != nil {
			return pkgerrors.WithMessage(err, "stop view buffer")
		}
		logger.Info(ctx, "view buffer flushed")
		return nil
	})
}

// addLoggerSync 将日志缓冲刷新纳入退出释放流程。
func addLoggerSync(hooks *shutdownHooks) {
	hooks.addCloser(func(ctx context.Context) error {
		if err := logger.Sync(); err != nil {
			logger.Warn(ctx, "sync logger failed", zap.Error(err))
		}
		return nil
	})
}

// runHookGroup 按注册顺序执行同一阶段钩子。
func runHookGroup(ctx context.Context, hooks []func(context.Context) error) error {
	for _, hook := range hooks {
		if err := hook(ctx); err != nil {
			return err
		}
	}
	return nil
}
