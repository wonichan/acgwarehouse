package service_test

import (
	"context"
	stderrors "errors"
	"testing"
	"time"

	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/internal/service"
)

type memoryTagRepository struct {
	tagsByID map[int64]do.Tag
	byImage  map[int64][]do.Tag
	nextID   int64
}

func newMemoryTagRepository() *memoryTagRepository {
	return &memoryTagRepository{
		tagsByID: make(map[int64]do.Tag),
		byImage:  make(map[int64][]do.Tag),
		nextID:   1,
	}
}

func (r *memoryTagRepository) Create(_ context.Context, tag do.Tag) (do.Tag, error) {
	for _, existing := range r.tagsByID {
		if existing.Name == tag.Name {
			return existing, nil
		}
	}
	tag.ID = r.nextID
	r.nextID++
	r.tagsByID[tag.ID] = tag
	return tag, nil
}

func (r *memoryTagRepository) List(_ context.Context) ([]do.Tag, error) {
	result := make([]do.Tag, 0, len(r.tagsByID))
	for _, tag := range r.tagsByID {
		result = append(result, tag)
	}
	return result, nil
}

func (r *memoryTagRepository) FindByID(_ context.Context, id int64) (do.Tag, error) {
	tag, ok := r.tagsByID[id]
	if !ok {
		return do.Tag{}, service.ErrTagNotFound
	}
	return tag, nil
}

func (r *memoryTagRepository) Update(_ context.Context, tag do.Tag) (do.Tag, error) {
	stored, ok := r.tagsByID[tag.ID]
	if !ok {
		return do.Tag{}, service.ErrTagNotFound
	}
	stored.Name = tag.Name
	r.tagsByID[tag.ID] = stored
	return stored, nil
}

func (r *memoryTagRepository) Delete(_ context.Context, id int64) error {
	if _, ok := r.tagsByID[id]; !ok {
		return service.ErrTagNotFound
	}
	delete(r.tagsByID, id)
	return nil
}

func (r *memoryTagRepository) Suggest(_ context.Context, _ string, _ int) ([]do.Tag, error) {
	return r.List(context.Background())
}

func (r *memoryTagRepository) ListByImageID(_ context.Context, imageID int64) ([]do.Tag, error) {
	return append([]do.Tag{}, r.byImage[imageID]...), nil
}

func (r *memoryTagRepository) AssignToImages(_ context.Context, imageIDs []int64, tagIDs []int64) ([]do.Image, error) {
	for _, imageID := range imageIDs {
		for _, tagID := range tagIDs {
			tag := r.tagsByID[tagID]
			r.byImage[imageID] = append(r.byImage[imageID], tag)
		}
	}
	return r.imagesFromIDs(imageIDs), nil
}

func (r *memoryTagRepository) UnassignFromImages(_ context.Context, imageIDs []int64, _ []int64) ([]do.Image, error) {
	for _, imageID := range imageIDs {
		delete(r.byImage, imageID)
	}
	return r.imagesFromIDs(imageIDs), nil
}

func (r *memoryTagRepository) imagesFromIDs(ids []int64) []do.Image {
	images := make([]do.Image, 0, len(ids))
	for _, id := range ids {
		tags := make([]string, 0, len(r.byImage[id]))
		for _, tag := range r.byImage[id] {
			tags = append(tags, tag.Name)
		}
		images = append(images, do.Image{
			ID:        id,
			COSKey:    "thumbnails/test.png",
			Filename:  "test.png",
			Status:    do.ImageStatusActive,
			CreatedAt: time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC),
			Tags:      tags,
		})
	}
	return images
}

type recordingImageIndexer struct {
	images []do.Image
}

func (i *recordingImageIndexer) Index(_ context.Context, image do.Image) error {
	i.images = append(i.images, image)
	return nil
}

func Test_TagService_Create_returns_global_tag_when_input_valid(t *testing.T) {
	// Given
	repo := newMemoryTagRepository()
	svc := service.NewTagService(repo, nil)

	// When
	first, err := svc.Create(context.Background(), do.Tag{Name: " miku "})
	second, secondErr := svc.Create(context.Background(), do.Tag{Name: "miku"})

	// Then
	if err != nil || secondErr != nil {
		t.Fatalf("create tags: %v %v", err, secondErr)
	}
	if first.ID != second.ID || first.Name != "miku" {
		t.Fatalf("first=%#v second=%#v, want shared normalized tag", first, second)
	}
}

func Test_TagService_Update_and_delete_require_admin_role(t *testing.T) {
	// Given
	repo := newMemoryTagRepository()
	svc := service.NewTagService(repo, nil)
	tag, err := svc.Create(context.Background(), do.Tag{Name: "miku"})
	if err != nil {
		t.Fatalf("create tag: %v", err)
	}

	// When
	_, updateErr := svc.Update(context.Background(), do.UserRoleUser, do.Tag{ID: tag.ID, Name: "rin"})
	deleteErr := svc.Delete(context.Background(), do.UserRoleUser, tag.ID)

	// Then
	if !stderrors.Is(updateErr, service.ErrForbidden) {
		t.Fatalf("update error = %v, want forbidden", updateErr)
	}
	if !stderrors.Is(deleteErr, service.ErrForbidden) {
		t.Fatalf("delete error = %v, want forbidden", deleteErr)
	}
}

func Test_TagService_AssignToImages_indexes_affected_images_with_tags(t *testing.T) {
	// Given
	repo := newMemoryTagRepository()
	indexer := &recordingImageIndexer{}
	svc := service.NewTagService(repo, indexer)
	tag, err := svc.Create(context.Background(), do.Tag{Name: "初音"})
	if err != nil {
		t.Fatalf("create tag: %v", err)
	}

	// When
	images, err := svc.AssignToImages(context.Background(), []int64{7}, []int64{tag.ID})

	// Then
	if err != nil {
		t.Fatalf("assign tags: %v", err)
	}
	if len(images) != 1 || len(indexer.images) != 1 || indexer.images[0].Tags[0] != "初音" {
		t.Fatalf("images=%#v indexed=%#v, want affected image indexed with tag", images, indexer.images)
	}
}

func Test_TagService_Suggest_uses_default_limit_when_limit_missing(t *testing.T) {
	// Given
	repo := newMemoryTagRepository()
	svc := service.NewTagService(repo, nil)
	_, _ = svc.Create(context.Background(), do.Tag{Name: "miku"})

	// When
	tags, err := svc.Suggest(context.Background(), "mi", 0)

	// Then
	if err != nil {
		t.Fatalf("suggest tags: %v", err)
	}
	if len(tags) != 1 || tags[0].Name != "miku" {
		t.Fatalf("tags = %#v, want miku", tags)
	}
}
