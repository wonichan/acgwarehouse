package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/wonichan/acgwarehouse-backend/internal/config"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
)

const defaultImageMoveLimit = 1000

var (
	ErrImageMoveInvalidRequest = errors.New("invalid image move request")
	ErrImageMoveTagNotFound    = errors.New("image move tag not found")
)

type ImageMoveService struct {
	imageRepo      repository.ImageMoveQuery
	tagRepo        repository.TagRepository
	configProvider func() *config.Config
	moveFile       func(src, dst string) error
}

func NewImageMoveService(imageRepo repository.ImageMoveQuery, tagRepo repository.TagRepository, configProvider func() *config.Config) *ImageMoveService {
	return &ImageMoveService{
		imageRepo:      imageRepo,
		tagRepo:        tagRepo,
		configProvider: configProvider,
		moveFile:       moveFileWithCopyFallback,
	}
}

func (s *ImageMoveService) PreviewMove(ctx context.Context, req domain.ImageMoveRequest) (*domain.ImageMovePreview, error) {
	plan, err := s.buildPlan(ctx, req)
	if err != nil {
		return nil, err
	}

	preview := &domain.ImageMovePreview{
		TotalMatched: plan.totalMatched,
		Items:        plan.items,
	}
	for _, item := range plan.items {
		switch item.Status {
		case domain.ImageMoveStatusMovable:
			preview.Movable++
		case domain.ImageMoveStatusSkipped:
			preview.Skipped++
		}
	}

	return preview, nil
}

func (s *ImageMoveService) ExecuteMove(ctx context.Context, req domain.ImageMoveRequest) (*domain.ImageMoveResult, error) {
	plan, err := s.buildPlan(ctx, req)
	if err != nil {
		return nil, err
	}

	result := &domain.ImageMoveResult{
		TotalMatched: plan.totalMatched,
		Items:        make([]domain.ImageMoveItem, 0, len(plan.items)),
	}

	for _, planned := range plan.items {
		if planned.Status != domain.ImageMoveStatusMovable {
			planned.Status = domain.ImageMoveStatusSkipped
			result.Skipped++
			result.Items = append(result.Items, planned)
			continue
		}

		item := planned
		if err := s.moveFile(item.SourcePath, item.TargetPath); err != nil {
			item.Status = domain.ImageMoveStatusFailed
			item.Reason = classifyMoveError(err)
			result.Failed++
			result.Items = append(result.Items, item)
			continue
		}

		if err := s.imageRepo.UpdateImageLocation(ctx, item.ImageID, item.TargetPath, filepath.Base(item.TargetPath), plan.targetSourceRoot); err != nil {
			item.Status = domain.ImageMoveStatusFailed
			item.Reason = domain.ImageMoveReasonDBUpdateFailed
			if rollbackErr := s.moveFile(item.TargetPath, item.SourcePath); rollbackErr != nil {
				item.Reason = domain.ImageMoveReasonRollbackFailed
			}
			result.Failed++
			result.Items = append(result.Items, item)
			continue
		}

		item.Status = domain.ImageMoveStatusMoved
		item.Reason = ""
		result.Moved++
		result.Items = append(result.Items, item)
	}

	return result, nil
}

type imageMovePlan struct {
	totalMatched     int64
	targetSourceRoot string
	items            []domain.ImageMoveItem
}

func (s *ImageMoveService) buildPlan(ctx context.Context, req domain.ImageMoveRequest) (*imageMovePlan, error) {
	normalized, err := s.normalizeRequest(req)
	if err != nil {
		return nil, err
	}
	if err := s.ensureTagExists(ctx, normalized.TagID); err != nil {
		return nil, err
	}

	total, err := s.imageRepo.CountBySourceDirsAndTag(ctx, normalized.SourceDirs, normalized.TagID)
	if err != nil {
		return nil, err
	}
	images, err := s.imageRepo.FindBySourceDirsAndTag(ctx, normalized.SourceDirs, normalized.TagID, normalized.Limit, 0)
	if err != nil {
		return nil, err
	}

	plan := &imageMovePlan{
		totalMatched:     total,
		targetSourceRoot: s.resolveTargetSourceRoot(normalized.TargetDir),
		items:            make([]domain.ImageMoveItem, 0, len(images)),
	}

	for _, image := range images {
		item := domain.ImageMoveItem{
			ImageID:    image.ID,
			Filename:   image.Filename,
			SourcePath: image.Path,
			TargetPath: filepath.Join(normalized.TargetDir, image.Filename),
			Status:     domain.ImageMoveStatusMovable,
		}
		if !pathInAnyDir(image.Path, normalized.SourceDirs) {
			continue
		}
		if _, err := os.Stat(image.Path); err != nil {
			item.Status = domain.ImageMoveStatusSkipped
			item.Reason = domain.ImageMoveReasonSourceMissing
			if !errors.Is(err, os.ErrNotExist) {
				item.Reason = classifyMoveError(err)
			}
		} else if _, err := os.Stat(item.TargetPath); err == nil {
			item.Status = domain.ImageMoveStatusSkipped
			item.Reason = domain.ImageMoveReasonTargetExists
		} else if err != nil && !errors.Is(err, os.ErrNotExist) {
			item.Status = domain.ImageMoveStatusSkipped
			item.Reason = classifyMoveError(err)
		}
		plan.items = append(plan.items, item)
	}

	return plan, nil
}

