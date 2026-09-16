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

// 资源处理契约：
// 媒体逐 segment 处理但不添加无依据的总大小、单段或数量上限;
// 临时文件唯一;最终发布原子 no-replace;close/cleanup 错误不静默吞。

// ---- 无固定资源上限 ----

func TestDownloadHLSPreservesPlaylistBeyondFormerLimit(t *testing.T) {
	// 该 playlist 超过历史 2 MiB 限制,但仍是合法的已结束媒体列表。
	var b strings.Builder
	b.WriteString("#EXTM3U\n")
	b.WriteString(strings.Repeat("# padding\n", 250000))
	b.WriteString("#EXTINF:1.0,\nseg.ts\n")
	b.WriteString("#EXT-X-ENDLIST\n")
	fetch := byteFetch(func(_ context.Context, uri string) ([]byte, error) {
		if strings.HasSuffix(uri, ".m3u8") {
			return []byte(b.String()), nil
		}
		return validTSSegmentAt(0), nil
	})
	written, err := DownloadHLS(context.Background(), fetch, "https://media.example.test/v.m3u8", filepath.Join(t.TempDir(), "v.ts"))
	if err != nil {
		t.Fatalf("download oversized playlist: %v", err)
	}
	if written != int64(len(validTSSegmentAt(0))) {
		t.Fatalf("written = %d, want one segment (%d)", written, len(validTSSegmentAt(0)))
	}
}

func TestDownloadHLSPreservesMoreThanFormerSegmentLimit(t *testing.T) {
	var b strings.Builder
	b.WriteString("#EXTM3U\n")
	for i := 0; i < 4200; i++ {
		b.WriteString("#EXTINF:1.0,\n")
		b.WriteString("seg.ts\n")
	}
	b.WriteString("#EXT-X-ENDLIST\n")
	var base uint64
	fetch := byteFetch(func(_ context.Context, uri string) ([]byte, error) {
		if strings.HasSuffix(uri, ".m3u8") {
			return []byte(b.String()), nil
		}
		segment := validTSSegmentAt(base)
		base += 90000 // 多段 fixture 必须维持真实媒体的递增时间轴。
		return segment, nil
	})
	written, err := DownloadHLS(context.Background(), fetch, "https://media.example.test/v.m3u8", filepath.Join(t.TempDir(), "v.ts"))
	if err != nil {
		t.Fatalf("download more than former segment limit: %v", err)
	}
	want := int64(4200 * len(validTSSegmentAt(0)))
	if written != want {
		t.Fatalf("written = %d, want %d", written, want)
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
	fetch := byteFetch(func(_ context.Context, uri string) ([]byte, error) {
		return nil, ctx.Err()
	})
	_, err := DownloadImage(ctx, fetch, "https://img.example.test/a.jpg", filepath.Join(t.TempDir(), "a.jpg"))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}

type countingReader struct {
	remaining int
	read      int
	reads     int
}

func (r *countingReader) Read(p []byte) (int, error) {
	r.reads++
	if r.remaining == 0 {
		return 0, io.EOF
	}
	n := len(p)
	if n > r.remaining {
		n = r.remaining
	}
	for i := 0; i < n; i++ {
		p[i] = 'x'
	}
	r.remaining -= n
	r.read += n
	return n, nil
}

func TestReadMediaBodyBoundedStopsAfterOneProbeByte(t *testing.T) {
	reader := &countingReader{remaining: 1024}
	_, err := readMediaBodyBounded(reader, 16)
	if err == nil || !strings.Contains(err.Error(), "exceeds fixed bound of 16 bytes") {
		t.Fatalf("bounded read error = %v, want fixed-bound error", err)
	}
	if reader.read != 17 {
		t.Fatalf("bounded reader consumed %d bytes, want exactly 17", reader.read)
	}
	if reader.reads != 1 {
		t.Fatalf("bounded reader performed %d reads, want 1", reader.reads)
	}
}
