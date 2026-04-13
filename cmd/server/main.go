package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/app"
	"github.com/wonichan/acgwarehouse-backend/internal/logger"
)

func main() {
	// Create application
	application, err := app.New("config.yaml")
	if err != nil {
		logger.Fatalf("failed to create application: %v", err)
	}

	// Setup graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		logger.Info("Shutting down...")

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		if err := application.Shutdown(shutdownCtx); err != nil {
			logger.Errorf("Shutdown error: %v", err)
		}
	}()

	// Run the application
	if err := application.Run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Fatalf("server error: %v", err)
	}
}
