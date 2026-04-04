package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
	"github.com/wonichan/acgwarehouse-backend/internal/sidecar"
)

const duplicatePollInterval = 2 * time.Second

// DuplicateService 重复检测服务
type DuplicateService struct {
	imageRepo      repository.ImageRepository
	duplicateRepo  repository.DuplicateRepository
	sidecarClient  *sidecar.SidecarClient
	sidecarRuntime *sidecar.Runtime
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
	}
}

// DetectDuplicates 执行重复检测
// 返回检测到的重复组数量
func (s *DuplicateService) DetectDuplicates(ctx context.Context, opts DetectOptions) (int, error) {
	threshold := opts.Threshold
	if threshold <= 0 {
		threshold = 40
	}

	if s.sidecarClient == nil {
		return 0, fmt.Errorf("sidecar client is not configured")
	}

	images, err := s.imageRepo.FindAll(1000000, 0, "id", "asc")
	if err != nil {
		return 0, err
	}
	if len(images) == 0 {
		return 0, nil
	}

	inputs := make([]sidecar.DetectionImageInput, len(images))
	for i, img := range images {
		inputs[i] = sidecar.DetectionImageInput{
			ID:       img.ID,
			Path:     img.Path,
			Width:    img.Width,
			Height:   img.Height,
			FileSize: img.FileSize,
			Format:   img.Format,
		}
	}

	taskID, err := s.sidecarClient.SubmitDetection(ctx, sidecar.DetectionRequest{
		Threshold: threshold,
		Images:    inputs,
	})
	if err != nil {
		return 0, fmt.Errorf("submit sidecar detection: %w", err)
	}

	for {
		status, pollErr := s.sidecarClient.PollProgress(ctx, taskID)
		if pollErr != nil {
			return 0, fmt.Errorf("poll sidecar detection task %s: %w", taskID, pollErr)
		}

		switch status.Status {
		case "completed":
			goto fetchResults
		case "failed":
			if status.Message == "" {
				return 0, fmt.Errorf("sidecar detection task %s failed", taskID)
			}
			return 0, fmt.Errorf("sidecar detection task %s failed: %s", taskID, status.Message)
		}

		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(duplicatePollInterval):
		}
	}

fetchResults:
	result, err := s.sidecarClient.FetchResults(ctx, taskID)
	if err != nil {
		return 0, fmt.Errorf("fetch sidecar detection result %s: %w", taskID, err)
	}

	if err := s.persistDetectionResults(threshold, result); err != nil {
		return 0, err
	}

	return result.TotalGroups, nil
}

func (s *DuplicateService) persistDetectionResults(threshold int, result *sidecar.DetectionResult) error {
	for _, group := range result.Groups {
		for _, member := range group.Members {
			if member.PHash == "" {
				continue
			}
			if err := s.imageRepo.UpdateImagePHashHex(member.ImageID, member.PHash); err != nil {
				return fmt.Errorf("persist image phash_hex for image %d: %w", member.ImageID, err)
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
			continue
		}

		images := make([]domain.DuplicateImage, len(relations))
		for j, rel := range relations {
			img, findErr := s.imageRepo.FindByID(rel.ImageID)
			if findErr != nil {
				continue
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
