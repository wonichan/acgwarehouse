package service

import (
	"testing"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

func TestResolveAITagImagePathBuildsAbsoluteURLFromRelativeThumbnailPath(t *testing.T) {
	t.Parallel()

	image := &domain.Image{
		Path:              "/images/original.png",
		ThumbnailLargeUrl: "acg/thumbnails/20260419/example-large.jpg",
	}

	got := ResolveAITagImagePath(image, "http://118.25.139.30:19003")
	want := "http://118.25.139.30:19003/acg/thumbnails/20260419/example-large.jpg"
	if got != want {
		t.Fatalf("ResolveAITagImagePath() = %q, want %q", got, want)
	}
}

func TestResolveAITagImagePathKeepsExistingAbsoluteThumbnailURL(t *testing.T) {
	t.Parallel()

	image := &domain.Image{
		Path:              "/images/original.png",
		ThumbnailLargeUrl: "https://cdn.example.com/acg/thumbnails/example-large.jpg",
	}

	got := ResolveAITagImagePath(image, "http://118.25.139.30:19003")
	if got != image.ThumbnailLargeUrl {
		t.Fatalf("ResolveAITagImagePath() = %q, want %q", got, image.ThumbnailLargeUrl)
	}
}

func TestResolveAITagImagePathFallsBackToOriginalPathWithoutThumbnail(t *testing.T) {
	t.Parallel()

	image := &domain.Image{Path: "/images/original.png"}

	got := ResolveAITagImagePath(image, "http://118.25.139.30:19003")
	if got != image.Path {
		t.Fatalf("ResolveAITagImagePath() = %q, want %q", got, image.Path)
	}
}

func TestResolveAITagImagePathBuildsCOSURLFromRelativeThumbnailPath(t *testing.T) {
	t.Parallel()

	image := &domain.Image{
		Path:              "/images/original.png",
		ThumbnailLargeUrl: "thumbnails/example-large.jpg",
	}

	got := ResolveAITagImagePath(image, "https://acg-1250000000.cos.ap-shanghai.myqcloud.com")
	want := "https://acg-1250000000.cos.ap-shanghai.myqcloud.com/thumbnails/example-large.jpg"
	if got != want {
		t.Fatalf("ResolveAITagImagePath() = %q, want %q", got, want)
	}
}

func TestResolveAITagImagePathKeepsRelativeThumbnailWhenBaseURLIsEmpty(t *testing.T) {
	t.Parallel()

	image := &domain.Image{
		ThumbnailLargeUrl: "acg/thumbnails/20260419/example-large.jpg",
	}

	got := ResolveAITagImagePath(image, "")
	if got != image.ThumbnailLargeUrl {
		t.Fatalf("ResolveAITagImagePath() = %q, want %q", got, image.ThumbnailLargeUrl)
	}
}

func TestResolveAITagImagePathKeepsRelativeThumbnailWhenBaseURLIsInvalid(t *testing.T) {
	t.Parallel()

	image := &domain.Image{
		ThumbnailLargeUrl: "acg/thumbnails/20260419/example-large.jpg",
	}

	got := ResolveAITagImagePath(image, "not-a-url")
	if got != image.ThumbnailLargeUrl {
		t.Fatalf("ResolveAITagImagePath() = %q, want %q", got, image.ThumbnailLargeUrl)
	}
}
