package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/wonichan/acgwarehouse-backend/internal/config"
	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
)

type imageMoveServiceTestEnv struct {
	db           *sql.DB
	imageRepo    repository.ImageRepository
	tagRepo      repository.TagRepository
	imageTagRepo repository.ImageTagRepository
	tag          *domain.Tag
	sourceDir    string
	targetDir    string
	svc          *ImageMoveService
}

func setupImageMoveServiceTest(t *testing.T) *imageMoveServiceTestEnv {
	t.Helper()
	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "image-move-service.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := repository.EnsureScanSchema(db); err != nil {
		t.Fatalf("EnsureScanSchema() error = %v", err)
	}

	sourceDir := filepath.Join(t.TempDir(), "source")
	targetDir := filepath.Join(t.TempDir(), "target")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}

	imageRepo := repository.NewImageRepository(db)
	historyRepo := repository.NewImageMoveHistoryRepository(db)
	tagRepo := repository.NewTagRepository(db)
	imageTagRepo := repository.NewImageTagRepository(db)
	tag := &domain.Tag{PreferredLabel: "target", Slug: "target", ReviewState: "confirmed"}
	if err := tagRepo.Save(context.Background(), tag); err != nil {
		t.Fatalf("save tag: %v", err)
	}

	svc := NewImageMoveService(imageRepo, tagRepo, historyRepo, func() *config.Config {
		return &config.Config{Storage: config.StorageConfig{ScanRoots: []string{targetDir}}}
	})

	return &imageMoveServiceTestEnv{
		db:           db,
		imageRepo:    imageRepo,
		tagRepo:      tagRepo,
		imageTagRepo: imageTagRepo,
		tag:          tag,
		sourceDir:    sourceDir,
		targetDir:    targetDir,
		svc:          svc,
	}
}

func TestImageMoveServicePreviewAppliesPathBoundaryAndConflictRules(t *testing.T) {
	t.Parallel()
	env := setupImageMoveServiceTest(t)
	ctx := context.Background()

	movable := env.saveImageWithFile(t, "move.png", env.sourceDir, "move")
	conflict := env.saveImageWithFile(t, "conflict.png", env.sourceDir, "conflict")
	missing := env.saveImageRecord(t, filepath.Join(env.sourceDir, "missing.png"), env.sourceDir)
	siblingDir := env.sourceDir + "2"
	if err := os.MkdirAll(siblingDir, 0755); err != nil {
		t.Fatalf("mkdir sibling: %v", err)
	}
	sibling := env.saveImageWithFile(t, "sibling.png", siblingDir, "sibling")
	for _, imageID := range []int64{movable.ID, conflict.ID, missing.ID, sibling.ID} {
		env.saveImageTag(t, imageID)
	}
	if err := os.WriteFile(filepath.Join(env.targetDir, "conflict.png"), []byte("exists"), 0644); err != nil {
		t.Fatalf("write conflict target: %v", err)
	}

	preview, err := env.svc.PreviewMove(ctx, domain.ImageMoveRequest{
		SourceDirs: []string{env.sourceDir},
		TagID:      env.tag.ID,
		TargetDir:  env.targetDir,
	})
	if err != nil {
		t.Fatalf("PreviewMove() error = %v", err)
	}

	if preview.TotalMatched != 3 {
		t.Fatalf("TotalMatched = %d, want 3", preview.TotalMatched)
	}
	if preview.Movable != 1 || preview.Skipped != 2 {
		t.Fatalf("Movable/Skipped = %d/%d, want 1/2", preview.Movable, preview.Skipped)
	}
	assertImageMoveItem(t, preview.Items, movable.ID, domain.ImageMoveStatusMovable, "")
	assertImageMoveItem(t, preview.Items, conflict.ID, domain.ImageMoveStatusSkipped, domain.ImageMoveReasonTargetExists)
	assertImageMoveItem(t, preview.Items, missing.ID, domain.ImageMoveStatusSkipped, domain.ImageMoveReasonSourceMissing)
}

