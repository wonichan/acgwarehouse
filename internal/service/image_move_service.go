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
	"sync"

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
	historyRepo    repository.ImageMoveHistoryRepository
	configProvider func() *config.Config
	moveFile       func(src, dst string) error
	jobsMu         sync.RWMutex
	jobs           map[int64]context.CancelFunc
}

func NewImageMoveService(imageRepo repository.ImageMoveQuery, tagRepo repository.TagRepository, historyRepo repository.ImageMoveHistoryRepository, configProvider func() *config.Config) *ImageMoveService {
	return &ImageMoveService{
		imageRepo:      imageRepo,
		tagRepo:        tagRepo,
		historyRepo:    historyRepo,
		configProvider: configProvider,
		moveFile:       moveFileWithCopyFallback,
		jobs:           make(map[int64]context.CancelFunc),
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
	return s.executePlan(ctx, plan, nil)
}

func (s *ImageMoveService) CreateMoveJob(ctx context.Context, req domain.ImageMoveRequest) (*domain.ImageMoveBatch, error) {
	plan, err := s.buildPlan(ctx, req)
	if err != nil {
		return nil, err
	}
	batch := &domain.ImageMoveBatch{
		TagID:            plan.request.TagID,
		SourceDirs:       plan.request.SourceDirs,
		TargetDir:        plan.request.TargetDir,
		ConflictStrategy: plan.request.Conflict,
		TotalMatched:     plan.totalMatched,
		Status:           domain.ImageMoveBatchStatusQueued,
		Progress: domain.ImageMoveProgress{
			Total: plan.totalMatched,
		},
	}
	if s.historyRepo == nil {
		return nil, fmt.Errorf("%w: image move history repository unavailable", ErrImageMoveInvalidRequest)
	}
	if err := s.historyRepo.CreateImageMoveBatch(ctx, batch); err != nil {
		return nil, err
	}

	jobCtx, cancel := context.WithCancel(context.Background())
	s.jobsMu.Lock()
	s.jobs[batch.ID] = cancel
	s.jobsMu.Unlock()

	go s.runMoveJob(jobCtx, batch.ID, plan.request, plan.targetSourceRoot, plan.totalMatched)
	return s.historyRepo.FindImageMoveBatch(ctx, batch.ID)
}

func (s *ImageMoveService) GetMoveJob(ctx context.Context, id int64) (*domain.ImageMoveBatch, error) {
	if s.historyRepo == nil {
		return nil, sql.ErrNoRows
	}
	return s.historyRepo.FindImageMoveBatch(ctx, id)
}

func (s *ImageMoveService) ListMoveHistory(ctx context.Context, limit int) ([]domain.ImageMoveBatch, error) {
	if s.historyRepo == nil {
		return []domain.ImageMoveBatch{}, nil
	}
	return s.historyRepo.ListImageMoveBatches(ctx, limit)
}

func (s *ImageMoveService) CancelMoveJob(ctx context.Context, id int64) (*domain.ImageMoveBatch, error) {
	s.jobsMu.Lock()
	cancel := s.jobs[id]
	if cancel != nil {
		cancel()
	}
	delete(s.jobs, id)
	s.jobsMu.Unlock()

	batch, err := s.GetMoveJob(ctx, id)
	if err != nil {
		return nil, err
	}
	if cancel == nil && batch.Status == domain.ImageMoveBatchStatusQueued {
		batch.Status = domain.ImageMoveBatchStatusCancelled
		if err := s.historyRepo.UpdateImageMoveBatch(ctx, batch); err != nil {
			return nil, err
		}
	}
	return s.GetMoveJob(ctx, id)
}

func (s *ImageMoveService) runMoveJob(ctx context.Context, batchID int64, req domain.ImageMoveRequest, targetSourceRoot string, totalMatched int64) {
	defer func() {
		s.jobsMu.Lock()
		delete(s.jobs, batchID)
		s.jobsMu.Unlock()
	}()
	batch := &domain.ImageMoveBatch{
		ID:               batchID,
		TagID:            req.TagID,
		SourceDirs:       req.SourceDirs,
		TargetDir:        req.TargetDir,
		ConflictStrategy: req.Conflict,
		TotalMatched:     totalMatched,
		Status:           domain.ImageMoveBatchStatusRunning,
	}
	_ = s.historyRepo.UpdateImageMoveBatch(context.Background(), batch)
	result, err := s.executeJobPlan(ctx, batch, req, targetSourceRoot)
	if err != nil && errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
		batch.Status = domain.ImageMoveBatchStatusCancelled
	} else if err != nil || result.Failed > 0 {
		batch.Status = domain.ImageMoveBatchStatusFailed
	} else {
		batch.Status = domain.ImageMoveBatchStatusCompleted
	}
	if result != nil {
		batch.Moved = result.Moved
		batch.Skipped = result.Skipped
		batch.Failed = result.Failed
	}
	_ = s.historyRepo.UpdateImageMoveBatch(context.Background(), batch)
}

