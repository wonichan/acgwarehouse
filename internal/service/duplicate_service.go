package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
	"github.com/wonichan/acgwarehouse-backend/internal/sidecar"
)

const duplicatePollInterval = 2 * time.Second

const (
	defaultDuplicatePageSize  = 2000
	defaultDuplicateBatchSize = 1000
)

// DuplicateService 重复检测服务
type DuplicateService struct {
	imageRepo      repository.ImageRepository
	duplicateRepo  repository.DuplicateRepository
	sidecarClient  *sidecar.SidecarClient
	sidecarRuntime *sidecar.Runtime
	taskService    *DuplicateTaskService
	statPath       func(path string) (os.FileInfo, error)
}

// DetectOptions 检测选项
type DetectOptions struct {
	Threshold int // 汉明距离阈值，默认 40（256-bit pHash）
}

// NewDuplicateService 创建重复检测服务实例
func NewDuplicateService(
	imageRepo repository.ImageRepository,
	duplicateRepo repository.DuplicateRepository,
	sidecarClient *sidecar.SidecarClient,
	sidecarRuntime *sidecar.Runtime,
) *DuplicateService {
	return &DuplicateService{
		imageRepo:      imageRepo,
		duplicateRepo:  duplicateRepo,
		sidecarClient:  sidecarClient,
		sidecarRuntime: sidecarRuntime,
		taskService:    NewDuplicateTaskService(),
		statPath:       os.Stat,
	}
}

func duplicatePageSize() int {
	if raw := os.Getenv("DUPLICATE_PAGE_SIZE"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			return v
		}
	}
	return defaultDuplicatePageSize
}

func duplicateBatchSize() int {
	if raw := os.Getenv("DUPLICATE_BATCH_SIZE"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			return v
		}
	}
	return defaultDuplicateBatchSize
}

func (s *DuplicateService) buildDetectionInput(img domain.Image) sidecar.DetectionImageInput {
	input := sidecar.DetectionImageInput{
		ID:       img.ID,
		Path:     img.Path,
		Width:    img.Width,
		Height:   img.Height,
		FileSize: img.FileSize,
		Format:   img.Format,
	}

	if img.SHA256 != "" && img.PHashHex != "" && s.statPath != nil {
		if stat, err := s.statPath(img.Path); err == nil {
			if img.SourceMTimeUnix == stat.ModTime().UnixNano() {
				input.SHA256 = img.SHA256
				input.PHashHex = img.PHashHex
			}
		}
	}

	return input
}

func (s *DuplicateService) StartDetectDuplicatesTask(opts DetectOptions) (DuplicateTaskSnapshot, error) {
	if s.taskService == nil {
		s.taskService = NewDuplicateTaskService()
	}
	task := s.taskService.CreateTask()

	go func(taskID string, options DetectOptions) {
		_, err := s.DetectDuplicates(context.Background(), options, taskID)
		if err != nil {
			log.Printf("duplicate detection async task failed: task_id=%s error=%v", taskID, err)
		}
	}(task.TaskID, opts)

	return task, nil
}

func (s *DuplicateService) GetDuplicateTask(taskID string) (DuplicateTaskSnapshot, bool) {
	if s.taskService == nil {
		return DuplicateTaskSnapshot{}, false
	}
	return s.taskService.GetTask(taskID)
}

func (s *DuplicateService) SubscribeDuplicateTask(taskID string) (<-chan DuplicateTaskSnapshot, func(), bool) {
	if s.taskService == nil {
		return nil, func() {}, false
	}
	return s.taskService.Subscribe(taskID)
}

