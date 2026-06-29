package main

import (
	"context"
	stderrors "errors"
	"flag"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	pkgerrors "github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/yachiyo/acgwarehouse/internal/conf"
	cosclient "github.com/yachiyo/acgwarehouse/internal/infra/client/cos"
	"github.com/yachiyo/acgwarehouse/internal/infra/db"
	"github.com/yachiyo/acgwarehouse/internal/infra/search"
	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/internal/repository"
	"github.com/yachiyo/acgwarehouse/pkg/logger"
)

const imageHeaderTimeout = 15 * time.Second

// syncStats 记录本次同步任务统计。
type syncStats struct {
	Listed  int
	Skipped int
	Upsert  int
	Indexed int
}

// main 启动 COS 同步或索引重建任务。
func main() {
	ctx := context.Background()
	if err := run(ctx, os.Args[1:]); err != nil {
		logger.Error(ctx, "sync command failed", zap.Error(err))
		os.Exit(1)
	}
}

// run 加载依赖并执行同步任务。
func run(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("sync", flag.ContinueOnError)
	reindex := flags.Bool("reindex", false, "rebuild bleve index from sqlite")
	if err := flags.Parse(args); err != nil {
		return pkgerrors.WithMessage(err, "parse flags")
	}
	cfg, err := conf.Load()
	if err != nil {
		return pkgerrors.WithMessage(err, "load config")
	}
	if err := setupLogger(cfg); err != nil {
		return err
	}
	if !*reindex {
		if err := cosclient.ValidateConfig(cfg.COS); err != nil {
			return pkgerrors.WithMessage(err, "validate cos config")
		}
	}
	return withResources(ctx, cfg, func(repo *repository.ImageRepository, index *search.Index) error {
		if *reindex {
			return reindexFromSQLite(ctx, repo, index)
		}
		return syncFromCOS(ctx, cfg, repo, index)
	})
}

// setupLogger 初始化同步任务日志。
func setupLogger(cfg conf.Config) error {
	zapLogger, err := logger.New(cfg.Log.Level)
	if err != nil {
		return pkgerrors.WithMessage(err, "create logger")
	}
	logger.ReplaceGlobal(zapLogger)
	return nil
}

// withResources 初始化 SQLite 与 bleve 资源并确保释放。
func withResources(ctx context.Context, cfg conf.Config, fn func(*repository.ImageRepository, *search.Index) error) (err error) {
	sqliteDB, err := db.NewSQLite(cfg.Database)
	if err != nil {
		return pkgerrors.WithMessage(err, "init sqlite")
	}
	defer func() {
		err = appendCleanupError(err, sqliteDB.Close(), "close sqlite")
	}()
	index, err := search.Open(cfg.Search.BlevePath)
	if err != nil {
		return pkgerrors.WithMessage(err, "open search index")
	}
	defer func() {
		err = appendCleanupError(err, index.Close(), "close search index")
	}()
	return fn(repository.NewImageRepository(sqliteDB.Read, sqliteDB.Write), index)
}

// syncFromCOS 从 COS 拉取对象并写入 SQLite 与 bleve。
func syncFromCOS(ctx context.Context, cfg conf.Config, repo *repository.ImageRepository, index *search.Index) error {
	client, err := cosclient.NewClient(cfg.COS)
	if err != nil {
		return pkgerrors.WithMessage(err, "create cos client")
	}
	objects, err := client.ListObjects(ctx, cfg.COS.Prefix)
	if err != nil {
		return pkgerrors.WithMessage(err, "list cos objects")
	}
	stats := syncStats{Listed: len(objects)}
	for _, object := range objects {
		if shouldSkipObject(object.Key) {
			stats.Skipped++
			continue
		}
		stored, err := upsertObject(ctx, client, repo, object)
		if err != nil {
			return pkgerrors.WithMessage(err, "upsert cos object")
		}
		stats.Upsert++
		if err := index.Index(ctx, stored); err != nil {
			logger.Warn(ctx, "index synced image failed",
				zap.Error(err),
				zap.Int64("image_id", stored.ID),
				zap.String("cos_key", stored.COSKey),
			)
			continue
		}
		stats.Indexed++
	}
	logger.Info(ctx, "cos sync completed",
		zap.Int("listed", stats.Listed),
		zap.Int("skipped", stats.Skipped),
		zap.Int("upsert", stats.Upsert),
		zap.Int("indexed", stats.Indexed),
	)
	return nil
}

func shouldSkipObject(key string) bool {
	return strings.Contains(filepath.Base(key), "small")
}

// upsertObject 解析单个 COS 对象展示元数据并落库。
func upsertObject(ctx context.Context, client *cosclient.Client, repo *repository.ImageRepository, object cosclient.Object) (do.Image, error) {
	width, height, err := decodeRemoteSize(ctx, client.ObjectURL(object.Key))
	if err != nil {
		logger.Warn(ctx, "decode image size failed", zap.Error(err), zap.String("cos_key", object.Key))
	}
	stored, err := repo.UpsertByCOSKey(ctx, do.Image{
		COSKey:       object.Key,
		Filename:     filepath.Base(object.Key),
		Size:         object.Size,
		LastModified: object.LastModified,
		Width:        width,
		Height:       height,
		Category:     inferCategory(object.Key),
	})
	if err != nil {
		return do.Image{}, pkgerrors.WithMessage(err, "upsert image")
	}
	return stored, nil
}

// reindexFromSQLite 从 SQLite 真相源全量重建 bleve 索引。
func reindexFromSQLite(ctx context.Context, repo *repository.ImageRepository, index *search.Index) error {
	images, err := repo.ListActive(ctx, repository.ImageListQuery{})
	if err != nil {
		return pkgerrors.WithMessage(err, "list active images")
	}
	if err := index.Reindex(ctx, images); err != nil {
		return pkgerrors.WithMessage(err, "reindex images")
	}
	count, err := index.Count(ctx)
	if err != nil {
		return pkgerrors.WithMessage(err, "count index after reindex")
	}
	logger.Info(ctx, "bleve reindex completed", zap.Int("images", len(images)), zap.Uint64("documents", count))
	return nil
}

// decodeRemoteSize 读取远程图片头并解析宽高。
func decodeRemoteSize(ctx context.Context, rawURL string) (int, int, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return 0, 0, pkgerrors.WithMessage(err, "create image request")
	}
	request.Header.Set("Range", "bytes=0-65535")
	client := http.Client{Timeout: imageHeaderTimeout}
	response, err := client.Do(request)
	if err != nil {
		return 0, 0, pkgerrors.WithMessage(err, "get image header")
	}
	defer response.Body.Close()
	if response.StatusCode >= http.StatusBadRequest {
		return 0, 0, pkgerrors.Errorf("get image header status %d", response.StatusCode)
	}
	cfg, _, err := image.DecodeConfig(response.Body)
	if err != nil {
		return 0, 0, pkgerrors.WithMessage(err, "decode image config")
	}
	return cfg.Width, cfg.Height, nil
}

// inferCategory 从 COS key 推断图片分类，未知规则返回空字符串。
func inferCategory(key string) string {
	parts := strings.Split(strings.Trim(key, "/"), "/")
	if len(parts) < 3 || parts[0] != "thumbnails" {
		return ""
	}
	return parts[1]
}

// appendCleanupError 合并主流程错误与清理错误。
func appendCleanupError(err error, cleanupErr error, message string) error {
	if cleanupErr == nil {
		return err
	}
	wrapped := pkgerrors.WithMessage(cleanupErr, message)
	if err != nil {
		return stderrors.Join(err, wrapped)
	}
	return wrapped
}