func (s *ImageMoveService) executeJobPlan(ctx context.Context, batch *domain.ImageMoveBatch, req domain.ImageMoveRequest, targetSourceRoot string) (*domain.ImageMoveResult, error) {
	plan, err := s.buildFullJobPlan(ctx, req, targetSourceRoot, batch.TotalMatched)
	if err != nil {
		return &domain.ImageMoveResult{TotalMatched: batch.TotalMatched}, err
	}
	return s.executePlan(ctx, plan, batch)
}

func (s *ImageMoveService) buildFullJobPlan(ctx context.Context, req domain.ImageMoveRequest, targetSourceRoot string, totalMatched int64) (*imageMovePlan, error) {
	plan := &imageMovePlan{
		totalMatched:     totalMatched,
		targetSourceRoot: targetSourceRoot,
		request:          req,
	}
	plannedTargets := make(map[string]struct{})

	for offset := 0; ; offset += req.Limit {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		images, err := s.imageRepo.FindBySourceDirsAndTag(ctx, req.SourceDirs, req.TagID, req.Limit, offset)
		if err != nil {
			return nil, err
		}
		if len(images) == 0 {
			return plan, nil
		}
		if err := s.appendPlanItems(plan, images, plannedTargets); err != nil {
			return nil, err
		}
		if len(images) < req.Limit {
			return plan, nil
		}
	}
}

func (s *ImageMoveService) executePlan(ctx context.Context, plan *imageMovePlan, batch *domain.ImageMoveBatch) (*domain.ImageMoveResult, error) {
	result := &domain.ImageMoveResult{
		TotalMatched: plan.totalMatched,
		Items:        make([]domain.ImageMoveItem, 0, len(plan.items)),
	}
	if err := s.executePlanInto(ctx, plan, batch, result); err != nil {
		return result, err
	}
	return result, nil
}

func (s *ImageMoveService) executePlanInto(ctx context.Context, plan *imageMovePlan, batch *domain.ImageMoveBatch, result *domain.ImageMoveResult) error {
	for _, planned := range plan.items {
		if err := ctx.Err(); err != nil {
			return err
		}
		if planned.Status != domain.ImageMoveStatusMovable {
			planned.Status = domain.ImageMoveStatusSkipped
			result.Skipped++
			result.Items = append(result.Items, planned)
			s.recordMoveItem(ctx, batch, planned, result)
			continue
		}

		item := planned
		moved, reason := s.executeMoveItem(ctx, item, plan.targetSourceRoot)
		if moved {
			item.Status = domain.ImageMoveStatusMoved
			item.Reason = ""
			result.Moved++
		} else {
			item.Status = domain.ImageMoveStatusFailed
			item.Reason = reason
			item.Retryable = domain.ImageMoveReasonIsRetryable(item.Reason)
			result.Failed++
		}
		result.Items = append(result.Items, item)
		s.recordMoveItem(ctx, batch, item, result)
	}
	return nil
}

func (s *ImageMoveService) executeMoveItem(ctx context.Context, item domain.ImageMoveItem, targetSourceRoot string) (bool, string) {
	backupPath := ""
	if item.Overwritten {
		var err error
		backupPath, err = reserveOverwriteBackup(item.TargetPath)
		if err != nil {
			return false, classifyMoveError(err)
		}
		if err := s.moveFile(item.TargetPath, backupPath); err != nil {
			_ = os.Remove(backupPath)
			return false, classifyMoveError(err)
		}
	}

	if err := s.moveFile(item.SourcePath, item.TargetPath); err != nil {
		if backupPath != "" {
			_ = s.moveFile(backupPath, item.TargetPath)
		}
		return false, classifyMoveError(err)
	}

	if err := s.imageRepo.UpdateImageLocation(ctx, item.ImageID, item.TargetPath, filepath.Base(item.TargetPath), targetSourceRoot); err != nil {
		reason := domain.ImageMoveReasonDBUpdateFailed
		if rollbackErr := s.moveFile(item.TargetPath, item.SourcePath); rollbackErr != nil {
			reason = domain.ImageMoveReasonRollbackFailed
		}
		if backupPath != "" {
			if restoreErr := s.moveFile(backupPath, item.TargetPath); restoreErr != nil && reason == domain.ImageMoveReasonDBUpdateFailed {
				reason = domain.ImageMoveReasonRollbackFailed
			}
		}
		return false, reason
	}

	if backupPath != "" {
		_ = os.Remove(backupPath)
	}
	return true, ""
}

