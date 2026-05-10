package repository

import (
	"context"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

// HybridImageRepository delegates reads to D1 and writes to local SQLite.
type HybridImageRepository struct {
	reader  ImageReader
	gallery GalleryImageQuery
	writer  ImageMutationStore
	backfill BackfillImageQuery
	move    ImageMoveQuery
}

func NewHybridImageRepository(reader ImageReader, gallery GalleryImageQuery, writer ImageMutationStore, backfill BackfillImageQuery, move ImageMoveQuery) *HybridImageRepository {
	return &HybridImageRepository{
		reader:   reader,
		gallery:  gallery,
		writer:   writer,
		backfill: backfill,
		move:     move,
	}
}

// ImageReader delegates

func (h *HybridImageRepository) FindByID(id int64) (*domain.Image, error) {
	return h.reader.FindByID(id)
}

func (h *HybridImageRepository) FindByPath(path string) (*domain.Image, error) {
	return h.reader.FindByPath(path)
}

func (h *HybridImageRepository) FindAll(limit, offset int, sortBy, sortDir string) ([]domain.Image, error) {
	return h.reader.FindAll(limit, offset, sortBy, sortDir)
}

func (h *HybridImageRepository) FindByIDRange(limit int, lastID int64) ([]domain.Image, error) {
	return h.reader.FindByIDRange(limit, lastID)
}

func (h *HybridImageRepository) FindBySourceRootsAfterID(limit int, lastID int64, sourceRoots []string) ([]domain.Image, error) {
	return h.reader.FindBySourceRootsAfterID(limit, lastID, sourceRoots)
}

func (h *HybridImageRepository) Count() (int64, error) {
	return h.reader.Count()
}

// GalleryImageQuery delegates

func (h *HybridImageRepository) FindByTagIDs(ctx context.Context, tagIDs []int64, limit, offset int, sortBy, sortDir string) ([]domain.Image, error) {
	return h.gallery.FindByTagIDs(ctx, tagIDs, limit, offset, sortBy, sortDir)
}

func (h *HybridImageRepository) CountByTagIDs(ctx context.Context, tagIDs []int64) (int64, error) {
	return h.gallery.CountByTagIDs(ctx, tagIDs)
}

func (h *HybridImageRepository) FindUntagged(ctx context.Context, limit, offset int, sortBy, sortDir string) ([]domain.Image, error) {
	return h.gallery.FindUntagged(ctx, limit, offset, sortBy, sortDir)
}

func (h *HybridImageRepository) CountUntagged(ctx context.Context) (int64, error) {
	return h.gallery.CountUntagged(ctx)
}

func (h *HybridImageRepository) FindPendingTags(ctx context.Context, limit, offset int, sortBy, sortDir string) ([]domain.Image, error) {
	return h.gallery.FindPendingTags(ctx, limit, offset, sortBy, sortDir)
}

func (h *HybridImageRepository) CountPendingTags(ctx context.Context) (int64, error) {
	return h.gallery.CountPendingTags(ctx)
}

func (h *HybridImageRepository) FindPendingTagsByTagIDs(ctx context.Context, tagIDs []int64, limit, offset int, sortBy, sortDir string) ([]domain.Image, error) {
	return h.gallery.FindPendingTagsByTagIDs(ctx, tagIDs, limit, offset, sortBy, sortDir)
}

func (h *HybridImageRepository) CountPendingTagsByTagIDs(ctx context.Context, tagIDs []int64) (int64, error) {
	return h.gallery.CountPendingTagsByTagIDs(ctx, tagIDs)
}

func (h *HybridImageRepository) FindByGalleryFilter(ctx context.Context, exactTagIDs, subtreeRootTagIDs []int64, limit, offset int, sortBy, sortDir string) ([]domain.Image, error) {
	return h.gallery.FindByGalleryFilter(ctx, exactTagIDs, subtreeRootTagIDs, limit, offset, sortBy, sortDir)
}

func (h *HybridImageRepository) CountByGalleryFilter(ctx context.Context, exactTagIDs, subtreeRootTagIDs []int64) (int64, error) {
	return h.gallery.CountByGalleryFilter(ctx, exactTagIDs, subtreeRootTagIDs)
}

// BackfillImageQuery delegates

func (h *HybridImageRepository) FindImagesWithoutAITags(ctx context.Context, limit int) ([]domain.Image, error) {
	return h.backfill.FindImagesWithoutAITags(ctx, limit)
}

func (h *HybridImageRepository) FindBackfillCandidates(ctx context.Context, filter BackfillCandidateFilter) ([]domain.Image, error) {
	return h.backfill.FindBackfillCandidates(ctx, filter)
}

func (h *HybridImageRepository) CountBackfillCandidates(ctx context.Context, filter BackfillCandidateFilter) (int64, error) {
	return h.backfill.CountBackfillCandidates(ctx, filter)
}

func (h *HybridImageRepository) CountBackfillSkippedWithAITag(ctx context.Context, filter BackfillCandidateFilter) (int64, error) {
	return h.backfill.CountBackfillSkippedWithAITag(ctx, filter)
}

func (h *HybridImageRepository) CountBackfillSkippedWithActiveTask(ctx context.Context, filter BackfillCandidateFilter) (int64, error) {
	return h.backfill.CountBackfillSkippedWithActiveTask(ctx, filter)
}

func (h *HybridImageRepository) CountBackfillHitCount(ctx context.Context, filter BackfillCandidateFilter) (int64, error) {
	return h.backfill.CountBackfillHitCount(ctx, filter)
}

// ImageMoveQuery delegates

func (h *HybridImageRepository) FindBySourceDirsAndTag(ctx context.Context, sourceDirs []string, tagID int64, limit, offset int) ([]domain.Image, error) {
	return h.move.FindBySourceDirsAndTag(ctx, sourceDirs, tagID, limit, offset)
}

func (h *HybridImageRepository) CountBySourceDirsAndTag(ctx context.Context, sourceDirs []string, tagID int64) (int64, error) {
	return h.move.CountBySourceDirsAndTag(ctx, sourceDirs, tagID)
}

func (h *HybridImageRepository) UpdateImageLocation(ctx context.Context, imageID int64, path, filename, sourceRoot string) error {
	return h.move.UpdateImageLocation(ctx, imageID, path, filename, sourceRoot)
}

// ImageMutationStore delegates

func (h *HybridImageRepository) SaveImage(image *domain.Image) (bool, error) {
	return h.writer.SaveImage(image)
}

func (h *HybridImageRepository) UpdateImagePHashHex(imageID int64, phashHex string) error {
	return h.writer.UpdateImagePHashHex(imageID, phashHex)
}

func (h *HybridImageRepository) UpdateImageDuplicateHashCache(imageID int64, sha256, phashHex string, sourceMTimeUnix int64) error {
	return h.writer.UpdateImageDuplicateHashCache(imageID, sha256, phashHex, sourceMTimeUnix)
}

func (h *HybridImageRepository) UpdateThumbnails(id int64, smallURL, largeURL string) error {
	return h.writer.UpdateThumbnails(id, smallURL, largeURL)
}

func (h *HybridImageRepository) Delete(id int64) error {
	return h.writer.Delete(id)
}