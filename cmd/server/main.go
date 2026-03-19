package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/app"
)

func main() {
	// Create application
	application, err := app.New("config.yaml")
	if err != nil {
		log.Fatalf("failed to create application: %v", err)
	}

	// Setup graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("Shutting down...")

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		if err := application.Shutdown(shutdownCtx); err != nil {
			log.Printf("Shutdown error: %v", err)
		}
	}()

	// Run the application
	if err := application.Run(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
