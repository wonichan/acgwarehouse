package worker

import (
	"context"
	"testing"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

func TestThumbnailHandlerUsesDistinctUploadKeysPerImage(t *testing.T) {
	t.Parallel()

	uploader := &stubThumbnailUploader{}
	thumbSvc := &stubThumbnailGenerator{
		small: &domain.Thumbnail{Data: []byte("small-bytes"), Size: "small"},
		large: &domain.Thumbnail{Data: []byte("large-bytes"), Size: "large"},
	}
	repo1 := &stubThumbnailImageRepo{}
	repo2 := &stubThumbnailImageRepo{}
	h := NewThumbnailHandler(thumbSvc, uploader, repo1)

	if err := h.Handle(context.Background(), 1, `{"image_id":11,"path":"C:/tmp/set-a/cover.jpg","filename":"cover"}`); err != nil {
		t.Fatalf("first Handle() error = %v", err)
	}
	h.imageRepo = repo2
	if err := h.Handle(context.Background(), 2, `{"image_id":22,"path":"C:/tmp/set-b/cover.png","filename":"cover"}`); err != nil {
		t.Fatalf("second Handle() error = %v", err)
	}

	if repo1.smallURL == repo2.smallURL {
		t.Fatalf("small thumbnail URLs should differ for distinct images, got same value %q", repo1.smallURL)
	}
	if repo1.largeURL == repo2.largeURL {
		t.Fatalf("large thumbnail URLs should differ for distinct images, got same value %q", repo1.largeURL)
	}
}
