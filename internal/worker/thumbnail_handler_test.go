package worker

import (
	"context"
	"errors"
	"testing"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

func TestThumbnailHandlerHandleSuccess(t *testing.T) {
	t.Parallel()

	thumbSvc := &stubThumbnailGenerator{
		small: &domain.Thumbnail{Data: []byte("small-bytes"), Size: "small"},
		large: &domain.Thumbnail{Data: []byte("large-bytes"), Size: "large"},
	}
	cosSvc := &stubThumbnailUploader{urls: map[string]string{
		"small": "https://cos.local/thumbnails/11_small.jpg",
		"large": "https://cos.local/thumbnails/11_large.jpg",
	}}
	repo := &stubThumbnailImageRepo{}

	h := NewThumbnailHandler(thumbSvc, cosSvc, repo)
	err := h.Handle(context.Background(), 99, `{"image_id":11,"path":"C:/tmp/a.png"}`)
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	if repo.updateCalls != 1 {
		t.Fatalf("UpdateThumbnails() calls = %d, want 1", repo.updateCalls)
	}
	if repo.id != 11 {
		t.Fatalf("updated image id = %d, want 11", repo.id)
	}
	if repo.smallURL != "https://cos.local/thumbnails/11_small.jpg" {
		t.Fatalf("small url = %q", repo.smallURL)
	}
	if repo.largeURL != "https://cos.local/thumbnails/11_large.jpg" {
		t.Fatalf("large url = %q", repo.largeURL)
	}
}

func TestThumbnailHandlerHandleInvalidPayload(t *testing.T) {
	t.Parallel()

	h := NewThumbnailHandler(&stubThumbnailGenerator{}, &stubThumbnailUploader{}, &stubThumbnailImageRepo{})
	if err := h.Handle(context.Background(), 1, "bad-json"); err == nil {
		t.Fatal("Handle() expected error for invalid payload")
	}
}

func TestThumbnailHandlerHandleGenerateError(t *testing.T) {
	t.Parallel()

	thumbSvc := &stubThumbnailGenerator{err: errors.New("generate failed")}
	cosSvc := &stubThumbnailUploader{}
	repo := &stubThumbnailImageRepo{}

	h := NewThumbnailHandler(thumbSvc, cosSvc, repo)
	err := h.Handle(context.Background(), 1, `{"image_id":11,"path":"C:/tmp/a.png"}`)
	if err == nil {
		t.Fatal("Handle() expected error when thumbnail generation fails")
	}
	if repo.updateCalls != 0 {
		t.Fatalf("UpdateThumbnails() calls = %d, want 0", repo.updateCalls)
	}
}

type stubThumbnailGenerator struct {
	small *domain.Thumbnail
	large *domain.Thumbnail
	err   error
}

func (s *stubThumbnailGenerator) GenerateBoth(path string) (small, large *domain.Thumbnail, err error) {
	if s.err != nil {
		return nil, nil, s.err
	}
	if s.small == nil {
		s.small = &domain.Thumbnail{Data: []byte("small"), Size: "small"}
	}
	if s.large == nil {
		s.large = &domain.Thumbnail{Data: []byte("large"), Size: "large"}
	}
	return s.small, s.large, nil
}

type stubThumbnailUploader struct {
	urls map[string]string
	err  error
}

func (s *stubThumbnailUploader) Upload(ctx context.Context, imageID int64, size string, data []byte) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	if s.urls != nil {
		if url, ok := s.urls[size]; ok {
			return url, nil
		}
	}
	return "https://cos.local/" + size, nil
}

type stubThumbnailImageRepo struct {
	updateCalls int
	id          int64
	smallURL    string
	largeURL    string
}

func (s *stubThumbnailImageRepo) UpdateThumbnails(id int64, smallURL, largeURL string) error {
	s.updateCalls++
	s.id = id
	s.smallURL = smallURL
	s.largeURL = largeURL
	return nil
}