func TestImageMoveServiceExecuteMovesFileAndKeepsTagAssociation(t *testing.T) {
	t.Parallel()
	env := setupImageMoveServiceTest(t)
	ctx := context.Background()

	image := env.saveImageWithFile(t, "move.png", env.sourceDir, "payload")
	env.saveImageTag(t, image.ID)

	result, err := env.svc.ExecuteMove(ctx, domain.ImageMoveRequest{
		SourceDirs: []string{env.sourceDir},
		TagID:      env.tag.ID,
		TargetDir:  env.targetDir,
	})
	if err != nil {
		t.Fatalf("ExecuteMove() error = %v", err)
	}
	if result.Moved != 1 || result.Skipped != 0 || result.Failed != 0 {
		t.Fatalf("Moved/Skipped/Failed = %d/%d/%d, want 1/0/0", result.Moved, result.Skipped, result.Failed)
	}

	targetPath := filepath.Join(env.targetDir, "move.png")
	if _, err := os.Stat(filepath.Join(env.sourceDir, "move.png")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("source stat error = %v, want os.ErrNotExist", err)
	}
	data, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if string(data) != "payload" {
		t.Fatalf("target payload = %q, want payload", data)
	}

	updated, err := env.imageRepo.FindByID(image.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if updated.Path != targetPath || updated.Filename != "move.png" || updated.SourceRoot != env.targetDir {
		t.Fatalf("updated image = (%q, %q, %q)", updated.Path, updated.Filename, updated.SourceRoot)
	}
	tags, err := env.imageTagRepo.FindByImageID(ctx, image.ID)
	if err != nil {
		t.Fatalf("FindByImageID() error = %v", err)
	}
	if len(tags) != 1 || tags[0].TagID != env.tag.ID {
		t.Fatalf("tags after move = %+v, want original tag", tags)
	}
}

func TestImageMoveServicePreviewSupportsRenameAndOverwriteConflicts(t *testing.T) {
	t.Parallel()
	env := setupImageMoveServiceTest(t)
	ctx := context.Background()

	renameImage := env.saveImageWithFile(t, "conflict.png", env.sourceDir, "rename")
	overwriteImage := env.saveImageWithFile(t, "overwrite.png", env.sourceDir, "overwrite")
	env.saveImageTag(t, renameImage.ID)
	env.saveImageTag(t, overwriteImage.ID)
	if err := os.WriteFile(filepath.Join(env.targetDir, "conflict.png"), []byte("exists"), 0644); err != nil {
		t.Fatalf("write conflict target: %v", err)
	}
	if err := os.WriteFile(filepath.Join(env.targetDir, "overwrite.png"), []byte("old"), 0644); err != nil {
		t.Fatalf("write overwrite target: %v", err)
	}

	renamePreview, err := env.svc.PreviewMove(ctx, domain.ImageMoveRequest{
		SourceDirs: []string{env.sourceDir},
		TagID:      env.tag.ID,
		TargetDir:  env.targetDir,
		Conflict:   domain.ImageMoveConflictRename,
	})
	if err != nil {
		t.Fatalf("PreviewMove(rename) error = %v", err)
	}
	renamed := findImageMoveItem(t, renamePreview.Items, renameImage.ID)
	if renamed.Status != domain.ImageMoveStatusMovable || renamed.TargetPath != filepath.Join(env.targetDir, "conflict (1).png") {
		t.Fatalf("rename item = %+v", renamed)
	}

	overwritePreview, err := env.svc.PreviewMove(ctx, domain.ImageMoveRequest{
		SourceDirs: []string{env.sourceDir},
		TagID:      env.tag.ID,
		TargetDir:  env.targetDir,
		Conflict:   domain.ImageMoveConflictOverwrite,
	})
	if err != nil {
		t.Fatalf("PreviewMove(overwrite) error = %v", err)
	}
	overwrite := findImageMoveItem(t, overwritePreview.Items, overwriteImage.ID)
	if overwrite.Status != domain.ImageMoveStatusMovable || !overwrite.Overwritten {
		t.Fatalf("overwrite item = %+v, want movable overwritten", overwrite)
	}
}

func TestImageMoveServiceExecuteRecordsHistory(t *testing.T) {
	t.Parallel()
	env := setupImageMoveServiceTest(t)
	ctx := context.Background()

	image := env.saveImageWithFile(t, "history.png", env.sourceDir, "payload")
	env.saveImageTag(t, image.ID)

	job, err := env.svc.CreateMoveJob(ctx, domain.ImageMoveRequest{
		SourceDirs: []string{env.sourceDir},
		TagID:      env.tag.ID,
		TargetDir:  env.targetDir,
	})
	if err != nil {
		t.Fatalf("CreateMoveJob() error = %v", err)
	}

	var refreshed *domain.ImageMoveBatch
	for i := 0; i < 50; i++ {
		refreshed, err = env.svc.GetMoveJob(ctx, job.ID)
		if err != nil {
			t.Fatalf("GetMoveJob() error = %v", err)
		}
		if refreshed.Status == domain.ImageMoveBatchStatusCompleted || refreshed.Status == domain.ImageMoveBatchStatusFailed {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if refreshed.Status != domain.ImageMoveBatchStatusCompleted {
		t.Fatalf("job status = %q, want completed; job=%+v", refreshed.Status, refreshed)
	}
	if refreshed.Moved != 1 || len(refreshed.Items) != 1 || refreshed.Items[0].Status != domain.ImageMoveStatusMoved {
		t.Fatalf("job history = %+v", refreshed)
	}
}

func TestImageMoveServiceRejectsUnsafeTargetDirs(t *testing.T) {
	t.Parallel()
	env := setupImageMoveServiceTest(t)
	ctx := context.Background()

	_, err := env.svc.PreviewMove(ctx, domain.ImageMoveRequest{
		SourceDirs: []string{env.sourceDir},
		TagID:      env.tag.ID,
		TargetDir:  filepath.Join(env.sourceDir, "nested"),
	})
	if !errors.Is(err, ErrImageMoveInvalidRequest) {
		t.Fatalf("PreviewMove nested target error = %v, want invalid request", err)
	}

	_, err = env.svc.PreviewMove(ctx, domain.ImageMoveRequest{
		SourceDirs:              []string{env.sourceDir},
		TagID:                   env.tag.ID,
		TargetDir:               filepath.Join(env.sourceDir, "nested"),
		AllowTargetInsideSource: true,
	})
	if err != nil {
		t.Fatalf("PreviewMove allow nested target error = %v", err)
	}
}

func TestImageMoveServiceExecuteRollsBackFileWhenDBUpdateFails(t *testing.T) {
	t.Parallel()
	env := setupImageMoveServiceTest(t)
	ctx := context.Background()

	image := env.saveImageWithFile(t, "move.png", env.sourceDir, "payload")
	env.saveImageTag(t, image.ID)

	failingRepo := &failingImageMoveRepo{ImageMoveQuery: env.imageRepo, updateErr: errors.New("db down")}
	svc := NewImageMoveService(failingRepo, env.tagRepo, nil, func() *config.Config {
		return &config.Config{Storage: config.StorageConfig{ScanRoots: []string{env.targetDir}}}
	})

	result, err := svc.ExecuteMove(ctx, domain.ImageMoveRequest{
		SourceDirs: []string{env.sourceDir},
		TagID:      env.tag.ID,
		TargetDir:  env.targetDir,
	})
	if err != nil {
		t.Fatalf("ExecuteMove() error = %v", err)
	}
	if result.Failed != 1 || result.Moved != 0 {
		t.Fatalf("Failed/Moved = %d/%d, want 1/0", result.Failed, result.Moved)
	}
	assertImageMoveItem(t, result.Items, image.ID, domain.ImageMoveStatusFailed, domain.ImageMoveReasonDBUpdateFailed)
	if _, err := os.Stat(filepath.Join(env.sourceDir, "move.png")); err != nil {
		t.Fatalf("source should be rolled back, stat error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(env.targetDir, "move.png")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("target stat error = %v, want os.ErrNotExist", err)
	}
}

func TestImageMoveServiceOverwriteRestoresOriginalTargetWhenDBUpdateFails(t *testing.T) {
	t.Parallel()
	env := setupImageMoveServiceTest(t)
	ctx := context.Background()

	image := env.saveImageWithFile(t, "overwrite.png", env.sourceDir, "new")
	env.saveImageTag(t, image.ID)
	targetPath := filepath.Join(env.targetDir, "overwrite.png")
	if err := os.WriteFile(targetPath, []byte("old"), 0644); err != nil {
		t.Fatalf("write target: %v", err)
	}

	failingRepo := &failingImageMoveRepo{ImageMoveQuery: env.imageRepo, updateErr: errors.New("db down")}
	svc := NewImageMoveService(failingRepo, env.tagRepo, nil, func() *config.Config {
		return &config.Config{Storage: config.StorageConfig{ScanRoots: []string{env.targetDir}}}
	})

	result, err := svc.ExecuteMove(ctx, domain.ImageMoveRequest{
		SourceDirs: []string{env.sourceDir},
		TagID:      env.tag.ID,
		TargetDir:  env.targetDir,
		Conflict:   domain.ImageMoveConflictOverwrite,
	})
	if err != nil {
		t.Fatalf("ExecuteMove() error = %v", err)
	}
	if result.Failed != 1 || result.Moved != 0 {
		t.Fatalf("Failed/Moved = %d/%d, want 1/0", result.Failed, result.Moved)
	}
	assertImageMoveItem(t, result.Items, image.ID, domain.ImageMoveStatusFailed, domain.ImageMoveReasonDBUpdateFailed)
	sourceData, err := os.ReadFile(filepath.Join(env.sourceDir, "overwrite.png"))
	if err != nil {
		t.Fatalf("read restored source: %v", err)
	}
	if string(sourceData) != "new" {
		t.Fatalf("source payload = %q, want new", sourceData)
	}
	targetData, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("read restored target: %v", err)
	}
	if string(targetData) != "old" {
		t.Fatalf("target payload = %q, want old", targetData)
	}
}

func TestImageMoveServiceJobProcessesAllPages(t *testing.T) {
	t.Parallel()
	env := setupImageMoveServiceTest(t)
	ctx := context.Background()

	const imageCount = 13
	for i := 0; i < imageCount; i++ {
		image := env.saveImageWithFile(t, "page-"+intToString(i)+".png", env.sourceDir, "payload")
		env.saveImageTag(t, image.ID)
	}

	job, err := env.svc.CreateMoveJob(ctx, domain.ImageMoveRequest{
		SourceDirs: []string{env.sourceDir},
		TagID:      env.tag.ID,
		TargetDir:  env.targetDir,
		Limit:      5,
	})
	if err != nil {
		t.Fatalf("CreateMoveJob() error = %v", err)
	}

	refreshed := waitImageMoveJob(t, env.svc, job.ID, domain.ImageMoveBatchStatusCompleted, domain.ImageMoveBatchStatusFailed, domain.ImageMoveBatchStatusCancelled)
	if refreshed.Status != domain.ImageMoveBatchStatusCompleted {
		t.Fatalf("job status = %q, want completed; job=%+v", refreshed.Status, refreshed)
	}
	if refreshed.TotalMatched != imageCount || refreshed.Moved != imageCount || refreshed.Progress.Processed != imageCount {
		t.Fatalf("job counts = total:%d moved:%d processed:%d, want %d", refreshed.TotalMatched, refreshed.Moved, refreshed.Progress.Processed, imageCount)
	}
	if len(refreshed.Items) != imageCount {
		t.Fatalf("history item count = %d, want %d", len(refreshed.Items), imageCount)
	}
}

func TestImageMoveServiceCancelJobFinalStateWinsOverWorker(t *testing.T) {
	t.Parallel()
	env := setupImageMoveServiceTest(t)
	ctx := context.Background()

	image := env.saveImageWithFile(t, "cancel.png", env.sourceDir, "payload")
	env.saveImageTag(t, image.ID)
	releaseMove := make(chan struct{})
	moveStarted := make(chan struct{})
	var signalStarted sync.Once
	env.svc.moveFile = func(src, dst string) error {
		signalStarted.Do(func() {
			close(moveStarted)
		})
		<-releaseMove
		return moveFileWithCopyFallback(src, dst)
	}

	job, err := env.svc.CreateMoveJob(ctx, domain.ImageMoveRequest{
		SourceDirs: []string{env.sourceDir},
		TagID:      env.tag.ID,
		TargetDir:  env.targetDir,
	})
	if err != nil {
		t.Fatalf("CreateMoveJob() error = %v", err)
	}

	select {
	case <-moveStarted:
	case <-time.After(time.Second):
		t.Fatal("move did not start")
	}
	if _, err := env.svc.CancelMoveJob(ctx, job.ID); err != nil {
		t.Fatalf("CancelMoveJob() error = %v", err)
	}
	close(releaseMove)

	refreshed := waitImageMoveJob(t, env.svc, job.ID, domain.ImageMoveBatchStatusCancelled, domain.ImageMoveBatchStatusCompleted, domain.ImageMoveBatchStatusFailed)
	if refreshed.Status != domain.ImageMoveBatchStatusCancelled {
		t.Fatalf("job status = %q, want cancelled; job=%+v", refreshed.Status, refreshed)
	}
}

func (env *imageMoveServiceTestEnv) saveImageWithFile(t *testing.T, filename, dir, payload string) *domain.Image {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(payload), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return env.saveImageRecord(t, path, dir)
}

func (env *imageMoveServiceTestEnv) saveImageRecord(t *testing.T, path, sourceRoot string) *domain.Image {
	t.Helper()
	image := &domain.Image{
		Path:       path,
		Filename:   filepath.Base(path),
		SourceRoot: sourceRoot,
		FileSize:   10,
		Format:     "png",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if _, err := env.imageRepo.SaveImage(image); err != nil {
		t.Fatalf("save image: %v", err)
	}
	return image
}

func (env *imageMoveServiceTestEnv) saveImageTag(t *testing.T, imageID int64) {
	t.Helper()
	if err := env.imageTagRepo.Save(context.Background(), &domain.ImageTag{ImageID: imageID, TagID: env.tag.ID, ReviewState: domain.ReviewStateConfirmed}); err != nil {
		t.Fatalf("save image-tag: %v", err)
	}
}

type failingImageMoveRepo struct {
	repository.ImageMoveQuery
	updateErr error
}

func (r *failingImageMoveRepo) UpdateImageLocation(ctx context.Context, imageID int64, path, filename, sourceRoot string) error {
	return r.updateErr
}

func assertImageMoveItem(t *testing.T, items []domain.ImageMoveItem, imageID int64, status, reason string) {
	t.Helper()
	item := findImageMoveItem(t, items, imageID)
	if item.Status != status || item.Reason != reason {
		t.Fatalf("item %d status/reason = %q/%q, want %q/%q", imageID, item.Status, item.Reason, status, reason)
	}
}

func findImageMoveItem(t *testing.T, items []domain.ImageMoveItem, imageID int64) domain.ImageMoveItem {
	t.Helper()
	for _, item := range items {
		if item.ImageID != imageID {
			continue
		}
		return item
	}
	t.Fatalf("item %d not found in %+v", imageID, items)
	return domain.ImageMoveItem{}
}

func waitImageMoveJob(t *testing.T, svc *ImageMoveService, jobID int64, terminalStatuses ...string) *domain.ImageMoveBatch {
	t.Helper()
	wanted := make(map[string]struct{}, len(terminalStatuses))
	for _, status := range terminalStatuses {
		wanted[status] = struct{}{}
	}
	var refreshed *domain.ImageMoveBatch
	var err error
	for i := 0; i < 200; i++ {
		refreshed, err = svc.GetMoveJob(context.Background(), jobID)
		if err != nil {
			t.Fatalf("GetMoveJob() error = %v", err)
		}
		if _, ok := wanted[refreshed.Status]; ok {
			return refreshed
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("job did not reach terminal status; last=%+v", refreshed)
	return refreshed
}

func intToString(value int) string {
	return fmt.Sprintf("%d", value)
}
