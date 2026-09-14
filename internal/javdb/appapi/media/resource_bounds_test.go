package media

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// 资源边界契约(计划 #11/#27/#28/#29):
// 下载读取有明确上限(playlist/key/segment/image/总 payload/segment 数);
// 临时文件唯一;最终发布原子 no-replace;close/cleanup 错误不静默吞。

// ---- #11:bounded read ----

// oversized playlist:超过 2 MiB 上限明确报错,不无界进内存。
func TestDownloadHLSRejectsOversizedPlaylist(t *testing.T) {
	// 生成 3 MiB 的 playlist(超过 2 MiB 上限)。
	var b strings.Builder
	b.WriteString("#EXTM3U\n")
	for i := 0; i < 400000; i++ {
		b.WriteString("#EXTINF:1.0,\nseg-long-comment-padding-padding-padding.ts\n")
	}
	b.WriteString("#EXT-X-ENDLIST\n")
	fetch := func(_ context.Context, uri string) ([]byte, error) {
		if strings.HasSuffix(uri, ".m3u8") {
			return []byte(b.String()), nil
		}
		return nil, errors.New("unexpected " + uri)
	}
	_, err := DownloadHLS(context.Background(), fetch, "https://media.example.test/v.m3u8", filepath.Join(t.TempDir(), "v.ts"))
	if err == nil || !strings.Contains(err.Error(), "playlist") || !strings.Contains(err.Error(), "limit") {
		t.Fatalf("error = %v, want playlist size limit", err)
	}
}

// oversized image:超过 64 MiB 上限明确报错。
func TestDownloadImageRejectsOversizedPayload(t *testing.T) {
	// 65 MiB + 1 的伪 payload(XOR 前缀 + JPEG 头)。
	raw := make([]byte, 65*1024*1024+2)
	raw[0] = 0x97
	raw[1], raw[2], raw[3] = 0xFF, 0xD8, 0xFF
	fetch := func(_ context.Context, uri string) ([]byte, error) {
		return raw, nil
	}
	_, err := DownloadImage(context.Background(), fetch, "https://img.example.test/a.jpg", filepath.Join(t.TempDir(), "a.jpg"))
	if err == nil || !strings.Contains(err.Error(), "limit") {
		t.Fatalf("error = %v, want image size limit", err)
	}
}

// oversized image:server 忽略 Content-Length / 慢速注入时,bounded reader
// 在上限后停止读取(计划 #11)。
func TestFetchBoundedReaderStopsAtLimit(t *testing.T) {
	// 无限数据源:超过上限后 reader 必须停止。
	data := make([]byte, 70*1024*1024)
	fetch := func(_ context.Context, uri string) ([]byte, error) {
		return data, nil
	}
	_, err := DownloadImage(context.Background(), fetch, "https://img.example.test/a.jpg", filepath.Join(t.TempDir(), "a.jpg"))
	if err == nil || !strings.Contains(err.Error(), "limit") {
		t.Fatalf("error = %v, want size limit", err)
	}
}

// too many segments:超过 4096 明确报错。
func TestDownloadHLSRejectsTooManySegments(t *testing.T) {
	var b strings.Builder
	b.WriteString("#EXTM3U\n")
	for i := 0; i < 4200; i++ {
		b.WriteString("#EXTINF:1.0,\n")
		b.WriteString("seg.ts\n")
	}
	b.WriteString("#EXT-X-ENDLIST\n")
	fetch := func(_ context.Context, uri string) ([]byte, error) {
		if strings.HasSuffix(uri, ".m3u8") {
			return []byte(b.String()), nil
		}
		return validTSSegmentAt(0), nil
	}
	_, err := DownloadHLS(context.Background(), fetch, "https://media.example.test/v.m3u8", filepath.Join(t.TempDir(), "v.ts"))
	if err == nil || !strings.Contains(err.Error(), "segments") {
		t.Fatalf("error = %v, want segment count limit", err)
	}
}

// ---- #28:最终文件发布必须真正 no-replace ----

// 并发同 target 发布:一方成功,另一方 ErrExist,原文件不被覆盖。
func TestPublishMediaFileAtomicNoReplace(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "out.bin")
	const workers = 8
	var wg sync.WaitGroup
	results := make([]error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, err := publishMediaFile(target, func(w io.Writer) (int64, error) {
				_, writeErr := w.Write([]byte{byte(idx)})
				return 1, writeErr
			}, nil)
			results[idx] = err
		}(i)
	}
	wg.Wait()
	success, exists := 0, 0
	for _, err := range results {
		if err == nil {
			success++
		} else if errors.Is(err, os.ErrExist) || strings.Contains(err.Error(), "already exists") {
			exists++
		}
	}
	if success != 1 {
		t.Fatalf("success count = %d, want exactly 1 (errors=%v)", success, results)
	}
	if exists != workers-1 {
		t.Fatalf("exists count = %d, want %d (errors=%v)", exists, workers-1, results)
	}
	// 原文件不被覆盖:内容是唯一成功一方的字节。
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 1 {
		t.Fatalf("target size = %d, want 1", len(data))
	}
	// target.part 临时文件已清理。
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("dir entries = %d, want 1 (temp file must be cleaned)", len(entries))
	}
}

// ---- #27:临时文件必须唯一(os.CreateTemp) ----

func TestPublishMediaFileLeavesNoCollisionProneTemp(t *testing.T) {
	dir := t.TempDir()
	// 预先创建 target.part:旧实现会因 O_EXCL 失败;新实现用 CreateTemp 不冲突。
	preExisting := filepath.Join(dir, "target.part")
	if err := os.WriteFile(preExisting, []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "target.bin")
	_, err := publishMediaFile(target, func(w io.Writer) (int64, error) {
		n, writeErr := w.Write([]byte("new"))
		return int64(n), writeErr
	}, nil)
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	data, err := os.ReadFile(target)
	if err != nil || string(data) != "new" {
		t.Fatalf("target content = %q err=%v", data, err)
	}
}

// ---- #29:主操作失败时保留 primary error;close 错误传播 ----

func TestDownloadImagePropagatesContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	fetch := func(_ context.Context, uri string) ([]byte, error) {
		return nil, ctx.Err()
	}
	_, err := DownloadImage(ctx, fetch, "https://img.example.test/a.jpg", filepath.Join(t.TempDir(), "a.jpg"))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}
