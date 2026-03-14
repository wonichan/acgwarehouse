package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"

	"github.com/yourusername/acgwarehouse-backend/internal/config"
	"github.com/yourusername/acgwarehouse-backend/internal/repository"
	"github.com/yourusername/acgwarehouse-backend/internal/service"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to config file")
	scanPath := flag.String("path", "", "Path to scan")
	workers := flag.Int("workers", 1, "Number of worker goroutines")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	paths := cfg.Storage.ScanRoots
	if *scanPath != "" {
		paths = []string{*scanPath}
	}
	if len(paths) == 0 {
		log.Fatal("no scan roots configured")
	}

	db, err := openDatabase(cfg)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := repository.EnsureScanSchema(db); err != nil {
		log.Fatalf("failed to ensure scan schema: %v", err)
	}

	metadataSvc := service.NewMetadataService()
	imageRepo := repository.NewImageRepository(db)
	jobRepo := repository.NewJobRepository(db)
	scannerSvc := service.NewScannerService(metadataSvc, imageRepo, jobRepo, *workers)

	result, err := scannerSvc.Scan(nil, paths)
	if err != nil {
		log.Fatalf("scan failed: %v", err)
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
	if cfg.Database.Type == "postgres" {
		return nil, fmt.Errorf("postgres scanning bootstrap is not implemented yet")
	}
	return sql.Open("sqlite3", cfg.Database.Path)
}
