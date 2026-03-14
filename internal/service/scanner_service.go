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

	"github.com/yourusername/acgwarehouse-backend/internal/domain"
	"github.com/yourusername/acgwarehouse-backend/internal/repository"
)

type ScanResult struct {
	TotalFiles int
	Imported   int
	Skipped    int
	Failed     int
	Duration   time.Duration
	Errors     []error
}

type ScannerService struct {
	metadataSvc *MetadataService
	imageRepo   repository.ImageRepository
	jobRepo     repository.JobRepository
	workers     int
}

func NewScannerService(metadataSvc *MetadataService, imageRepo repository.ImageRepository, jobRepo repository.JobRepository, workers int) *ScannerService {
	if workers < 1 {
		workers = 1
	}
	return &ScannerService{
		metadataSvc: metadataSvc,
		imageRepo:   imageRepo,
		jobRepo:     jobRepo,
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
		wg sync.WaitGroup
		mu sync.Mutex
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
				if err := s.importFile(task.path, task.root); err != nil {
					mu.Lock()
					result.Failed++
					result.Errors = append(result.Errors, fmt.Errorf("%s: %w", task.path, err))
					mu.Unlock()
					continue
				}
				mu.Lock()
				result.Imported++
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
	result.Duration = time.Since(start)
	return result, nil
}

type fileTask struct {
	path string
	root string
}

func (s *ScannerService) importFile(path, root string) error {
	image, err := s.metadataSvc.ExtractMetadata(path)
	if err != nil {
		return err
	}
	image.SourceRoot = root

	if err := s.imageRepo.SaveImage(image); err != nil {
		return err
	}

	if s.jobRepo != nil {
		payload, err := json.Marshal(map[string]any{
			"image_id": image.ID,
			"path":     image.Path,
		})
		if err != nil {
			return err
		}
		job := &domain.AsyncJob{
			Type:      "image_imported",
			Status:    "ready",
			Payload:   string(payload),
			Progress:  0,
			CreatedAt: time.Now(),
		}
		if err := s.jobRepo.Save(job); err != nil {
			return err
		}
	}

	return nil
}

func sameOrChildPath(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != "" && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}
