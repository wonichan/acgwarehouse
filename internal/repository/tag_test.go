package repository_test

import (
	"context"
	stderrors "errors"
	"testing"

	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/internal/repository"
)

func Test_TagRepository_Create_reuses_existing_tag_when_name_repeated(t *testing.T) {
	// Given
	database := openTestDatabase(t)
	imageRepo := repository.NewImageRepository(database.Read, database.Write)
	repo := repository.NewTagRepository(database.Read, database.Write, imageRepo)

	// When
	created, err := repo.Create(context.Background(), do.Tag{Name: "  miku  "})
	repeated, repeatErr := repo.Create(context.Background(), do.Tag{Name: "miku"})

	// Then
	if err != nil {
		t.Fatalf("create tag: %v", err)
	}
	if repeatErr != nil {
		t.Fatalf("repeat tag: %v", repeatErr)
	}
	if created.ID == 0 || repeated.ID != created.ID || repeated.Name != "miku" {
		t.Fatalf("created=%#v repeated=%#v, want same normalized tag", created, repeated)
	}
}

func Test_TagRepository_AssignToImages_creates_links_and_increments_usage_once(t *testing.T) {
	// Given
	database := openTestDatabase(t)
	imageRepo := repository.NewImageRepository(database.Read, database.Write)
	tagRepo := repository.NewTagRepository(database.Read, database.Write, imageRepo)
	first := mustCreateImage(t, imageRepo, "thumbnails/one.png")
	second := mustCreateImage(t, imageRepo, "thumbnails/two.png")
	tag := mustCreateTag(t, tagRepo, "初音")

	// When
	images, err := tagRepo.AssignToImages(context.Background(), []int64{first.ID, second.ID}, []int64{tag.ID})
	imagesAgain, repeatErr := tagRepo.AssignToImages(context.Background(), []int64{first.ID}, []int64{tag.ID})
	stored, findErr := tagRepo.FindByID(context.Background(), tag.ID)

	// Then
	if err != nil {
		t.Fatalf("assign tags: %v", err)
	}
	if repeatErr != nil {
		t.Fatalf("repeat assign tags: %v", repeatErr)
	}
	if findErr != nil {
		t.Fatalf("find tag: %v", findErr)
	}
	if stored.UsageCount != 2 {
		t.Fatalf("usage_count = %d, want 2", stored.UsageCount)
	}
	if len(images) != 2 || len(imagesAgain) != 1 {
		t.Fatalf("images=%#v imagesAgain=%#v, want affected images", images, imagesAgain)
	}
}

func Test_TagRepository_UnassignFromImages_removes_links_and_decrements_usage(t *testing.T) {
	// Given
	database := openTestDatabase(t)
	imageRepo := repository.NewImageRepository(database.Read, database.Write)
	tagRepo := repository.NewTagRepository(database.Read, database.Write, imageRepo)
	first := mustCreateImage(t, imageRepo, "thumbnails/one.png")
	second := mustCreateImage(t, imageRepo, "thumbnails/two.png")
	tag := mustCreateTag(t, tagRepo, "壁纸")
	if _, err := tagRepo.AssignToImages(context.Background(), []int64{first.ID, second.ID}, []int64{tag.ID}); err != nil {
		t.Fatalf("assign tags: %v", err)
	}

	// When
	images, err := tagRepo.UnassignFromImages(context.Background(), []int64{first.ID}, []int64{tag.ID})
	stored, findErr := tagRepo.FindByID(context.Background(), tag.ID)
	firstTags, listErr := tagRepo.ListByImageID(context.Background(), first.ID)

	// Then
	if err != nil {
		t.Fatalf("unassign tags: %v", err)
	}
	if findErr != nil {
		t.Fatalf("find tag: %v", findErr)
	}
	if listErr != nil {
		t.Fatalf("list first tags: %v", listErr)
	}
	if stored.UsageCount != 1 || len(firstTags) != 0 || len(images) != 1 || images[0].ID != first.ID {
		t.Fatalf("tag=%#v firstTags=%#v images=%#v, want one usage on second only", stored, firstTags, images)
	}
}

