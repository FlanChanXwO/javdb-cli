package assets

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
)

// assets download 契约(input.md 计划 #13/#14/#42/#43/#55):
// stdin 读 TYPE<TAB>URL;-d DIR 默认 .;-o 仅允许恰好一个资产;
// 默认命名 image-NNN.<magic ext> / video-NNN.mp4;不覆盖已有文件;坏输入明确失败。

var testJPEG = []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00, 0x01}

var testPNG = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\x01")

func downloadServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch {
		case strings.HasSuffix(request.URL.Path, ".jpg"):
			_, _ = writer.Write(testJPEG)
		case strings.HasSuffix(request.URL.Path, ".png"):
			_, _ = writer.Write(testPNG)
		case strings.HasSuffix(request.URL.Path, ".m3u8"):
			_, _ = writer.Write([]byte("#EXTM3U\n#EXT-X-TARGETDURATION:2\n#EXTINF:1.0,\nseg1.ts\n#EXT-X-ENDLIST\n"))
		case strings.HasSuffix(request.URL.Path, ".ts"):
			_, _ = writer.Write(validTSSegmentFixture())
		default:
			http.NotFound(writer, request)
		}
	}))
}

func runDownload(t *testing.T, stdin string, args ...string) (string, error) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	t.Setenv("HOMEDRIVE", filepath.VolumeName(t.TempDir()))
	t.Setenv("HOMEPATH", strings.TrimPrefix(t.TempDir(), filepath.VolumeName(t.TempDir())))
	server := downloadServer(t)
	defer server.Close()

	streams := invocation.NewStreams(strings.NewReader(strings.ReplaceAll(stdin, "<SERVER>", server.URL)), &strings.Builder{}, &strings.Builder{})
	cmd := NewDownload(&invocation.RootOptions{Host: server.URL}, streams)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return streams.Out.(*strings.Builder).String(), err
}

func TestDownloadSingleImageAutoNamesByMagic(t *testing.T) {
	dir := t.TempDir()
	out, err := runDownload(t, "image\t<SERVER>/pic.jpg\n", "-d", dir)
	if err != nil {
		t.Fatalf("execute error = %v", err)
	}
	target := filepath.Join(dir, "image-001.jpg")
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read %s: %v", target, err)
	}
	if string(data) != string(testJPEG) {
		t.Fatalf("image bytes mismatch")
	}
	if !strings.Contains(out, "image-001.jpg") {
		t.Fatalf("output %q must mention target", out)
	}
}

func TestDownloadMultipleImagesDetectEachExtension(t *testing.T) {
	dir := t.TempDir()
	stdin := "image\t<SERVER>/a.jpg\nimage\t<SERVER>/b.png\n"
	if _, err := runDownload(t, stdin, "-d", dir); err != nil {
		t.Fatalf("execute error = %v", err)
	}
	for _, name := range []string{"image-001.jpg", "image-002.png"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("expected %s: %v", name, err)
		}
	}
}

func TestDownloadVideoToExplicitTS(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "preview.ts")
	if _, err := runDownload(t, "video\t<SERVER>/v.m3u8\n", "-o", target); err != nil {
		t.Fatalf("execute error = %v", err)
	}
	data, err := os.ReadFile(target)
	if err != nil || string(data) != string(validTSSegmentFixture()) {
		t.Fatalf("ts content mismatch: %v", err)
	}
}

func TestDownloadDirFlagDefaultsToWorkingDir(t *testing.T) {
	wd := t.TempDir()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(wd); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	out, err := runDownload(t, "image\t<SERVER>/pic.jpg\n")
	if err != nil {
		t.Fatalf("execute error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(wd, "image-001.jpg")); err != nil {
		t.Fatalf("expected default-dir output: %v", err)
	}
	if !strings.Contains(out, "image-001.jpg") {
		t.Fatalf("output %q must mention target", out)
	}
}

func TestDownloadOWithMultipleInputsFails(t *testing.T) {
	dir := t.TempDir()
	stdin := "image\t<SERVER>/a.jpg\nimage\t<SERVER>/b.jpg\n"
	_, err := runDownload(t, stdin, "-d", dir, "-o", filepath.Join(dir, "one.jpg"))
	if err == nil || !strings.Contains(err.Error(), "-o requires exactly one asset") {
		t.Fatalf("error = %v, want -o requires exactly one asset", err)
	}
}

func TestDownloadRefusesExistingOutput(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "taken.jpg")
	if err := os.WriteFile(existing, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runDownload(t, "image\t<SERVER>/a.jpg\n", "-o", existing); err == nil {
		t.Fatal("expected failure for existing output")
	}
	if data, err := os.ReadFile(existing); err != nil || string(data) != "keep" {
		t.Fatalf("existing file must stay untouched: %v %q", err, data)
	}
}

func TestDownloadRefusesExistingAutoName(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "image-001.jpg"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runDownload(t, "image\t<SERVER>/a.jpg\n", "-d", dir); err == nil {
		t.Fatal("expected failure for existing auto-named target")
	}
	if data, err := os.ReadFile(filepath.Join(dir, "image-001.jpg")); err != nil || string(data) != "keep" {
		t.Fatalf("existing file must stay untouched: %v %q", err, data)
	}
}

func TestDownloadRejectsInvalidLines(t *testing.T) {
	dir := t.TempDir()
	for name, line := range map[string]string{
		"no tab":     "<SERVER>/a.jpg",
		"empty url":  "image\t",
		"only type":  "image",
		"garbage":    "garbage",
		"unknown":    "audio\t<SERVER>/a.mp3",
		"three cols": "image\t<SERVER>/a.jpg\textra",
	} {
		_, err := runDownload(t, line+"\n", "-d", dir)
		if err == nil {
			t.Fatalf("%s: expected failure", name)
		}
	}
}

func TestDownloadSkipsBlankLines(t *testing.T) {
	dir := t.TempDir()
	stdin := "\nimage\t<SERVER>/pic.jpg\n\n"
	if _, err := runDownload(t, stdin, "-d", dir); err != nil {
		t.Fatalf("execute error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "image-001.jpg")); err != nil {
		t.Fatalf("expected image-001.jpg: %v", err)
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
