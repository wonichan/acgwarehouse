package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/internal/repository"
)

func mustCreateDailyImages(t *testing.T, repo *repository.ImageRepository, count int) []do.Image {
	t.Helper()
	images := make([]do.Image, 0, count)
	for index := 1; index <= count; index++ {
		image, err := repo.UpsertByCOSKey(context.Background(), do.Image{
			COSKey:       dailyCOSKey(index),
			Filename:     dailyFilename(index),
			Size:         int64(100 + index),
			Width:        640,
			Height:       480,
			LastModified: fixedDailyNow(),
		})
		if err != nil {
			t.Fatalf("create daily image %d: %v", index, err)
		}
		images = append(images, image)
	}
	return images
}

func dailyCOSKey(index int) string {
	return "thumbnails/daily-" + dailyImageSuffix(index) + ".png"
}

func dailyFilename(index int) string {
	return "daily-" + dailyImageSuffix(index) + ".png"
}

func dailyImageSuffix(index int) string {
	if index < 10 {
		return "0" + string(rune('0'+index))
	}
	return string(rune('0'+index/10)) + string(rune('0'+index%10))
}

func imageIDs(images []do.Image) []int64 {
	ids := make([]int64, 0, len(images))
	for _, image := range images {
		ids = append(ids, image.ID)
	}
	return ids
}

func dailyRange(start int64, end int64) []int64 {
	ids := make([]int64, 0, end-start+1)
	for id := start; id <= end; id++ {
		ids = append(ids, id)
	}
	return ids
}

func assertImageIDsEqual(t *testing.T, got []int64, want []int64) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("ids = %#v, want %#v", got, want)
	}
	for index, id := range got {
		if id != want[index] {
			t.Fatalf("ids = %#v, want %#v", got, want)
		}
	}
}

func assertUniqueImageIDs(t *testing.T, images []do.Image) {
	t.Helper()
	seen := make(map[int64]struct{}, len(images))
	for _, image := range images {
		if _, ok := seen[image.ID]; ok {
			t.Fatalf("images = %#v, want unique IDs", images)
		}
		seen[image.ID] = struct{}{}
	}
}

func fixedDailyNow() time.Time {
	return time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
}
