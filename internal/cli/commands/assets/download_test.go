package assets

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	javdb "github.com/FlanChanXwO/javdb-cli/sdk"
)

// assets download 契约：
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

func TestDownloadConcurrentAutoNamesPublishWithoutOverwrite(t *testing.T) {
	server := downloadServer(t)
	defer server.Close()
	client, err := javdb.New(javdb.WithHost(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	const workers = 8
	errs := make([]error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			_, errs[index] = downloadAutoNamed(
				context.Background(),
				client,
				assetRecord{Type: "image", URL: server.URL + "/image.jpg"},
				dir,
				1,
			)
		}(i)
	}
	wg.Wait()

	successes := 0
	for _, err := range errs {
		if err == nil {
			successes++
			continue
		}
		if !errors.Is(err, os.ErrExist) {
			t.Fatalf("concurrent download error = %v, want os.ErrExist for losers", err)
		}
	}
	if successes != 1 {
		t.Fatalf("successful concurrent publishes = %d, want 1", successes)
	}
	data, err := os.ReadFile(filepath.Join(dir, "image-001.jpg"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, testJPEG) {
		t.Fatalf("published bytes = %x, want JPEG payload", data)
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
	// 真实可解析 SPS(baseline 66/level 40,640x480),供 MP4 mux 使用。
	sps := append(append([]byte{}, start...), 0x67, 0x42, 0x00, 0x28, 0xF8, 0x14, 0x07, 0xB2)
	pps := append(append([]byte{}, start...), 0x68, 0xCC, 0xDD)
	idr := append(append([]byte{}, start...), 0x65, 0x01, 0x02)
	nonIDR := append(append([]byte{}, start...), 0x41, 0x03, 0x04)
	adts := append([]byte{0xFF, 0xF1, 0x50, 0x80, 0x01, 0x40, 0x00}, 0x21, 0x10, 0x30)
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

// T11 补充:视频自动命名 .mp4 端到端(T10 remux 落地后),以及
// assets list 输出直接喂给 assets download 的真管道组合。

func TestDownloadVideoAutoNamesMP4(t *testing.T) {
	dir := t.TempDir()
	if _, err := runDownload(t, "video\t<SERVER>/v.m3u8\n", "-d", dir); err != nil {
		t.Fatalf("execute error = %v", err)
	}
	target := filepath.Join(dir, "video-001.mp4")
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read %s: %v", target, err)
	}
	if len(data) < 8 || string(data[4:8]) != "ftyp" {
		t.Fatalf("output is not an MP4: head=% X", data[:min(12, len(data))])
	}
	// Fast Start:moov 必须在 mdat 之前。
	if bytes.Index(data, []byte("moov")) > bytes.Index(data, []byte("mdat")) {
		t.Fatal("moov must precede mdat (fast start)")
	}
}

func TestListPipeIntoDownload(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	t.Setenv("HOMEDRIVE", filepath.VolumeName(t.TempDir()))
	t.Setenv("HOMEPATH", strings.TrimPrefix(t.TempDir(), filepath.VolumeName(t.TempDir())))
	server, _ := newListServer(t)
	defer server.Close()

	// list:非 TTY 输出 TYPE<TAB>URL。
	streams := invocation.NewStreams(strings.NewReader(""), &strings.Builder{}, &strings.Builder{})
	listCmd := NewList(&invocation.RootOptions{Host: server.URL}, streams)
	listCmd.SetArgs([]string{"SSIS-589", "--type", "image", "1-2"})
	if err := listCmd.Execute(); err != nil {
		t.Fatalf("list error = %v", err)
	}
	listOut := streams.Out.(*strings.Builder).String()

	// 把 list 输出原样接到 download stdin(与真实管道一致)。
	dlStreams := invocation.NewStreams(strings.NewReader(listOut), &strings.Builder{}, &strings.Builder{})
	dlCmd := NewDownload(&invocation.RootOptions{Host: server.URL}, dlStreams)
	dlCmd.SetArgs([]string{"-d", dir})
	if err := dlCmd.Execute(); err != nil {
		t.Fatalf("download error = %v", err)
	}
	for _, name := range []string{"image-001.jpg", "image-002.jpg"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("expected %s: %v", name, err)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ---- download stdout 与 stdin 流式处理 ----

// stdout 只输出最终路径;不再输出 `saved <path> (<bytes> bytes)`。
func TestDownloadOutputsOnlyFinalPath(t *testing.T) {
	dir := t.TempDir()
	stdout, err := runDownload(t, "image\t<SERVER>/a.jpg\n", "-d", dir)
	if err != nil {
		t.Fatalf("execute error = %v", err)
	}
	lines := strings.Split(strings.TrimRight(stdout, "\n"), "\n")
	if len(lines) != 1 {
		t.Fatalf("stdout lines = %d, want 1 (out=%q)", len(lines), stdout)
	}
	if strings.Contains(stdout, "saved") || strings.Contains(stdout, "bytes") {
		t.Fatalf("stdout must be path-only, got %q", stdout)
	}
	// Windows 路径分隔符与 Unix 不同:用 filepath.Separator 构造前缀断言。
	if !strings.HasPrefix(lines[0], dir+string(filepath.Separator)) {
		t.Fatalf("stdout = %q, want final path under %s", lines[0], dir)
	}
}

// -o 模式 stdout 只输出最终路径。
func TestDownloadOutOutputsOnlyFinalPath(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "preview.ts")
	stdout, err := runDownload(t, "video\t<SERVER>/v.m3u8\n", "-o", target)
	if err != nil {
		t.Fatalf("execute error = %v", err)
	}
	if strings.TrimRight(stdout, "\n") != target {
		t.Fatalf("stdout = %q, want %q", stdout, target)
	}
}

// stdin 流式处理：第一条失败立即停止，不缓存全部记录；
// 巨型 stdin 不应先全量读入。
func TestDownloadStreamsStdin(t *testing.T) {
	dir := t.TempDir()
	// 第一条坏行必须立即报错,后续行不处理。
	_, err := runDownload(t, "badline\nimage\t<SERVER>/a.jpg\n", "-d", dir)
	if err == nil || !strings.Contains(err.Error(), "invalid input") {
		t.Fatalf("error = %v, want invalid input at line 1", err)
	}
}

// -o 只需读取两条判断：第二条存在时报错，不读完整个 stdin。
func TestDownloadOStopsAfterSecondRecord(t *testing.T) {
	dir := t.TempDir()
	// 大量后续行;-o 模式不应处理它们。
	var lines strings.Builder
	lines.WriteString("image\t<SERVER>/a.jpg\nimage\t<SERVER>/b.jpg\n")
	for i := 0; i < 1000; i++ {
		lines.WriteString("bad-line-without-tab\n")
	}
	_, err := runDownload(t, lines.String(), "-o", filepath.Join(dir, "out.jpg"))
	if err == nil || !strings.Contains(err.Error(), "exactly one") {
		t.Fatalf("error = %v, want exactly one asset", err)
	}
}

// stdin 单行设置明确上限(64 KiB):超长行明确报错。
func TestDownloadRejectsOverlongLine(t *testing.T) {
	dir := t.TempDir()
	longURL := "image\t" + strings.Repeat("a", 70000) + "\n"
	_, err := runDownload(t, longURL, "-d", dir)
	if err == nil || !strings.Contains(err.Error(), "too long") {
		t.Fatalf("error = %v, want line too long", err)
	}
}