func (s *ImageMoveService) normalizeRequest(req domain.ImageMoveRequest) (domain.ImageMoveRequest, error) {
	req.Conflict = strings.TrimSpace(req.Conflict)
	if req.Conflict == "" {
		req.Conflict = domain.ImageMoveConflictSkip
	}
	if req.Conflict != domain.ImageMoveConflictSkip {
		return req, fmt.Errorf("%w: conflict must be skip", ErrImageMoveInvalidRequest)
	}
	if req.TagID <= 0 {
		return req, fmt.Errorf("%w: tag_id is required", ErrImageMoveInvalidRequest)
	}
	if req.Limit <= 0 || req.Limit > defaultImageMoveLimit {
		req.Limit = defaultImageMoveLimit
	}

	sourceDirs := make([]string, 0, len(req.SourceDirs))
	seen := make(map[string]struct{}, len(req.SourceDirs))
	for _, sourceDir := range req.SourceDirs {
		normalized, err := normalizeAbsoluteDir(sourceDir)
		if err != nil {
			return req, fmt.Errorf("%w: %s", ErrImageMoveInvalidRequest, domain.ImageMoveReasonInvalidSourceDir)
		}
		key := pathCompareKey(normalized)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		sourceDirs = append(sourceDirs, normalized)
	}
	if len(sourceDirs) == 0 {
		return req, fmt.Errorf("%w: source_dirs is required", ErrImageMoveInvalidRequest)
	}

	targetDir, err := normalizeAbsoluteDir(req.TargetDir)
	if err != nil {
		return req, fmt.Errorf("%w: %s", ErrImageMoveInvalidRequest, domain.ImageMoveReasonInvalidTargetDir)
	}
	for _, sourceDir := range sourceDirs {
		if samePath(targetDir, sourceDir) || pathInDir(targetDir, sourceDir) {
			return req, fmt.Errorf("%w: target_dir must not be inside source_dirs", ErrImageMoveInvalidRequest)
		}
	}

	req.SourceDirs = sourceDirs
	req.TargetDir = targetDir
	return req, nil
}

func (s *ImageMoveService) ensureTagExists(ctx context.Context, tagID int64) error {
	_, err := s.tagRepo.FindByID(ctx, tagID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrImageMoveTagNotFound
	}
	return err
}

func (s *ImageMoveService) resolveTargetSourceRoot(targetDir string) string {
	cfg := (*config.Config)(nil)
	if s.configProvider != nil {
		cfg = s.configProvider()
	}
	if cfg == nil {
		return targetDir
	}
	for _, root := range cfg.Storage.ScanRoots {
		normalized, err := normalizeAbsoluteDir(root)
		if err != nil {
			continue
		}
		if samePath(targetDir, normalized) || pathInDir(targetDir, normalized) {
			return normalized
		}
	}
	return targetDir
}

func normalizeAbsoluteDir(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", errors.New("path is empty")
	}
	cleaned := filepath.Clean(trimmed)
	if !filepath.IsAbs(cleaned) {
		return "", fmt.Errorf("path %q is not absolute", path)
	}
	return cleaned, nil
}

func pathInAnyDir(path string, dirs []string) bool {
	for _, dir := range dirs {
		if pathInDir(path, dir) {
			return true
		}
	}
	return false
}

func pathInDir(path, dir string) bool {
	cleanPath := filepath.Clean(path)
	cleanDir := filepath.Clean(dir)
	if samePath(cleanPath, cleanDir) {
		return true
	}
	rel, err := filepath.Rel(cleanDir, cleanPath)
	if err != nil {
		return false
	}
	return rel != "." && rel != "" && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".." && !filepath.IsAbs(rel)
}

func samePath(a, b string) bool {
	return pathCompareKey(filepath.Clean(a)) == pathCompareKey(filepath.Clean(b))
}

func pathCompareKey(path string) string {
	if runtime.GOOS == "windows" {
		return strings.ToLower(path)
	}
	return path
}

func classifyMoveError(err error) string {
	if errors.Is(err, os.ErrPermission) {
		return domain.ImageMoveReasonPermissionDenied
	}
	if errors.Is(err, os.ErrNotExist) {
		return domain.ImageMoveReasonSourceMissing
	}
	return domain.ImageMoveReasonMoveFailed
}

func moveFileWithCopyFallback(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	return copyThenRemove(src, dst)
}

func copyThenRemove(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, srcInfo.Mode())
	if err != nil {
		return err
	}
	copyErr := func() error {
		if _, err := io.Copy(dstFile, srcFile); err != nil {
			return err
		}
		return dstFile.Close()
	}()
	if copyErr != nil {
		_ = dstFile.Close()
		_ = os.Remove(dst)
		return copyErr
	}

	dstInfo, err := os.Stat(dst)
	if err != nil {
		_ = os.Remove(dst)
		return err
	}
	if dstInfo.Size() != srcInfo.Size() {
		_ = os.Remove(dst)
		return fmt.Errorf("copied size mismatch: source=%d target=%d", srcInfo.Size(), dstInfo.Size())
	}

	return os.Remove(src)
}
