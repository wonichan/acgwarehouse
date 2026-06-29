// Package main starts the production Vue SPA static server and API reverse proxy.
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	pkgerrors "github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/yachiyo/acgwarehouse/internal/frontendserver"
	"github.com/yachiyo/acgwarehouse/pkg/logger"
)

const (
	defaultFrontendPort        = "2017"
	defaultFrontendDist        = "/opt/acgwarehouse/frontend/vue-gallery/dist"
	defaultBackendURL          = "http://127.0.0.1:2018"
	defaultMaxRequestBodyBytes = "1048576"
	shutdownTimeout            = 10 * time.Second
)

// main 启动前端静态服务与 API 反向代理。
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := run(ctx); err != nil {
		logger.Error(ctx, "frontend server stopped with error", zap.Error(err))
		os.Exit(1)
	}
}

// run 创建 HTTP 服务并在收到退出信号时优雅关闭。
func run(ctx context.Context) error {
	maxRequestBodyBytes, err := envInt64("MAX_REQUEST_BODY_BYTES", defaultMaxRequestBodyBytes)
	if err != nil {
		return pkgerrors.WithMessage(err, "load max request body bytes")
	}
	handler, err := frontendserver.NewHandler(frontendserver.Config{
		DistDir:             envString("FRONTEND_DIST", defaultFrontendDist),
		BackendURL:          envString("BACKEND_URL", defaultBackendURL),
		MaxRequestBodyBytes: maxRequestBodyBytes,
	})
	if err != nil {
		return pkgerrors.WithMessage(err, "create frontend handler")
	}
	server := &http.Server{
		Addr:              ":" + envString("FRONTEND_PORT", defaultFrontendPort),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		logger.Info(ctx, "frontend server starting", zap.String("addr", server.Addr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

// envString 读取环境变量，空值时返回默认值。
func envString(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

// envInt64 读取整数环境变量，空值时使用默认值。
func envInt64(key string, fallback string) (int64, error) {
	value := envString(key, fallback)
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, err
	}
	if parsed < 1 {
		return 0, errors.New("value must be positive")
	}
	return parsed, nil
}