// DetectDuplicates 执行重复检测
// 返回检测到的重复组数量
func (s *DuplicateService) DetectDuplicates(ctx context.Context, opts DetectOptions, taskIDs ...string) (int, error) {
	threshold := opts.Threshold
	if threshold <= 0 {
		threshold = 40
	}

	var currentTaskID string
	if len(taskIDs) > 0 {
		currentTaskID = taskIDs[0]
	}

	updateTask := func(status string, progress float64, processed, total int, message string, errText string, groupsFound int) {
		if s.taskService == nil || currentTaskID == "" {
			return
		}
		_, _ = s.taskService.UpdateTask(currentTaskID, func(snapshot *DuplicateTaskSnapshot) {
			snapshot.Status = status
			snapshot.Progress = progress
			snapshot.Processed = processed
			snapshot.Total = total
			snapshot.Message = message
			snapshot.Error = errText
			snapshot.GroupsFound = groupsFound
		})
	}

	if s.sidecarClient == nil {
		updateTask(DuplicateTaskStatusFailed, 100, 0, 0, "sidecar client missing", "sidecar client is not configured", 0)
		return 0, fmt.Errorf("sidecar client is not configured")
	}

	updateTask(DuplicateTaskStatusPreparing, 5, 0, 0, "loading image metadata", "", 0)

	pageSize := duplicatePageSize()
	batchSize := duplicateBatchSize()
	lastID := int64(0)
	inputs := make([]sidecar.DetectionImageInput, 0, pageSize)
	imagesCount := 0

	for {
		page, pageErr := s.imageRepo.FindByIDRange(pageSize, lastID)
		if pageErr != nil {
			updateTask(DuplicateTaskStatusFailed, 100, 0, 0, "load images failed", pageErr.Error(), 0)
			log.Printf("duplicate detection image page load failed: threshold=%d last_id=%d error=%v", threshold, lastID, pageErr)
			return 0, pageErr
		}
		if len(page) == 0 {
			break
		}
		for _, img := range page {
			inputs = append(inputs, s.buildDetectionInput(img))
		}
		imagesCount += len(page)
		lastID = page[len(page)-1].ID
	}

	if imagesCount == 0 {
		updateTask(DuplicateTaskStatusCompleted, 100, 0, 0, "no images to process", "", 0)
		log.Printf("duplicate detection skipped: threshold=%d image_count=0", threshold)
		return 0, nil
	}
	updateTask(DuplicateTaskStatusPreparing, 10, imagesCount, imagesCount, "image metadata prepared", "", 0)

	log.Printf("duplicate detection started: threshold=%d image_count=%d", threshold, imagesCount)
	log.Printf("duplicate detection inputs prepared: threshold=%d image_count=%d page_size=%d batch_size=%d", threshold, len(inputs), pageSize, batchSize)

	sidecarTaskID, err := s.sidecarClient.SubmitDetection(ctx, sidecar.DetectionRequest{
		Threshold: threshold,
		Images:    inputs,
	})
	if err != nil {
		updateTask(DuplicateTaskStatusFailed, 100, 0, imagesCount, "submit sidecar task failed", err.Error(), 0)
		log.Printf("duplicate detection submit failed: threshold=%d image_count=%d error=%v", threshold, len(inputs), err)
		return 0, fmt.Errorf("submit sidecar detection: %w", err)
	}

	updateTask(DuplicateTaskStatusHashing, 15, 0, imagesCount, "sidecar task submitted", "", 0)
	log.Printf("duplicate detection task submitted: task_id=%s threshold=%d image_count=%d", sidecarTaskID, threshold, len(inputs))

	lastStatus := ""
	lastMessage := ""
	lastProgressBucket := -1

	for {
		status, pollErr := s.sidecarClient.PollProgress(ctx, sidecarTaskID)
		if pollErr != nil {
			updateTask(DuplicateTaskStatusFailed, 100, 0, imagesCount, "poll sidecar task failed", pollErr.Error(), 0)
			log.Printf("duplicate detection poll failed: task_id=%s error=%v", sidecarTaskID, pollErr)
			return 0, fmt.Errorf("poll sidecar detection task %s: %w", sidecarTaskID, pollErr)
		}

		mappedStatus := DuplicateTaskStatusHashing
		mappedMessage := status.Message
		if mappedMessage == "" {
			mappedMessage = status.Status
		}
		if status.Status == "completed" {
			mappedStatus = DuplicateTaskStatusGrouping
			mappedMessage = "grouping completed hashes"
		}
		if status.Status == "failed" {
			mappedStatus = DuplicateTaskStatusFailed
		}

		processed := int(float64(imagesCount) * status.Progress / 100.0)
		if processed > imagesCount {
			processed = imagesCount
		}
		globalProgress := 10 + (float64(processed)/float64(imagesCount))*70
		updateTask(mappedStatus, globalProgress, processed, imagesCount, mappedMessage, "", 0)

		progressBucket := int(status.Progress / 10)
		if status.Status != lastStatus || status.Message != lastMessage || progressBucket != lastProgressBucket {
			log.Printf(
				"duplicate detection status: task_id=%s status=%s progress=%.1f message=%s",
				sidecarTaskID,
				status.Status,
				status.Progress,
				status.Message,
			)
			lastStatus = status.Status
			lastMessage = status.Message
			lastProgressBucket = progressBucket
		}

		switch status.Status {
		case "completed":
			goto fetchResults
		case "failed":
			updateTask(DuplicateTaskStatusFailed, 100, processed, imagesCount, "sidecar task failed", status.Message, 0)
			log.Printf("duplicate detection failed: task_id=%s message=%s", sidecarTaskID, status.Message)
			if status.Message == "" {
				return 0, fmt.Errorf("sidecar detection task %s failed", sidecarTaskID)
			}
			return 0, fmt.Errorf("sidecar detection task %s failed: %s", sidecarTaskID, status.Message)
		}

		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(duplicatePollInterval):
		}
	}

