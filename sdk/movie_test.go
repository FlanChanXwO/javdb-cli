package javdb

import (
	"strings"
	"testing"
)

func TestMovieAssetURLsUsesOnlyFirstPreviewImage(t *testing.T) {
	sources := movieAssetURLs(map[string]any{
		"thumb_url":         "https://media.example.test/thumb.jpg",
		"preview_video_url": "https://media.example.test/preview.m3u8",
		"preview_images": []any{
			map[string]any{"large_url": "https://media.example.test/first-large.jpg", "thumb_url": "https://media.example.test/first-thumb.jpg"},
			map[string]any{"large_url": "https://media.example.test/second-large.jpg"},
		},
	})
	if sources.thumbnail != "https://media.example.test/thumb.jpg" {
		t.Fatalf("thumbnail = %q", sources.thumbnail)
	}
	if sources.previewImage != "https://media.example.test/first-large.jpg" {
		t.Fatalf("preview image = %q", sources.previewImage)
	}
	if strings.Contains(sources.previewImage, "second") {
		t.Fatalf("preview image must not select later preview: %q", sources.previewImage)
	}
	if sources.previewVideo != "https://media.example.test/preview.m3u8" {
		t.Fatalf("preview video = %q", sources.previewVideo)
	}
}

func TestDistinctMovieAssetPaths(t *testing.T) {
	err := distinctMovieAssetPaths(MovieAssetDownloadOptions{
		ThumbnailPath:    "same.jpg",
		PreviewImagePath: "same.jpg",
	})
	if err == nil {
		t.Fatal("duplicate asset paths unexpectedly accepted")
	}
}
