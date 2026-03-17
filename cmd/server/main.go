package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/gin-gonic/gin"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/wonichan/acgwarehouse-backend/internal/ai"
	"github.com/wonichan/acgwarehouse-backend/internal/config"
	"github.com/wonichan/acgwarehouse-backend/internal/handler"
	"github.com/wonichan/acgwarehouse-backend/internal/middleware"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
	"github.com/wonichan/acgwarehouse-backend/internal/service"
	"github.com/wonichan/acgwarehouse-backend/internal/worker"
)

var registerAITagHandler = worker.RegisterAITagHandler

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	if strings.EqualFold(cfg.Server.Env, "production") {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(middleware.Logger())
	r.Use(middleware.CORS())
	r.Use(gin.Recovery())

	db, err := openDatabase(cfg)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := repository.EnsureScanSchema(db); err != nil {
		log.Fatalf("failed to ensure scan schema: %v", err)
	}

	imageRepo := repository.NewImageRepository(db)
	jobRepo := repository.NewJobRepository(db)
	tagRepo := repository.NewTagRepository(db)
	aliasRepo := repository.NewTagAliasRepository(db)
	obsRepo := repository.NewTagObservationRepository(db)
	imageTagRepo := repository.NewImageTagRepository(db)
	duplicateRepo := repository.NewDuplicateRepository(db)
	searchRepo := repository.NewSearchRepository(db)
	governanceSvc := service.NewTagGovernanceService(tagRepo, aliasRepo, obsRepo, imageTagRepo)
	hashSvc := service.NewHashService()
	duplicateSvc := service.NewDuplicateService(imageRepo, duplicateRepo, hashSvc)
	searchSvc := service.NewSearchService(imageRepo, tagRepo, searchRepo)
	jobManager := worker.NewManager(jobRepo)
	jobManager.Start(context.Background())
	defer jobManager.Stop()

	provider, err := ai.NewProvider(&cfg.AI)
	if err == nil {
		client := ai.NewRateLimitedClient(provider, cfg.AI.RequestsPerMinute)
		registerAIWorker(jobManager, client, obsRepo, governanceSvc)
	} else {
		log.Printf("AI provider not configured for background processing: %v", err)
	}

	handler.SetupRoutes(r, &handler.Dependencies{
		ImageRepo:     imageRepo,
		JobRepo:       jobRepo,
		TagRepo:       tagRepo,
		AliasRepo:     aliasRepo,
		ObsRepo:       obsRepo,
		ImageTagRepo:  imageTagRepo,
		DuplicateRepo: duplicateRepo,
		SearchRepo:    searchRepo,
		GovernanceSvc: governanceSvc,
		DuplicateSvc:  duplicateSvc,
		SearchSvc:     searchSvc,
		HashSvc:       hashSvc,
		JobManager:    jobManager,
	})

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("ACGWarehouse server starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}

func openDatabase(cfg *config.Config) (*sql.DB, error) {
	if strings.EqualFold(cfg.Database.Type, "postgres") {
		return nil, fmt.Errorf("postgres server bootstrap is not implemented yet")
	}
	return sql.Open("sqlite3", cfg.Database.Path)
}

func registerAIWorker(manager *worker.Manager, client ai.AIProvider, obsRepo repository.TagObservationRepository, governanceSvc worker.TagGovernanceMerger) {
	registerAITagHandler(manager, client, obsRepo, governanceSvc)
}
