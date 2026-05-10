package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"strings"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"

	"github.com/wonichan/acgwarehouse-backend/internal/config"
	"github.com/wonichan/acgwarehouse-backend/internal/d1client"
	"github.com/wonichan/acgwarehouse-backend/internal/logger"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
	"github.com/wonichan/acgwarehouse-backend/internal/service"
	"github.com/wonichan/acgwarehouse-backend/internal/sqliteutil"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to config file")
	scanPath := flag.String("path", "", "Path to scan")
	workers := flag.Int("workers", 1, "Number of worker goroutines")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		logger.Fatalf("failed to load config: %v", err)
	}

	paths := cfg.Storage.ScanRoots
	if *scanPath != "" {
		paths = []string{*scanPath}
	}
	if len(paths) == 0 {
		logger.Fatal("no scan roots configured")
	}

	metadataSvc := service.NewMetadataService()
	imageRepo, jobRepo, taskPlatformSvc, closeDB := initScanStores(cfg)
	if closeDB != nil {
		defer closeDB()
	}
	provider := cfg.ThumbnailStorageProvider
	if provider == "" {
		provider = "cos"
	}

	var thumbnailDeleter interface {
		DeleteByURL(ctx context.Context, objectURL string) error
	}

	switch provider {
	case "minio":
		minioSvc, err := service.NewMinioService(
			cfg.Minio.Endpoint,
			cfg.Minio.AccessKey,
			cfg.Minio.SecretKey,
			cfg.Minio.Bucket,
			cfg.Minio.UseSSL,
		)
		if err != nil {
			logger.Fatalf("failed to initialize minio storage deleter: %v", err)
		}
		thumbnailDeleter = minioSvc
	default:
		cosSvc, err := service.NewCOSService(&cfg.COS)
		if err != nil {
			logger.Fatalf("failed to initialize cos storage deleter: %v", err)
		}
		thumbnailDeleter = cosSvc
	}

	scannerSvc := service.NewScannerService(metadataSvc, imageRepo, jobRepo, taskPlatformSvc, thumbnailDeleter, *workers)

	result, err := scannerSvc.Scan(context.TODO(), paths)
	if err != nil {
		logger.Fatalf("scan failed: %v", err)
	}

	fmt.Printf("Total files: %d\n", result.TotalFiles)
	fmt.Printf("Imported: %d\n", result.Imported)
	fmt.Printf("Skipped: %d\n", result.Skipped)
	fmt.Printf("Failed: %d\n", result.Failed)
	fmt.Printf("Duration: %s\n", result.Duration)

	if len(result.Errors) > 0 {
		for _, err := range result.Errors {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
		}
		os.Exit(1)
	}
}

func openDatabase(cfg *config.Config) (*sql.DB, error) {
	return sqliteutil.Open(cfg)
}

func initScanStores(cfg *config.Config) (repository.ImageRepository, repository.JobRepository, *service.TaskPlatformService, func()) {
	if strings.EqualFold(cfg.Database.Type, "d1") {
		client := d1client.NewClientWithAPIKeyAndReadOnly(cfg.Database.D1APIURL, cfg.Database.D1APIKey, cfg.Database.D1ReadOnly)
		tagRepo := repository.NewD1TagRepository(client)
		imageRepo := repository.NewD1ImageRepositoryWithTags(client, tagRepo)
		jobRepo := repository.NewD1JobRepository(client)
		taskSvc := service.NewTaskPlatformService(
			repository.NewD1TaskBatchRepository(client),
			repository.NewD1PlatformTaskRepository(client),
			jobRepo,
		)
		return imageRepo, jobRepo, taskSvc, nil
	}

	db, err := openDatabase(cfg)
	if err != nil {
		logger.Fatalf("failed to open database: %v", err)
	}
	if err := repository.EnsureScanSchema(db); err != nil {
		_ = db.Close()
		logger.Fatalf("failed to ensure scan schema: %v", err)
	}

	imageRepo := repository.NewImageRepository(db)
	jobRepo := repository.NewJobRepository(db)
	taskSvc := service.NewTaskPlatformService(repository.NewTaskBatchRepository(db), repository.NewPlatformTaskRepository(db), jobRepo)
	return imageRepo, jobRepo, taskSvc, func() { _ = db.Close() }
}
