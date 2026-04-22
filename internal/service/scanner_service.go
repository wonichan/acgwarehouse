package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/logger"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
)

type ScanResult struct {
	TotalFiles                 int
	Imported                   int
	Skipped                    int
	Failed                     int
	Duration                   time.Duration
	Errors                     []error
	BatchID                    int64
	BatchStatus                string
	BatchSourceType            string
	SummaryLabel               string
	PlatformStatus             string
	BatchNewImages             int64
	BatchSkippedImages         int64
	BatchSkippedUnchanged      int64
	BatchSkippedDuplicateTasks int64
	TotalImagesInBatch         int64
	CreatedTasks               int
	SkippedTasks               int
	SourceRoots                []string
	PlannedTaskTypes           []string
	ImportedImageIDs           []int64
	ImportedImagePaths         []string
	CreatedPlatformTaskIDs     []int64
	DeletedStale               int
}

type ScannerService struct {
	metadataSvc *MetadataService
	imageRepo   repository.ImageRepository
	jobRepo     repository.JobRepository
	taskSvc     *TaskPlatformService
	deleter     thumbnailRemoteDeleter
	workers     int
}

type thumbnailRemoteDeleter interface {
	DeleteByURL(ctx context.Context, objectURL string) error
}

func NewScannerService(metadataSvc *MetadataService, imageRepo repository.ImageRepository, jobRepo repository.JobRepository, taskSvc *TaskPlatformService, deleter thumbnailRemoteDeleter, workers int) *ScannerService {
	if workers < 1 {
		workers = 1
	}
	return &ScannerService{
		metadataSvc: metadataSvc,
		imageRepo:   imageRepo,
		jobRepo:     jobRepo,
		taskSvc:     taskSvc,
		deleter:     deleter,
		workers:     workers,
	}
}

func (s *ScannerService) Scan(ctx context.Context, roots []string) (*ScanResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	logger.Infof("[service] Scan started: roots=%d workers=%d", len(roots), s.workers)

	start := time.Now()
	result := &ScanResult{}
	fileCh := make(chan fileTask, 64)

	var (
		wg    sync.WaitGroup
		mu    sync.Mutex
		items []TaskPlatformPlanItem
	)

	workerCount := s.workers
	if workerCount < 1 {
		workerCount = 1
	}

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range fileCh {
				imported, err := s.importFile(task.path, task.root)
				if err != nil {
					mu.Lock()
					result.Failed++
					result.Errors = append(result.Errors, fmt.Errorf("%s: %w", task.path, err))
					mu.Unlock()
					continue
				}
				mu.Lock()
				if imported.IsNew {
					result.Imported++
					result.ImportedImageIDs = append(result.ImportedImageIDs, imported.Image.ID)
					result.ImportedImagePaths = append(result.ImportedImagePaths, imported.Image.Path)
				} else {
					result.Skipped++
				}
				items = append(items, TaskPlatformPlanItem{
					ImageID:          imported.Image.ID,
					ImageVersionKey:  BuildImageVersionKey(imported.Image),
					SourceDescriptor: imported.Image.Path,
					SkipPlanning:     !imported.IsNew,
					SkipReason:       domain.PlatformTaskSkipReasonUnchanged,
				})
				mu.Unlock()
			}
		}()
	}

	for _, root := range roots {
		root := filepath.Clean(root)
		walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				mu.Lock()
				result.Errors = append(result.Errors, err)
				mu.Unlock()
				return nil
			}
			if d.IsDir() {
				return nil
			}
			if !s.metadataSvc.IsImage(path) {
				mu.Lock()
				result.Skipped++
				mu.Unlock()
				return nil
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case fileCh <- fileTask{path: path, root: root}:
				mu.Lock()
				result.TotalFiles++
				mu.Unlock()
				return nil
			}
		})
		if walkErr != nil && !errors.Is(walkErr, context.Canceled) {
			close(fileCh)
			wg.Wait()
			logger.Errorf("[service] Scan failed: root=%s error=%v", root, walkErr)
			return nil, walkErr
		}
		if errors.Is(walkErr, context.Canceled) {
			close(fileCh)
			wg.Wait()
			logger.Infof("[service] Scan cancelled: total_files=%d imported=%d", result.TotalFiles, result.Imported)
			return nil, ctx.Err()
		}
	}

	close(fileCh)
	wg.Wait()
	result.SourceRoots = uniqueNonEmptyStrings(roots)
	cleaned, err := s.cleanupStaleImages(ctx, result.SourceRoots, items)
	if err != nil {
		logger.Errorf("[service] Scan failed: stale cleanup error=%v", err)
		return nil, err
	}
	result.DeletedStale = cleaned
	result.PlannedTaskTypes = []string{domain.PlatformTaskTypeThumbnailGenerate}
	result.TotalImagesInBatch = int64(len(items))
	result.SummaryLabel = BuildTaskBatchSummaryLabel(domain.TaskBatchSourceImportScan, result.SourceRoots, len(items))

	if s.taskSvc != nil {
		planResult, err := s.taskSvc.PlanBatch(ctx, TaskPlatformPlanRequest{
			SourceType:   domain.TaskBatchSourceImportScan,
			SummaryLabel: result.SummaryLabel,
			SourceRoots:  result.SourceRoots,
			TaskTypes:    result.PlannedTaskTypes,
			Items:        items,
		})
		if err != nil {
			logger.Errorf("[service] Scan failed: task planning error=%v", err)
			return nil, err
		}
		result.BatchID = planResult.Batch.ID
		result.BatchStatus = planResult.Batch.Status
		result.PlatformStatus = planResult.Batch.Status
		result.BatchSourceType = planResult.Batch.SourceType
		result.BatchNewImages = planResult.Batch.NewImages
		result.BatchSkippedImages = planResult.Batch.SkippedImages
		result.BatchSkippedUnchanged = planResult.Batch.SkippedUnchanged
		result.BatchSkippedDuplicateTasks = planResult.Batch.SkippedDuplicateTasks
		result.CreatedTasks = len(planResult.CreatedTasks)
		result.SkippedTasks = len(items) - len(planResult.CreatedTasks)
		for _, task := range planResult.CreatedTasks {
			result.CreatedPlatformTaskIDs = append(result.CreatedPlatformTaskIDs, task.ID)
		}

		// Build imageID→path map for payload construction
		imageIDToPath := make(map[int64]string, len(items))
		for _, item := range items {
			imageIDToPath[item.ImageID] = item.SourceDescriptor
		}

		// Queue thumbnail tasks for worker processing
		for i := range planResult.CreatedTasks {
			task := &planResult.CreatedTasks[i]
			path := imageIDToPath[task.ImageID]

			// Extract filename without extension for thumbnail naming
			filename := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))

			payload, err := json.Marshal(map[string]any{
				"image_id": task.ImageID,
				"path":     path,
				"filename": filename,
			})
			if err != nil {
				logger.Errorf("[service] Scan failed: payload marshal error=%v task_id=%d", err, task.ID)
				return nil, fmt.Errorf("marshal thumbnail payload for task %d: %w", task.ID, err)
			}

			if _, err := s.taskSvc.QueueTask(ctx, task, domain.PlatformTaskTypeThumbnailGenerate, string(payload)); err != nil {
				logger.Errorf("[service] Scan failed: queue task error=%v task_id=%d", err, task.ID)
				return nil, fmt.Errorf("queue thumbnail task %d: %w", task.ID, err)
			}
		}
	}

	result.Duration = time.Since(start)
	logger.Infof("[service] Scan completed: total_files=%d imported=%d skipped=%d failed=%d duration=%v", result.TotalFiles, result.Imported, result.Skipped, result.Failed, result.Duration)
	return result, nil
}

