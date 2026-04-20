package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/config"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/logger"
)

type aiTagImageFinder interface {
	FindImagesWithoutAITags(ctx context.Context, limit int) ([]domain.Image, error)
}

type aiTagTaskPlatform interface {
	PlanBatch(ctx context.Context, req TaskPlatformPlanRequest) (*TaskPlatformPlanResult, error)
	QueueTask(ctx context.Context, task *domain.PlatformTask, jobType, payload string) (*domain.AsyncJob, error)
}

type schedulerTicker interface {
	C() <-chan time.Time
	Stop()
}

type timeTicker struct {
	ticker *time.Ticker
}

func (t *timeTicker) C() <-chan time.Time {
	return t.ticker.C
}

func (t *timeTicker) Stop() {
	t.ticker.Stop()
}

type AITagAutoScheduler struct {
	imageFinder    aiTagImageFinder
	taskPlatform   aiTagTaskPlatform
	config         *config.Config
	ticker         schedulerTicker
	tickerFactory  func(time.Duration) schedulerTicker
	scanAndEnqueue func(ctx context.Context) (int, error)
	stopCh         chan struct{}
	startOnce      sync.Once
	stopOnce       sync.Once
}

func NewAITagAutoScheduler(imageFinder aiTagImageFinder, taskPlatform aiTagTaskPlatform, cfg *config.Config) *AITagAutoScheduler {
	scheduler := &AITagAutoScheduler{
		imageFinder:   imageFinder,
		taskPlatform:  taskPlatform,
		config:        cfg,
		tickerFactory: newSchedulerTicker,
		stopCh:        make(chan struct{}),
	}
	scheduler.scanAndEnqueue = scheduler.ScanAndEnqueue
	return scheduler
}

func (s *AITagAutoScheduler) Start(ctx context.Context) {
	if s == nil {
		return
	}

	s.startOnce.Do(func() {
		interval := 5 * time.Minute
		if s.config != nil && s.config.AI.AutoScanIntervalMinutes > 0 {
			interval = time.Duration(s.config.AI.AutoScanIntervalMinutes) * time.Minute
		}
		s.ticker = s.tickerFactory(interval)

		go func() {
			for {
				select {
				case <-s.stopCh:
					return
				case <-ctx.Done():
					return
				case <-s.ticker.C():
					if _, err := s.scanAndEnqueue(context.Background()); err != nil {
						logger.Errorf("AI 标签自动调度扫描失败: %v", err)
					}
				}
			}
		}()
	})
}

func (s *AITagAutoScheduler) Stop() {
	if s == nil {
		return
	}

	s.stopOnce.Do(func() {
		if s.ticker != nil {
			s.ticker.Stop()
		}
		close(s.stopCh)
	})
}

func (s *AITagAutoScheduler) ScanAndEnqueue(ctx context.Context) (int, error) {
	if s == nil || s.config == nil || !s.config.AI.AutoAITagOnImport {
		return 0, nil
	}
	thumbnailBaseURL := ResolveThumbnailBaseURL(s.config)
	if s.imageFinder == nil || s.taskPlatform == nil {
		return 0, nil
	}

	limit := s.config.AI.AutoScanBatchSize
	if limit <= 0 {
		limit = 100
	}

	images, err := s.imageFinder.FindImagesWithoutAITags(ctx, limit)
	if err != nil {
		return 0, fmt.Errorf("find images without AI tags: %w", err)
	}
	if len(images) == 0 {
		return 0, nil
	}

	items := make([]TaskPlatformPlanItem, 0, len(images))
	sourceRoots := make([]string, 0, len(images))
	imagesByID := make(map[int64]domain.Image, len(images))
	for _, image := range images {
		imageCopy := image
		imagesByID[image.ID] = imageCopy
		sourceRoots = append(sourceRoots, image.SourceRoot)
		items = append(items, TaskPlatformPlanItem{
			ImageID:          image.ID,
			ImageVersionKey:  BuildImageVersionKey(&imageCopy),
			SourceDescriptor: image.Path,
		})
	}

	plan, err := s.taskPlatform.PlanBatch(ctx, TaskPlatformPlanRequest{
		SourceType:   domain.TaskBatchSourceImportScan,
		SummaryLabel: BuildTaskBatchSummaryLabel(domain.TaskBatchSourceImportScan, sourceRoots, len(images)),
		SourceRoots:  sourceRoots,
		TaskTypes:    []string{domain.PlatformTaskTypeAITagGeneration},
		Items:        items,
	})
	if err != nil {
		return 0, fmt.Errorf("plan AI tag batch: %w", err)
	}

	queued := 0
	for i := range plan.CreatedTasks {
		image, ok := imagesByID[plan.CreatedTasks[i].ImageID]
		if !ok {
			continue
		}
		payload, err := json.Marshal(map[string]any{
			"image_id": image.ID,
			"path":     ResolveAITagImagePath(&image, thumbnailBaseURL),
		})
		if err != nil {
			return queued, fmt.Errorf("marshal AI tag payload: %w", err)
		}
		if _, err := s.taskPlatform.QueueTask(ctx, &plan.CreatedTasks[i], domain.PlatformTaskTypeAITagGeneration, string(payload)); err != nil {
			return queued, fmt.Errorf("queue AI tag task for image %d: %w", image.ID, err)
		}
		queued++
	}

	return queued, nil
}

func newSchedulerTicker(interval time.Duration) schedulerTicker {
	return &timeTicker{ticker: time.NewTicker(interval)}
}