func Test_TagRepository_Suggest_orders_by_prefix_and_usage_count(t *testing.T) {
	// Given
	database := openTestDatabase(t)
	imageRepo := repository.NewImageRepository(database.Read, database.Write)
	tagRepo := repository.NewTagRepository(database.Read, database.Write, imageRepo)
	first := mustCreateImage(t, imageRepo, "thumbnails/one.png")
	second := mustCreateImage(t, imageRepo, "thumbnails/two.png")
	miku := mustCreateTag(t, tagRepo, "miku")
	mikuBlue := mustCreateTag(t, tagRepo, "miku-blue")
	luka := mustCreateTag(t, tagRepo, "luka")
	if _, err := tagRepo.AssignToImages(context.Background(), []int64{first.ID, second.ID}, []int64{mikuBlue.ID}); err != nil {
		t.Fatalf("assign miku-blue: %v", err)
	}
	if _, err := tagRepo.AssignToImages(context.Background(), []int64{first.ID}, []int64{miku.ID}); err != nil {
		t.Fatalf("assign miku: %v", err)
	}
	if _, err := tagRepo.AssignToImages(context.Background(), []int64{first.ID}, []int64{luka.ID}); err != nil {
		t.Fatalf("assign luka: %v", err)
	}

	// When
	tags, err := tagRepo.Suggest(context.Background(), "miku", 10)

	// Then
	if err != nil {
		t.Fatalf("suggest tags: %v", err)
	}
	if len(tags) != 2 || tags[0].Name != "miku-blue" || tags[1].Name != "miku" {
		t.Fatalf("tags = %#v, want miku-blue before miku by usage_count", tags)
	}
}

func Test_TagRepository_Update_and_delete_return_not_found_when_missing(t *testing.T) {
	// Given
	database := openTestDatabase(t)
	imageRepo := repository.NewImageRepository(database.Read, database.Write)
	repo := repository.NewTagRepository(database.Read, database.Write, imageRepo)

	// When
	_, updateErr := repo.Update(context.Background(), do.Tag{ID: 404, Name: "missing"})
	deleteErr := repo.Delete(context.Background(), 404)

	// Then
	if !stderrors.Is(updateErr, repository.ErrTagNotFound) {
		t.Fatalf("update error = %v, want tag not found", updateErr)
	}
	if !stderrors.Is(deleteErr, repository.ErrTagNotFound) {
		t.Fatalf("delete error = %v, want tag not found", deleteErr)
	}
}

func Test_TagRepository_Delete_removes_image_tag_links(t *testing.T) {
	// Given
	database := openTestDatabase(t)
	imageRepo := repository.NewImageRepository(database.Read, database.Write)
	tagRepo := repository.NewTagRepository(database.Read, database.Write, imageRepo)
	image := mustCreateImage(t, imageRepo, "thumbnails/tagged.png")
	tag := mustCreateTag(t, tagRepo, "待删除")
	if _, err := tagRepo.AssignToImages(context.Background(), []int64{image.ID}, []int64{tag.ID}); err != nil {
		t.Fatalf("assign tag: %v", err)
	}

	// When
	err := tagRepo.Delete(context.Background(), tag.ID)
	imageTags, listErr := tagRepo.ListByImageID(context.Background(), image.ID)

	// Then
	if err != nil {
		t.Fatalf("delete tag: %v", err)
	}
	if listErr != nil {
		t.Fatalf("list image tags: %v", listErr)
	}
	if len(imageTags) != 0 {
		t.Fatalf("imageTags = %#v, want empty after tag delete", imageTags)
	}
}

func mustCreateImage(t *testing.T, repo *repository.ImageRepository, key string) do.Image {
	t.Helper()
	image, err := repo.UpsertByCOSKey(context.Background(), do.Image{
		COSKey:       key,
		Filename:     key,
		LastModified: fixedImageTime(),
	})
	if err != nil {
		t.Fatalf("create image %s: %v", key, err)
	}
	return image
}

func mustCreateTag(t *testing.T, repo *repository.TagRepository, name string) do.Tag {
	t.Helper()
	tag, err := repo.Create(context.Background(), do.Tag{Name: name})
	if err != nil {
		t.Fatalf("create tag %s: %v", name, err)
	}
	return tag
}