func (s *ImageMoveService) recordMoveItem(ctx context.Context, batch *domain.ImageMoveBatch, item domain.ImageMoveItem, result *domain.ImageMoveResult) {
	if batch == nil || s.historyRepo == nil {
		return
	}
	_ = s.historyRepo.AddImageMoveItem(context.Background(), batch.ID, item)
	batch.Moved = result.Moved
	batch.Skipped = result.Skipped
	batch.Failed = result.Failed
	if item.Status == domain.ImageMoveStatusMovable || item.Status == domain.ImageMoveStatusMoved || item.Status == domain.ImageMoveStatusSkipped || item.Status == domain.ImageMoveStatusFailed {
		batch.Status = domain.ImageMoveBatchStatusRunning
	}
	batch.Progress = domain.ImageMoveProgress{
		Total:       batch.TotalMatched,
		Processed:   result.Moved + result.Skipped + result.Failed,
		Moved:       result.Moved,
		Skipped:     result.Skipped,
		Failed:      result.Failed,
		CurrentPath: item.SourcePath,
	}
	_ = s.historyRepo.UpdateImageMoveBatch(ctx, batch)
}

type imageMovePlan struct {
	totalMatched     int64
	targetSourceRoot string
	request          domain.ImageMoveRequest
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
		request:          normalized,
		items:            make([]domain.ImageMoveItem, 0, len(images)),
	}
	plannedTargets := make(map[string]struct{}, len(images))
	if err := s.appendPlanItems(plan, images, plannedTargets); err != nil {
		return nil, err
	}

	return plan, nil
}

func (s *ImageMoveService) appendPlanItems(plan *imageMovePlan, images []domain.Image, plannedTargets map[string]struct{}) error {
	for _, image := range images {
		targetPath := filepath.Join(plan.request.TargetDir, image.Filename)
		if plan.request.Conflict == domain.ImageMoveConflictRename {
			targetPath = nextAvailableMoveTarget(targetPath, plannedTargets)
		}
		item := domain.ImageMoveItem{
			ImageID:    image.ID,
			Filename:   image.Filename,
			SourcePath: image.Path,
			TargetPath: targetPath,
			Status:     domain.ImageMoveStatusMovable,
		}
		if !pathInAnyDir(image.Path, plan.request.SourceDirs) {
			continue
		}
		if _, err := os.Stat(image.Path); err != nil {
			item.Status = domain.ImageMoveStatusSkipped
			item.Reason = domain.ImageMoveReasonSourceMissing
			if !errors.Is(err, os.ErrNotExist) {
				item.Reason = classifyMoveError(err)
			}
		} else if _, err := os.Stat(item.TargetPath); err == nil {
			switch plan.request.Conflict {
			case domain.ImageMoveConflictOverwrite:
				item.Overwritten = true
			default:
				item.Status = domain.ImageMoveStatusSkipped
				item.Reason = domain.ImageMoveReasonTargetExists
			}
		} else if err != nil && !errors.Is(err, os.ErrNotExist) {
			item.Status = domain.ImageMoveStatusSkipped
			item.Reason = classifyMoveError(err)
		}
		if item.Reason != "" {
			item.Retryable = domain.ImageMoveReasonIsRetryable(item.Reason)
		}
		plannedTargets[pathCompareKey(item.TargetPath)] = struct{}{}
		plan.items = append(plan.items, item)
	}
	return nil
}

