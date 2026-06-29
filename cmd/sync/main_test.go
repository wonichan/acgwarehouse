package main

import (
	"context"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"testing"
)

func Test_shouldSkipObject_returns_true_when_filename_contains_small(t *testing.T) {
	// Given
	key := "thumbnails/gallery/1-(28)-small.jpg"

	// When
	skipped := shouldSkipObject(key)

	// Then
	if !skipped {
		t.Fatal("shouldSkipObject() = false, want true")
	}
}

func Test_shouldSkipObject_returns_false_when_filename_does_not_contain_small(t *testing.T) {
	// Given
	key := "thumbnails/gallery/1-(28)-large.jpg"

	// When
	skipped := shouldSkipObject(key)

	// Then
	if skipped {
		t.Fatal("shouldSkipObject() = true, want false")
	}
}

func Test_decodeRemoteSize_returns_dimensions_from_remote_image_header(t *testing.T) {
	// Given
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Range") != "bytes=0-65535" {
			t.Fatalf("range = %q, want bytes=0-65535", request.Header.Get("Range"))
		}
		imageData := image.NewRGBA(image.Rect(0, 0, 32, 24))
		imageData.Set(0, 0, color.White)
		if err := png.Encode(writer, imageData); err != nil {
			t.Fatalf("encode png: %v", err)
		}
	}))
	t.Cleanup(server.Close)

	// When
	width, height, err := decodeRemoteSize(context.Background(), server.URL)

	// Then
	if err != nil {
		t.Fatalf("decode remote size: %v", err)
	}
	if width != 32 || height != 24 {
		t.Fatalf("dimensions = %dx%d, want 32x24", width, height)
	}
}
