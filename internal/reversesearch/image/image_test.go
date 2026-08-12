package image

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

var (
	jpegMagic = []byte{0xFF, 0xD8, 0xFF, 0xE0}
	pngMagic  = []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}
	webpMagic = append([]byte("RIFF"), append(make([]byte, 4), []byte("WEBP")...)...)
)

func TestDetectFormatSupportsThreeFormats(t *testing.T) {
	for _, tc := range []struct {
		name   string
		header []byte
		want   Format
	}{
		{name: "jpeg", header: jpegMagic, want: JPEG},
		{name: "png", header: pngMagic, want: PNG},
		{name: "webp", header: webpMagic, want: WEBP},
		{name: "empty", header: nil, want: Unknown},
		{name: "short jpeg", header: []byte{0xFF, 0xD8}, want: Unknown},
		{name: "garbage", header: []byte("not an image at all"), want: Unknown},
		{name: "riff without webp", header: append([]byte("RIFF"), []byte("XXXX")...), want: Unknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := DetectFormat(tc.header); got != tc.want {
				t.Errorf("DetectFormat = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestReadStreamPreservesBytesAndComputesSHA256(t *testing.T) {
	raw := append([]byte{}, jpegMagic...)
	raw = append(raw, bytes.Repeat([]byte{0x01}, 1024)...)
	got, err := ReadStream(bytes.NewReader(raw), "sample.jpg")
	if err != nil {
		t.Fatalf("ReadStream: %v", err)
	}
	if !bytes.Equal(got.Bytes, raw) {
		t.Error("ReadStream did not preserve the original bytes")
	}
	if got.Format != JPEG {
		t.Errorf("format = %v, want jpeg", got.Format)
	}
	if got.Filename != "sample.jpg" {
		t.Errorf("filename = %q", got.Filename)
	}
	sum := sha256Sum(t, raw)
	if got.SHA256 != sum {
		t.Errorf("SHA-256 mismatch")
	}
}

func TestReadStreamRejectsEmptyAndUnknownData(t *testing.T) {
	if _, err := ReadStream(bytes.NewReader(nil), "empty.bin"); err == nil {
		t.Fatal("ReadStream accepted empty data")
	} else if imageError(err).Stage != StageFormat {
		t.Errorf("empty data stage = %q, want format", imageError(err).Stage)
	}
	if _, err := ReadStream(strings.NewReader("plain text"), "note.txt"); err == nil {
		t.Fatal("ReadStream accepted non-image data")
	} else if imageError(err).Stage != StageFormat {
		t.Errorf("text stage = %q, want format", imageError(err).Stage)
	}
}

func TestReadStreamAcceptsExactlyEightMiB(t *testing.T) {
	raw := make([]byte, MaxSize)
	copy(raw, pngMagic)
	got, err := ReadStream(bytes.NewReader(raw), "exact.png")
	if err != nil {
		t.Fatalf("ReadStream at exactly 8 MiB: %v", err)
	}
	if len(got.Bytes) != MaxSize {
		t.Errorf("length = %d, want %d", len(got.Bytes), MaxSize)
	}
}

func TestReadStreamRejectsBeyondEightMiB(t *testing.T) {
	raw := make([]byte, MaxSize+1)
	copy(raw, jpegMagic)
	_, err := ReadStream(bytes.NewReader(raw), "oversize.jpg")
	if err == nil {
		t.Fatal("ReadStream accepted data beyond 8 MiB")
	}
	imageErr := imageError(err)
	if imageErr.Stage != StageSize {
		t.Errorf("stage = %q, want size", imageErr.Stage)
	}
}

func TestReadStreamRejectsReaderErrors(t *testing.T) {
	_, err := ReadStream(&failingReader{}, "broken.jpg")
	if err == nil {
		t.Fatal("ReadStream accepted a failing reader")
	}
	if imageError(err).Stage != StageInput {
		t.Errorf("stage = %q, want input", imageError(err).Stage)
	}
}

func TestReadFileUsesMagicOverExtension(t *testing.T) {
	// 伪扩展名：文件名叫 .png，内容其实是 JPEG magic。
	path := filepath.Join(t.TempDir(), "fake.png")
	raw := append([]byte{}, jpegMagic...)
	raw = append(raw, 0x00, 0x01)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if got.Format != JPEG {
		t.Errorf("format = %v, want jpeg (magic must win over extension)", got.Format)
	}
}

func TestReadFileRejectsMissingPathAndDirectory(t *testing.T) {
	if _, err := ReadFile(filepath.Join(t.TempDir(), "missing.jpg")); err == nil {
		t.Fatal("ReadFile accepted a missing path")
	} else if imageError(err).Stage != StageInput {
		t.Errorf("missing path stage = %q, want input", imageError(err).Stage)
	}
	if _, err := ReadFile(t.TempDir()); err == nil {
		t.Fatal("ReadFile accepted a directory")
	} else if imageError(err).Stage != StageInput {
		t.Errorf("directory stage = %q, want input", imageError(err).Stage)
	}
}

func TestReadURLReadsFromLocalServer(t *testing.T) {
	raw := append([]byte{}, webpMagic...)
	raw = append(raw, 0x00)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write(raw)
	}))
	defer server.Close()
	got, err := ReadURL(context.Background(), server.Client(), server.URL)
	if err != nil {
		t.Fatalf("ReadURL: %v", err)
	}
	if got.Format != WEBP || !bytes.Equal(got.Bytes, raw) {
		t.Errorf("ReadURL returned wrong image: format=%v len=%d", got.Format, len(got.Bytes))
	}
}

func TestReadURLFollowsRedirectsAndRejectsBadScheme(t *testing.T) {
	finalRaw := append([]byte{}, pngMagic...)
	finalRaw = append(finalRaw, 0x00)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/start":
			http.Redirect(writer, request, "/middle", http.StatusFound)
		case "/middle":
			http.Redirect(writer, request, "/final", http.StatusTemporaryRedirect)
		case "/final":
			_, _ = writer.Write(finalRaw)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	got, err := ReadURL(context.Background(), server.Client(), server.URL+"/start")
	if err != nil {
		t.Fatalf("ReadURL through redirects: %v", err)
	}
	if got.Format != PNG {
		t.Errorf("format = %v, want png", got.Format)
	}

	// 重定向到非 HTTP(S) scheme 必须被拒绝。
	weird := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.Redirect(writer, request, "file:///etc/hosts", http.StatusFound)
	}))
	defer weird.Close()
	if _, err := ReadURL(context.Background(), weird.Client(), weird.URL); err == nil {
		t.Fatal("ReadURL accepted a redirect to a non-HTTP scheme")
	}
}

func TestReadURLRejectsNonSuccessStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.Error(writer, "not found", http.StatusNotFound)
	}))
	defer server.Close()
	_, err := ReadURL(context.Background(), server.Client(), server.URL)
	if err == nil {
		t.Fatal("ReadURL accepted HTTP 404")
	}
	if imageError(err).Stage != StageStatus {
		t.Errorf("stage = %q, want status", imageError(err).Stage)
	}
}

func TestReadURLRejectsNonHTTPScheme(t *testing.T) {
	_, err := ReadURL(context.Background(), http.DefaultClient, "ftp://example.test/image.jpg")
	if err == nil {
		t.Fatal("ReadURL accepted ftp://")
	}
	if imageError(err).Stage != StageInput {
		t.Errorf("stage = %q, want input", imageError(err).Stage)
	}
}

func TestReadURLUsesProxyTransport(t *testing.T) {
	raw := append([]byte{}, jpegMagic...)
	raw = append(raw, 0x00)
	var proxySawRequest sync.Once
	seen := make(chan struct{}, 1)
	proxy := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		proxySawRequest.Do(func() { seen <- struct{}{} })
		_, _ = writer.Write(raw)
	}))
	defer proxy.Close()

	proxyURL, err := parseProxyURL(proxy.URL)
	if err != nil {
		t.Fatalf("parse proxy URL: %v", err)
	}
	client := &http.Client{Transport: newProxyTransport(proxyURL)}
	_, err = ReadURL(context.Background(), client, "http://upstream.example.test/image.jpg")
	if err != nil {
		t.Fatalf("ReadURL through proxy: %v", err)
	}
	select {
	case <-seen:
	case <-time.After(5 * time.Second):
		t.Fatal("proxy did not receive the image request")
	}
}

func TestReadURLCancellation(t *testing.T) {
	block := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		close(block)
		<-release
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-block
		cancel()
	}()
	_, err := ReadURL(ctx, server.Client(), server.URL)
	close(release)
	if err == nil {
		t.Fatal("ReadURL accepted a cancelled download")
	}
	if imageError(err).Stage != StageCancel {
		t.Errorf("stage = %q, want cancel", imageError(err).Stage)
	}
}

func TestReadURLStreamingSizeLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write(pngMagic)
		// 持续输出零字节；流式读取在第 8 MiB 边界立即失败，不等待 body 结束。
		for {
			if _, err := writer.Write(make([]byte, 64*1024)); err != nil {
				return
			}
		}
	}))
	defer server.Close()
	_, err := ReadURL(context.Background(), server.Client(), server.URL)
	if err == nil {
		t.Fatal("ReadURL accepted an oversized stream")
	}
	if imageError(err).Stage != StageSize {
		t.Errorf("stage = %q, want size", imageError(err).Stage)
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, errors.New("boom")
}

func sha256Sum(t *testing.T, raw []byte) [32]byte {
	t.Helper()
	return sha256.Sum256(raw)
}

func parseProxyURL(raw string) (*url.URL, error) {
	return url.Parse(raw)
}

func newProxyTransport(proxyURL *url.URL) *http.Transport {
	return &http.Transport{Proxy: http.ProxyURL(proxyURL)}
}

func imageError(err error) *Error {
	var imageErr *Error
	if !errors.As(err, &imageErr) {
		panic("expected image.Error, got " + err.Error())
	}
	return imageErr
}