fetchResults:
	updateTask(DuplicateTaskStatusGrouping, 82, imagesCount, imagesCount, "fetching grouping results", "", 0)
	log.Printf("duplicate detection fetching results: task_id=%s", sidecarTaskID)

	result, err := s.sidecarClient.FetchResults(ctx, sidecarTaskID)
	if err != nil {
		updateTask(DuplicateTaskStatusFailed, 100, imagesCount, imagesCount, "fetch results failed", err.Error(), 0)
		log.Printf("duplicate detection result fetch failed: task_id=%s error=%v", sidecarTaskID, err)
		return 0, fmt.Errorf("fetch sidecar detection result %s: %w", sidecarTaskID, err)
	}

	log.Printf(
		"duplicate detection result fetched: task_id=%s total_images=%d total_groups=%d skipped_images=%d computation_time_ms=%d",
		sidecarTaskID,
		result.TotalImages,
		result.TotalGroups,
		len(result.SkippedImages),
		result.ComputationTimeMs,
	)

	updateTask(DuplicateTaskStatusPersisting, 90, imagesCount, imagesCount, "persisting duplicate groups", "", result.TotalGroups)
	if err := s.persistDetectionResults(threshold, result); err != nil {
		updateTask(DuplicateTaskStatusFailed, 100, imagesCount, imagesCount, "persist results failed", err.Error(), result.TotalGroups)
		log.Printf("duplicate detection persist failed: error=%v", err)
		return 0, err
	}
	log.Printf("duplicate detection persisted: total_groups=%d", result.TotalGroups)
	updateTask(DuplicateTaskStatusCompleted, 100, imagesCount, imagesCount, "completed", "", result.TotalGroups)

	return result.TotalGroups, nil
}

func (s *DuplicateService) persistDetectionResults(threshold int, result *sidecar.DetectionResult) error {
	memberCount := 0
	for _, group := range result.Groups {
		memberCount += len(group.Members)
	}
	log.Printf("duplicate detection persist started: threshold=%d total_groups=%d total_members=%d skipped_images=%d", threshold, len(result.Groups), memberCount, len(result.SkippedImages))

	for _, group := range result.Groups {
		for _, member := range group.Members {
			if member.PHash == "" && member.SHA256 == "" {
				continue
			}
			sourceMTimeUnix := int64(0)
			if s.statPath != nil {
				if image, findErr := s.imageRepo.FindByID(member.ImageID); findErr == nil {
					if stat, statErr := s.statPath(image.Path); statErr == nil {
						sourceMTimeUnix = stat.ModTime().UnixNano()
					}
				}
			}
			if err := s.imageRepo.UpdateImageDuplicateHashCache(member.ImageID, member.SHA256, member.PHash, sourceMTimeUnix); err != nil {
				return fmt.Errorf("persist image hash cache for image %d: %w", member.ImageID, err)
			}
		}
	}

	if err := s.duplicateRepo.DeleteAllDuplicateGroups(); err != nil {
		return err
	}

	for _, group := range result.Groups {
		dbGroup := &domain.DuplicateGroup{
			RecommendedImageID:  group.RecommendedID,
			SimilarityThreshold: threshold,
			CreatedAt:           time.Now(),
		}

		relations := make([]domain.DuplicateRelation, len(group.Members))
		for i, member := range group.Members {
			rationale, err := json.Marshal(member.RecommendationReasons)
			if err != nil {
				return fmt.Errorf("marshal recommendation rationale for image %d: %w", member.ImageID, err)
			}

			relations[i] = domain.DuplicateRelation{
				ImageID:                 member.ImageID,
				IsRecommended:           member.IsRecommended,
				FileHash:                member.SHA256,
				PHashDistance:           member.Distance,
				RecommendationScore:     member.RecommendationScore,
				RecommendationRationale: json.RawMessage(rationale),
			}
		}

		if err := s.duplicateRepo.SaveDuplicateGroup(dbGroup, relations); err != nil {
			return err
		}
	}
	log.Printf("duplicate detection persist completed: threshold=%d total_groups=%d total_members=%d", threshold, len(result.Groups), memberCount)

	return nil
}