type fileTask struct {
	path string
	root string
}

type importedImageResult struct {
	Image *domain.Image
	IsNew bool
}

func (s *ScannerService) importFile(path, root string) (*importedImageResult, error) {
	image, err := s.metadataSvc.ExtractMetadata(path)
	if err != nil {
		return nil, err
	}
	image.SourceRoot = root

	// SaveImage 返回 isNew 表示是否为新插入的记录
	// INSERT OR IGNORE 会自动处理重复路径，只有新图片才会返回 isNew=true
	isNew, err := s.imageRepo.SaveImage(image)
	if err != nil {
		return nil, err
	}

	if s.taskSvc == nil && isNew && s.jobRepo != nil {
		// Extract filename without extension for thumbnail naming
		filename := strings.TrimSuffix(filepath.Base(image.Path), filepath.Ext(image.Path))
		payload, err := json.Marshal(map[string]any{
			"image_id": image.ID,
			"path":     image.Path,
			"filename": filename,
		})
		if err != nil {
			return nil, err
		}
		job := &domain.AsyncJob{
			Type:      "image_imported",
			Status:    "ready",
			Payload:   string(payload),
			Progress:  0,
			CreatedAt: time.Now(),
		}
		if err := s.jobRepo.Save(job); err != nil {
			return nil, err
		}
	}

	return &importedImageResult{Image: image, IsNew: isNew}, nil
}

func (s *ScannerService) cleanupStaleImages(ctx context.Context, sourceRoots []string, items []TaskPlatformPlanItem) (int, error) {
	if len(sourceRoots) == 0 {
		return 0, nil
	}

	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.SourceDescriptor) == "" {
			continue
		}
		seen[item.SourceDescriptor] = struct{}{}
	}

	const pageSize = 1000
	lastID := int64(0)
	deleted := 0

	for {
		if ctx != nil {
			select {
			case <-ctx.Done():
				return deleted, ctx.Err()
			default:
			}
		}

		images, err := s.imageRepo.FindBySourceRootsAfterID(pageSize, lastID, sourceRoots)
		if err != nil {
			return deleted, err
		}
		if len(images) == 0 {
			break
		}

		for _, image := range images {
			if _, ok := seen[image.Path]; ok {
				continue
			}
			if err := s.deleteStaleRemoteThumbnails(ctx, image); err != nil {
				return deleted, err
			}
			if err := s.imageRepo.Delete(image.ID); err != nil {
				return deleted, err
			}
			deleted++
		}

		lastID = images[len(images)-1].ID
	}

	if deleted > 0 {
		logger.Infof("[service] Scan stale cleanup completed: deleted=%d roots=%d", deleted, len(sourceRoots))
	}

	return deleted, nil
}

func (s *ScannerService) deleteStaleRemoteThumbnails(ctx context.Context, image domain.Image) error {
	urls := orderedUniqueThumbnailURLs(image.ThumbnailSmallUrl, image.ThumbnailLargeUrl)
	if len(urls) == 0 {
		return nil
	}
	if s.deleter == nil {
		return fmt.Errorf("thumbnail remote deleter is not configured")
	}
	for _, objectURL := range urls {
		if err := s.deleter.DeleteByURL(ctx, objectURL); err != nil {
			return fmt.Errorf("delete stale remote thumbnail for image %d: %w", image.ID, err)
		}
	}
	return nil
}

func orderedUniqueThumbnailURLs(values ...string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func sameOrChildPath(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != "" && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}
