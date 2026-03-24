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
}

type ScannerService struct {
	metadataSvc *MetadataService
	imageRepo   repository.ImageRepository
	jobRepo     repository.JobRepository
	taskSvc     *TaskPlatformService
	workers     int
}

func NewScannerService(metadataSvc *MetadataService, imageRepo repository.ImageRepository, jobRepo repository.JobRepository, taskSvc *TaskPlatformService, workers int) *ScannerService {
	if workers < 1 {
		workers = 1
	}
	return &ScannerService{
		metadataSvc: metadataSvc,
		imageRepo:   imageRepo,
		jobRepo:     jobRepo,
		taskSvc:     taskSvc,
		workers:     workers,
	}
}

func (s *ScannerService) Scan(ctx context.Context, roots []string) (*ScanResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}

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
			return nil, walkErr
		}
		if errors.Is(walkErr, context.Canceled) {
			close(fileCh)
			wg.Wait()
			return nil, ctx.Err()
		}
	}

	close(fileCh)
	wg.Wait()
	result.SourceRoots = uniqueNonEmptyStrings(roots)
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
	}

	result.Duration = time.Since(start)
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

func sameOrChildPath(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != "" && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}