// GetDuplicateGroups 获取重复组列表
func (s *DuplicateService) GetDuplicateGroups(limit, offset int) ([]domain.DuplicateGroupWithImages, int64, error) {
	groups, err := s.duplicateRepo.FindDuplicateGroups(limit, offset)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.duplicateRepo.CountDuplicateGroups()
	if err != nil {
		return nil, 0, err
	}

	result := make([]domain.DuplicateGroupWithImages, len(groups))
	for i, group := range groups {
		_, relations, err := s.duplicateRepo.FindDuplicateGroupByID(group.ID)
		if err != nil {
			return nil, 0, fmt.Errorf("load duplicate group %d relations: %w", group.ID, err)
		}

		images := make([]domain.DuplicateImage, len(relations))
		for j, rel := range relations {
			img, findErr := s.imageRepo.FindByID(rel.ImageID)
			if findErr != nil {
				return nil, 0, fmt.Errorf("load image %d for duplicate group %d: %w", rel.ImageID, group.ID, findErr)
			}
			images[j] = domain.DuplicateImage{
				ID:                      img.ID,
				Path:                    img.Path,
				Filename:                img.Filename,
				SourceRoot:              img.SourceRoot,
				Width:                   img.Width,
				Height:                  img.Height,
				FileSize:                img.FileSize,
				Format:                  img.Format,
				PHash:                   img.PHash,
				PHashHex:                img.PHashHex,
				ThumbnailSmallUrl:       img.ThumbnailSmallUrl,
				ThumbnailLargeUrl:       img.ThumbnailLargeUrl,
				CreatedAt:               img.CreatedAt,
				UpdatedAt:               img.UpdatedAt,
				IsRecommended:           rel.IsRecommended,
				FileHash:                rel.FileHash,
				PHashDistance:           rel.PHashDistance,
				RecommendationScore:     rel.RecommendationScore,
				RecommendationRationale: rel.RecommendationRationale,
			}
		}

		sort.Slice(images, func(i, j int) bool {
			if images[i].IsRecommended != images[j].IsRecommended {
				return images[i].IsRecommended
			}
			return images[i].PHashDistance < images[j].PHashDistance
		})

		result[i] = domain.DuplicateGroupWithImages{Group: group, Images: images}
	}

	return result, total, nil
}

// GetDuplicateGroup 获取单个重复组详情
func (s *DuplicateService) GetDuplicateGroup(id int64) (*domain.DuplicateGroupWithImages, error) {
	group, relations, err := s.duplicateRepo.FindDuplicateGroupByID(id)
	if err != nil {
		return nil, err
	}

	images := make([]domain.DuplicateImage, len(relations))
	for i, rel := range relations {
		img, findErr := s.imageRepo.FindByID(rel.ImageID)
		if findErr != nil {
			continue
		}
		images[i] = domain.DuplicateImage{
			ID:                      img.ID,
			Path:                    img.Path,
			Filename:                img.Filename,
			SourceRoot:              img.SourceRoot,
			Width:                   img.Width,
			Height:                  img.Height,
			FileSize:                img.FileSize,
			Format:                  img.Format,
			PHash:                   img.PHash,
			PHashHex:                img.PHashHex,
			ThumbnailSmallUrl:       img.ThumbnailSmallUrl,
			ThumbnailLargeUrl:       img.ThumbnailLargeUrl,
			CreatedAt:               img.CreatedAt,
			UpdatedAt:               img.UpdatedAt,
			IsRecommended:           rel.IsRecommended,
			FileHash:                rel.FileHash,
			PHashDistance:           rel.PHashDistance,
			RecommendationScore:     rel.RecommendationScore,
			RecommendationRationale: rel.RecommendationRationale,
		}
	}

	sort.Slice(images, func(i, j int) bool {
		if images[i].IsRecommended != images[j].IsRecommended {
			return images[i].IsRecommended
		}
		return images[i].PHashDistance < images[j].PHashDistance
	})

	return &domain.DuplicateGroupWithImages{Group: *group, Images: images}, nil
}

// DeleteDuplicateGroup 删除重复组记录
func (s *DuplicateService) DeleteDuplicateGroup(id int64) error {
	return s.duplicateRepo.DeleteDuplicateGroup(id)
}
