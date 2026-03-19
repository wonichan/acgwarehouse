package main

import (
	"context"
	"database/sql"
	"encoding/json"
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
	collectionRepo := repository.NewCollectionRepository(db)
	governanceSvc := service.NewTagGovernanceService(tagRepo, aliasRepo, obsRepo, imageTagRepo)
	hashSvc := service.NewHashService()
	duplicateSvc := service.NewDuplicateService(imageRepo, duplicateRepo, hashSvc)
	searchSvc := service.NewSearchService(imageRepo, tagRepo, searchRepo)
	jobManager := worker.NewManager(jobRepo)
	jobManager.Start(context.Background())
	defer jobManager.Stop()

	// Thumbnail generation handler
	thumbnailSvc := service.NewThumbnailService()
	cosSvc, err := service.NewCOSService(&cfg.COS)
	if err != nil {
		log.Printf("thumbnail job handler not registered: %v", err)
	} else {
		thumbnailHandler := worker.NewThumbnailHandler(thumbnailSvc, cosSvc, imageRepo)
		jobManager.RegisterHandler("thumbnail_generate", thumbnailHandler.Handle)

		// 注册 image_imported 处理器 - 自动触发缩略图生成任务
		jobManager.RegisterHandler("image_imported", func(ctx context.Context, id int64, payload string) error {
			// 解析 payload 获取 image_id 和 path
			var p struct {
				ImageID int64  `json:"image_id"`
				Path    string `json:"path"`
			}
			if err := json.Unmarshal([]byte(payload), &p); err != nil {
				return fmt.Errorf("解析 image_imported payload 失败: %w", err)
			}

			// 创建缩略图生成任务
			thumbnailPayload, err := json.Marshal(map[string]interface{}{
				"image_id": p.ImageID,
				"path":     p.Path,
			})
			if err != nil {
				return err
			}

			// 添加到任务队列
			_, err = jobManager.AddJob(ctx, "thumbnail_generate", string(thumbnailPayload))
			if err != nil {
				return fmt.Errorf("添加缩略图生成任务失败: %w", err)
			}

			log.Printf("已为新导入的图片 %d 创建缩略图生成任务", p.ImageID)
			return nil
		})
		log.Printf("已注册 image_imported 处理器 - 将自动触发缩略图生成")
	}

	// Admin service - wrap jobManager to implement the control interface
	adminSvc := service.NewAdminService(
		cfg,
		jobRepo,
		imageRepo,
		tagRepo,
		collectionRepo,
		jobManager,
	)

	provider, err := ai.NewProvider(&cfg.AI)
	if err == nil {
		client := ai.NewRateLimitedClient(provider, cfg.AI.RequestsPerMinute)
		registerAIWorker(jobManager, client, obsRepo, governanceSvc)
	} else {
		log.Printf("AI provider not configured for background processing: %v", err)
	}

	handler.SetupRoutes(r, &handler.Dependencies{
		ImageRepo:      imageRepo,
		JobRepo:        jobRepo,
		TagRepo:        tagRepo,
		AliasRepo:      aliasRepo,
		ObsRepo:        obsRepo,
		ImageTagRepo:   imageTagRepo,
		DuplicateRepo:  duplicateRepo,
		SearchRepo:     searchRepo,
		CollectionRepo: collectionRepo,
		GovernanceSvc:  governanceSvc,
		DuplicateSvc:   duplicateSvc,
		SearchSvc:      searchSvc,
		HashSvc:        hashSvc,
		JobManager:     jobManager,
		AdminSvc:       adminSvc,
		AdminCfg:       cfg,
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
