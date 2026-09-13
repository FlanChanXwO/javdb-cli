package javdb

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// 资产契约:MovieAsset 只有 Type("image"/"video")与 URL 两个字段;
// 序列顺序固定 thumbnail → cover(若详情提供)→ preview_images[](large_url 优先)→ preview video;
// 详情中缺失的项直接跳过(input.md 计划 #2/#4/#5)。

func TestMovieAssetsFromDetailOrdersAllAssetKinds(t *testing.T) {
	got := MovieAssetsFromDetail(map[string]any{
		"thumb_url":         "https://media.example.test/thumb.jpg",
		"cover_url":         "https://media.example.test/cover.jpg",
		"preview_video_url": "https://media.example.test/preview.m3u8",
		"preview_images": []any{
			map[string]any{"large_url": "https://media.example.test/p1-large.jpg", "thumb_url": "https://media.example.test/p1-thumb.jpg"},
			map[string]any{"thumb_url": "https://media.example.test/p2-thumb.jpg"},
			map[string]any{"large_url": "https://media.example.test/p3-large.jpg"},
		},
	})
	want := []MovieAsset{
		{Type: "image", URL: "https://media.example.test/thumb.jpg"},
		{Type: "image", URL: "https://media.example.test/cover.jpg"},
		{Type: "image", URL: "https://media.example.test/p1-large.jpg"},
		{Type: "image", URL: "https://media.example.test/p2-thumb.jpg"},
		{Type: "image", URL: "https://media.example.test/p3-large.jpg"},
		{Type: "video", URL: "https://media.example.test/preview.m3u8"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("assets mismatch:\n got  = %+v\n want = %+v", got, want)
	}
}

func TestMovieAssetsFromDetailSkipsMissingItems(t *testing.T) {
	// 无 cover_url、无 thumb_url、无 preview_video_url:只保留唯一一张 preview。
	got := MovieAssetsFromDetail(map[string]any{
		"preview_images": []any{map[string]any{"large_url": "https://media.example.test/only.jpg"}},
	})
	want := []MovieAsset{{Type: "image", URL: "https://media.example.test/only.jpg"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("assets mismatch:\n got  = %+v\n want = %+v", got, want)
	}
}

func TestMovieAssetsFromDetailSkipsPreviewWithoutAnyURL(t *testing.T) {
	// 预览项既无 large_url 也无 thumb_url:整项跳过,不产出空 URL 资产。
	got := MovieAssetsFromDetail(map[string]any{
		"thumb_url": "https://media.example.test/thumb.jpg",
		"preview_images": []any{
			map[string]any{"other": "field"},
			map[string]any{"large_url": "https://media.example.test/p2.jpg"},
		},
	})
	want := []MovieAsset{
		{Type: "image", URL: "https://media.example.test/thumb.jpg"},
		{Type: "image", URL: "https://media.example.test/p2.jpg"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("assets mismatch:\n got  = %+v\n want = %+v", got, want)
	}
}

func TestMovieAssetsFromDetailToleratesMalformedPreviewImages(t *testing.T) {
	// preview_images 非数组、元素非 map、空数组:全部安全跳过。
	for name, detail := range map[string]map[string]any{
		"not an array":   {"preview_images": "oops"},
		"mixed elements": {"preview_images": []any{"string", 42, map[string]any{"large_url": "https://media.example.test/ok.jpg"}}},
		"empty array":    {"preview_images": []any{}},
	} {
		got := MovieAssetsFromDetail(detail)
		if name == "mixed elements" {
			if len(got) != 1 || got[0].URL != "https://media.example.test/ok.jpg" {
				t.Fatalf("%s: got %+v", name, got)
			}
			continue
		}
		if len(got) != 0 {
			t.Fatalf("%s: expected no assets, got %+v", name, got)
		}
	}
}

func TestMovieAssetsFromDetailEmptyDetailReturnsEmptySlice(t *testing.T) {
	got := MovieAssetsFromDetail(map[string]any{})
	if got == nil {
		t.Fatal("expected non-nil empty slice for stable JSON output")
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 assets, got %+v", got)
	}
}

// 描述文本与资产序列同序,仅供 assets list TTY 渲染;跳过项不占 preview 编号。

func TestMovieAssetDescriptionsMatchAssetOrder(t *testing.T) {
	movie := map[string]any{
		"thumb_url":         "https://media.example.test/thumb.jpg",
		"cover_url":         "https://media.example.test/cover.jpg",
		"preview_video_url": "https://media.example.test/preview.m3u8",
		"preview_images": []any{
			map[string]any{"large_url": "https://media.example.test/p1-large.jpg"},
			map[string]any{"thumb_url": "https://media.example.test/p2-thumb.jpg"},
			map[string]any{"large_url": "https://media.example.test/p3-large.jpg"},
		},
	}
	assets := MovieAssetsFromDetail(movie)
	descs := MovieAssetDescriptions(movie)
	wantDescs := []string{"thumbnail", "cover", "preview 1", "preview 2", "preview 3", "preview"}
	if !reflect.DeepEqual(descs, wantDescs) {
		t.Fatalf("descriptions mismatch:\n got  = %q\n want = %q", descs, wantDescs)
	}
	if len(assets) != len(descs) {
		t.Fatalf("assets/descriptions length mismatch: %d vs %d", len(assets), len(descs))
	}
}

func TestMovieAssetDescriptionsSkipMissingItems(t *testing.T) {
	movie := map[string]any{
		"thumb_url": "https://media.example.test/thumb.jpg",
		"preview_images": []any{
			map[string]any{"other": "field"},
			map[string]any{"large_url": "https://media.example.test/p2.jpg"},
		},
	}
	descs := MovieAssetDescriptions(movie)
	want := []string{"thumbnail", "preview 1"}
	if !reflect.DeepEqual(descs, want) {
		t.Fatalf("descriptions mismatch:\n got  = %q\n want = %q", descs, want)
	}
}

func TestClientMovieAssetsReadsDetail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/v4/movies/abc123" {
			http.NotFound(writer, request)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"success":true,"data":{"movie":{"thumb_url":"https://m.example.test/t.jpg","preview_video_url":"https://m.example.test/p.m3u8"}}}`))
	}))
	defer server.Close()

	client, err := New(WithHost(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	assets, err := client.MovieAssets(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("MovieAssets() error = %v", err)
	}
	want := []MovieAsset{
		{Type: "image", URL: "https://m.example.test/t.jpg"},
		{Type: "video", URL: "https://m.example.test/p.m3u8"},
	}
	if !reflect.DeepEqual(assets, want) {
		t.Fatalf("MovieAssets mismatch:\n got  = %+v\n want = %+v", assets, want)
	}
}

func TestClientMovieAssetsPropagatesDetailError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.Error(writer, "boom", http.StatusInternalServerError)
	}))
	defer server.Close()

	client, err := New(WithHost(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.MovieAssets(context.Background(), "abc123"); err == nil {
		t.Fatal("expected detail error to propagate")
	}
}

// DownloadMovieAsset:image 走验证链落盘;video 输出格式由 target 后缀决定,
// .ts 保留 MPEG-TS,其余后缀在 remux 层落地前一律拒绝。

var testJPEGPayload = []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00, 0x01}

func TestClientDownloadMovieAssetImageWritesValidatedFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write(testJPEGPayload)
	}))
	defer server.Close()

	client, err := New(WithHost(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	target := t.TempDir() + "/out.jpg"
	written, err := client.DownloadMovieAsset(context.Background(), MovieAsset{Type: "image", URL: server.URL + "/img.bin"}, target)
	if err != nil {
		t.Fatalf("DownloadMovieAsset() error = %v", err)
	}
	if written != int64(len(testJPEGPayload)) {
		t.Fatalf("written = %d, want %d", written, len(testJPEGPayload))
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, testJPEGPayload) {
		t.Fatalf("target bytes = %x, want %x", got, testJPEGPayload)
	}
}

func TestClientDownloadMovieAssetVideoTSDownloadsHLS(t *testing.T) {
	segment := []byte{0x47, 0x40, 0x00, 0x10, 0x00}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/video.m3u8":
			_, _ = writer.Write([]byte("#EXTM3U\n#EXT-X-TARGETDURATION:2\n#EXTINF:1.0,\nseg1.ts\n#EXT-X-ENDLIST\n"))
		case "/seg1.ts":
			_, _ = writer.Write(segment)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client, err := New(WithHost(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	target := t.TempDir() + "/out.ts"
	written, err := client.DownloadMovieAsset(context.Background(), MovieAsset{Type: "video", URL: server.URL + "/video.m3u8"}, target)
	if err != nil {
		t.Fatalf("DownloadMovieAsset() error = %v", err)
	}
	if written != int64(len(segment)) {
		t.Fatalf("written = %d, want %d", written, len(segment))
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, segment) {
		t.Fatalf("target bytes = %x, want %x", got, segment)
	}
}

func TestClientDownloadMovieAssetVideoRejectsUnsupportedFormats(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		t.Fatal("no media request expected for unsupported format")
	}))
	defer server.Close()

	client, err := New(WithHost(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for _, target := range []string{dir + "/preview.mkv", dir + "/preview.mp4", dir + "/novalue"} {
		_, err := client.DownloadMovieAsset(context.Background(), MovieAsset{Type: "video", URL: server.URL + "/video.m3u8"}, target)
		if err == nil {
			t.Fatalf("target %q: expected unsupported format error", target)
		}
		want := fmt.Sprintf("unsupported video output format %q", filepath.Ext(target))
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("target %q: error %q, want contains %q", target, err.Error(), want)
		}
		if _, statErr := os.Stat(target); !os.IsNotExist(statErr) {
			t.Fatalf("target %q: expected no output file", target)
		}
	}
}

func TestClientDownloadMovieAssetRejectsUnknownAssetType(t *testing.T) {
	client, err := New(WithHost("https://unused.example.test"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.DownloadMovieAsset(context.Background(), MovieAsset{Type: "audio", URL: "https://x.example.test/a"}, t.TempDir()+"/a.bin")
	if err == nil || !strings.Contains(err.Error(), `unsupported asset type "audio"`) {
		t.Fatalf("error = %v, want unsupported asset type", err)
	}
}

// ImageAssetFormat 读取已验证的本地图片文件头,返回下载层默认命名用的格式名。

func TestImageAssetFormatReadsLocalFile(t *testing.T) {
	dir := t.TempDir()
	jpg := filepath.Join(dir, "a.bin")
	if err := os.WriteFile(jpg, testJPEGPayload, 0o644); err != nil {
		t.Fatal(err)
	}
	format, ok := ImageAssetFormat(jpg)
	if !ok || format != "jpg" {
		t.Fatalf("format = (%q, %v), want jpg", format, ok)
	}
	notImage := filepath.Join(dir, "b.bin")
	if err := os.WriteFile(notImage, []byte("<html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := ImageAssetFormat(notImage); ok {
		t.Fatal("html file must not be detected as image")
	}
	if _, ok := ImageAssetFormat(filepath.Join(dir, "missing.bin")); ok {
		t.Fatal("missing file must not be detected as image")
	}
}
