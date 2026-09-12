package javdb

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

func TestDownloadMovieAssetsWritesSelectedAssets(t *testing.T) {
	thumbnail := []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x01}
	previewImage := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0x00}
	previewVideo := []byte("first preview segment\nsecond preview segment\n")
	var detailCalls int
	var secondPreviewCalls int
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writeJSON := func(value any) {
			writer.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(writer).Encode(value)
		}
		switch request.URL.Path {
		case "/api/v4/movies/movie-id":
			detailCalls++
			writeJSON(map[string]any{"success": true, "data": map[string]any{
				"movie": map[string]any{
					"id":                "movie-id",
					"thumb_url":         server.URL + "/media/thumbnail.jpg",
					"preview_images":    []map[string]any{{"large_url": server.URL + "/media/preview-first.png"}, {"large_url": server.URL + "/media/preview-second.png"}},
					"preview_video_url": server.URL + "/media/preview/index.m3u8",
				},
			}})
		case "/media/thumbnail.jpg":
			_, _ = writer.Write(thumbnail)
		case "/media/preview-first.png":
			_, _ = writer.Write(previewImage)
		case "/media/preview-second.png":
			secondPreviewCalls++
			_, _ = writer.Write([]byte("must not be requested"))
		case "/media/preview/index.m3u8":
			_, _ = writer.Write([]byte("#EXTM3U\n#EXTINF:1.0,\npart-1.ts\n#EXTINF:1.0,\npart-2.ts\n#EXT-X-ENDLIST\n"))
		case "/media/preview/part-1.ts":
			_, _ = writer.Write(previewVideo[:len("first preview segment\n")])
		case "/media/preview/part-2.ts":
			_, _ = writer.Write(previewVideo[len("first preview segment\n"):])
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client, err := New(WithHost(server.URL))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	dir := t.TempDir()
	paths := MovieAssetDownloadOptions{
		ThumbnailPath:    filepath.Join(dir, "thumbnail.jpg"),
		PreviewImagePath: filepath.Join(dir, "preview.png"),
		PreviewVideoPath: filepath.Join(dir, "preview.ts"),
	}
	result, err := client.DownloadMovieAssets(context.Background(), "movie-id", paths)
	if err != nil {
		t.Fatalf("DownloadMovieAssets: %v", err)
	}
	if detailCalls != 1 {
		t.Fatalf("movie detail calls = %d, want 1", detailCalls)
	}
	if secondPreviewCalls != 0 {
		t.Fatalf("later preview image calls = %d, want 0", secondPreviewCalls)
	}

	for _, asset := range []struct {
		name string
		path string
		want []byte
		got  int64
	}{
		{name: "thumbnail", path: paths.ThumbnailPath, want: thumbnail, got: result.ThumbnailBytes},
		{name: "preview image", path: paths.PreviewImagePath, want: previewImage, got: result.PreviewImageBytes},
		{name: "preview video", path: paths.PreviewVideoPath, want: previewVideo, got: result.PreviewVideoBytes},
	} {
		if got, want := asset.got, int64(len(asset.want)); got != want {
			t.Errorf("%s bytes = %d, want %d", asset.name, got, want)
		}
		if got, err := os.ReadFile(asset.path); err != nil {
			t.Errorf("read %s: %v", asset.name, err)
		} else if !bytes.Equal(got, asset.want) {
			t.Errorf("%s content = %x, want %x", asset.name, got, asset.want)
		}
	}
	if result.ThumbnailPath != paths.ThumbnailPath || result.PreviewImagePath != paths.PreviewImagePath || result.PreviewVideoPath != paths.PreviewVideoPath {
		t.Fatalf("result paths = %+v, want %+v", result, paths)
	}
}
