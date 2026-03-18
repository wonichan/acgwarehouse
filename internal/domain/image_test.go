package domain

import "testing"

func TestImageFields(t *testing.T) {
	img := Image{}

	if img.ThumbnailSmallUrl != "" {
		t.Fatalf("ThumbnailSmallUrl default = %q, want empty", img.ThumbnailSmallUrl)
	}
	if img.ThumbnailLargeUrl != "" {
		t.Fatalf("ThumbnailLargeUrl default = %q, want empty", img.ThumbnailLargeUrl)
	}
}