func (s *ImageMoveService) normalizeRequest(req domain.ImageMoveRequest) (domain.ImageMoveRequest, error) {
	req.Conflict = strings.TrimSpace(req.Conflict)
	if req.Conflict == "" {
		req.Conflict = domain.ImageMoveConflictSkip
	}
	switch req.Conflict {
	case domain.ImageMoveConflictSkip, domain.ImageMoveConflictRename, domain.ImageMoveConflictOverwrite:
	default:
		return req, fmt.Errorf("%w: conflict must be skip, rename, or overwrite", ErrImageMoveInvalidRequest)
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
	if isSystemCriticalDir(targetDir) {
		return req, fmt.Errorf("%w: %s", ErrImageMoveInvalidRequest, domain.ImageMoveReasonSystemTargetDir)
	}
	for _, sourceDir := range sourceDirs {
		if samePath(targetDir, sourceDir) {
			return req, fmt.Errorf("%w: target_dir must not equal source_dirs", ErrImageMoveInvalidRequest)
		}
		if pathInDir(targetDir, sourceDir) && !req.AllowTargetInsideSource {
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
	if hasIllegalPathChars(trimmed) {
		return "", fmt.Errorf("path %q contains illegal characters", path)
	}
	cleaned := filepath.Clean(trimmed)
	if !filepath.IsAbs(cleaned) {
		return "", fmt.Errorf("path %q is not absolute", path)
	}
	resolved, err := filepath.EvalSymlinks(cleaned)
	if err == nil {
		cleaned = resolved
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	return filepath.Clean(cleaned), nil
}

func hasIllegalPathChars(path string) bool {
	if strings.ContainsAny(path, "\x00") {
		return true
	}
	if runtime.GOOS == "windows" {
		volume := filepath.VolumeName(path)
		rest := strings.TrimPrefix(path, volume)
		return strings.ContainsAny(rest, `<>:"|?*`)
	}
	return false
}

func isSystemCriticalDir(path string) bool {
	cleaned := filepath.Clean(path)
	if runtime.GOOS == "windows" {
		key := pathCompareKey(cleaned)
		systemRoot := os.Getenv("SystemRoot")
		windir := os.Getenv("windir")
		programFiles := []string{os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)"), os.Getenv("ProgramData")}
		for _, protected := range append([]string{systemRoot, windir}, programFiles...) {
			if protected == "" {
				continue
			}
			protected = filepath.Clean(protected)
			if key == pathCompareKey(protected) || pathInDir(cleaned, protected) {
				return true
			}
		}
		volume := filepath.VolumeName(cleaned)
		return volume != "" && samePath(cleaned, volume+string(filepath.Separator))
	}
	protectedDirs := []string{"/", "/bin", "/boot", "/dev", "/etc", "/lib", "/lib64", "/proc", "/root", "/sbin", "/sys", "/usr", "/var"}
	for _, protected := range protectedDirs {
		if samePath(cleaned, protected) {
			return true
		}
	}
	return false
}

func nextAvailableMoveTarget(targetPath string, reserved map[string]struct{}) string {
	if _, ok := reserved[pathCompareKey(targetPath)]; !ok {
		if _, err := os.Stat(targetPath); errors.Is(err, os.ErrNotExist) {
			return targetPath
		}
	}
	ext := filepath.Ext(targetPath)
	base := strings.TrimSuffix(filepath.Base(targetPath), ext)
	dir := filepath.Dir(targetPath)
	for i := 1; i < 100000; i++ {
		candidate := filepath.Join(dir, fmt.Sprintf("%s (%d)%s", base, i, ext))
		if _, ok := reserved[pathCompareKey(candidate)]; ok {
			continue
		}
		if _, err := os.Stat(candidate); errors.Is(err, os.ErrNotExist) {
			return candidate
		}
	}
	return targetPath
}

func reserveOverwriteBackup(targetPath string) (string, error) {
	dir := filepath.Dir(targetPath)
	base := filepath.Base(targetPath)
	for i := 0; i < 100000; i++ {
		candidate := filepath.Join(dir, fmt.Sprintf(".%s.acgwarehouse-overwrite-%d.tmp", base, i))
		file, err := os.OpenFile(candidate, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err == nil {
			if closeErr := file.Close(); closeErr != nil {
				_ = os.Remove(candidate)
				return "", closeErr
			}
			if err := os.Remove(candidate); err != nil {
				return "", err
			}
			return candidate, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return "", err
		}
	}
	return "", fmt.Errorf("reserve overwrite backup for %q: no available name", targetPath)
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
