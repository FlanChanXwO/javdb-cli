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
	segment := validTSSegmentFixture()
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

// validTSSegmentFixture 构造最小合法 MPEG-TS:通过 Layer A 结构校验与
// Layer B 媒体校验(H.264 SPS/PPS+IDR/非 IDR 两帧带 PTS、AAC ADTS 一帧)。
func validTSSegmentFixture() []byte {
	tsPacket := func(pid uint16, payload []byte) []byte {
		p := make([]byte, 188)
		p[0] = 0x47
		p[1] = 0x40 | byte(pid>>8)
		p[2] = byte(pid)
		p[3] = 0x10
		copy(p[4:], payload)
		for i := 4 + len(payload); i < 188; i++ {
			p[i] = 0xFF
		}
		return p
	}
	section := func(tableID byte, body []byte) []byte {
		length := len(body) + 4
		out := []byte{0x00, tableID, byte(length>>8)&0x0F | 0x30, byte(length)}
		out = append(out, body...)
		return append(out, 0, 0, 0, 0)
	}
	pes := func(streamID byte, flags, headerLen byte, body []byte) []byte {
		p := []byte{0x00, 0x00, 0x01, streamID, 0x00, 0x00, 0x80, flags, headerLen}
		p = append(p, body...)
		pesLen := len(p) - 6
		p[4] = byte(pesLen >> 8)
		p[5] = byte(pesLen)
		return p
	}
	pat := section(0x00, []byte{0x00, 0x01, 0xC1, 0x00, 0x00, 0x00, 0x01, 0xF0, 0x00})
	pmt := section(0x02, []byte{
		0x00, 0x01, 0xC1, 0x00, 0x00, 0xE1, 0x01, 0xF0, 0x00,
		0x1B, 0xE1, 0x01, 0xF0, 0x00, 0x0F, 0xE1, 0x02, 0xF0, 0x00,
	})
	start := []byte{0x00, 0x00, 0x00, 0x01}
	sps := append(append([]byte{}, start...), 0x67, 0xAA, 0xBB)
	pps := append(append([]byte{}, start...), 0x68, 0xCC, 0xDD)
	idr := append(append([]byte{}, start...), 0x65, 0x01, 0x02)
	nonIDR := append(append([]byte{}, start...), 0x41, 0x03, 0x04)
	adts := append([]byte{0xFF, 0xF1, 0x51, 0x00, 0x01, 0x40, 0x00}, 0x21, 0x10, 0x30)
	frame0 := append(append(append([]byte{}, sps...), pps...), idr...)
	video0 := pes(0xE0, 0xC0, 10, append([]byte{0x31, 0x00, 0x05, 0xBF, 0x21, 0x11, 0x00, 0x05, 0xBF, 0x21}, frame0...))
	video1 := pes(0xE0, 0xC0, 10, append([]byte{0x31, 0x00, 0x05, 0xDB, 0x41, 0x11, 0x00, 0x05, 0xDB, 0x41}, nonIDR...))
	audio := pes(0xC0, 0x80, 5, append([]byte{0x21, 0x00, 0x05, 0xBF, 0x21}, adts...))
	var data []byte
	data = append(data, tsPacket(0x0000, pat)...)
	data = append(data, tsPacket(0x1000, pmt)...)
	data = append(data, tsPacket(0x0101, video0)...)
	data = append(data, tsPacket(0x0101, video1)...)
	data = append(data, tsPacket(0x0102, audio)...)
	return data
}
