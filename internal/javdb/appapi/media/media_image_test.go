package media

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 图片资产契约(input.md 计划 #19/#20/#41):
// 下载 → 必要时 XOR 解包 → 图片魔数校验 → 原子发布;
// HTML/JSON/403 页/空响应/坏 XOR/未知二进制一律失败且不落盘。

var testJPEG = []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00, 0x01}

func xorWrapped(key byte, payload []byte) []byte {
	wrapped := make([]byte, len(payload)+1)
	wrapped[0] = key
	for i, b := range payload {
		wrapped[i+1] = b ^ key
	}
	return wrapped
}

func TestDownloadImageAcceptsPlainAndXORWrappedImages(t *testing.T) {
	cases := []struct {
		name    string
		payload []byte
	}{
		{"plain jpeg", testJPEG},
		{"xor wrapped jpeg", xorWrapped(0xA5, testJPEG)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			target := filepath.Join(t.TempDir(), "out.jpg")
			written, err := DownloadImage(context.Background(), byteFetch(func(_ context.Context, _ string) ([]byte, error) { return tc.payload, nil }), "https://media.example.test/i", target)
			if err != nil {
				t.Fatalf("DownloadImage error = %v", err)
			}
			if written != int64(len(testJPEG)) {
				t.Fatalf("written = %d, want %d", written, len(testJPEG))
			}
		})
	}
}

func TestDownloadImageRejectsNonImagePayloads(t *testing.T) {
	cases := []struct {
		name    string
		payload []byte
	}{
		{"html error page", []byte("<html><body>403 Forbidden</body></html>")},
		{"json error", []byte(`{"success":false,"message":"denied"}`)},
		{"cloudflare page", []byte("<!DOCTYPE html><title>Attention Required! | Cloudflare</title>")},
		{"empty response", nil},
		{"unknown binary", []byte{0x00, 0x01, 0x02, 0x03, 0xDE, 0xAD, 0xBE, 0xEF}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			target := filepath.Join(t.TempDir(), "out.jpg")
			_, err := DownloadImage(context.Background(), byteFetch(func(_ context.Context, _ string) ([]byte, error) { return tc.payload, nil }), "https://media.example.test/i", target)
			if err == nil {
				t.Fatal("expected failure for non-image payload")
			}
			if _, statErr := os.Stat(target); !os.IsNotExist(statErr) {
				t.Fatalf("target %q must not exist after failed download", target)
			}
		})
	}
}

func TestDownloadImagePropagatesFetchError(t *testing.T) {
	target := filepath.Join(t.TempDir(), "out.jpg")
	_, err := DownloadImage(context.Background(), byteFetch(func(_ context.Context, _ string) ([]byte, error) { return nil, errors.New("HTTP 403") }), "https://media.example.test/i", target)
	if err == nil || !strings.Contains(err.Error(), "HTTP 403") {
		t.Fatalf("error = %v, want fetch error to propagate", err)
	}
}

func TestImagePayloadFormatDetectsKnownFormats(t *testing.T) {
	png := []byte("\x89PNG\r\n\x1a\n\x00\x00")
	gif := []byte("GIF89a\x00\x00")
	webp := append([]byte("RIFF\x00\x00\x00\x00WEBP"), make([]byte, 4)...)
	avif := append([]byte("\x00\x00\x00\x18ftypavif\x00\x00\x00\x00"), make([]byte, 8)...)
	heic := append([]byte("\x00\x00\x00\x18ftypheic\x00\x00\x00\x00"), make([]byte, 8)...)
	cases := []struct {
		name    string
		payload []byte
		want    string
	}{
		{"jpeg", testJPEG, "jpg"},
		{"png", png, "png"},
		{"gif", gif, "gif"},
		{"webp", webp, "webp"},
		{"avif", avif, "avif"},
		{"heic", heic, "heic"},
		{"html", []byte("<html>"), ""},
		{"empty", nil, ""},
	}
	for _, tc := range cases {
		got, ok := ImagePayloadFormat(tc.payload)
		if tc.want == "" {
			if ok {
				t.Fatalf("%s: expected unknown format", tc.name)
			}
			continue
		}
		if !ok || got != tc.want {
			t.Fatalf("%s: format = (%q, %v), want %q", tc.name, got, ok, tc.want)
		}
	}
}
