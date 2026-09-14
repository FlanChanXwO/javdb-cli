package media

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
)

// probe 契约(计划 #2/#4/#5/#6):
// 图片单次 bounded probe,失败省略 metadata;视频 duration 来自 EXTINF,
// 宽高来自真实 SPS(不从 720p 名字推断);URL 去重;并发上界;取消传播。

// 构造最小 JPEG:0xFFD8 + SOF0(640x404)。
func testJPEG640x404() []byte {
	out := []byte{0xFF, 0xD8, 0xFF, 0xC0, 0x00, 0x11, 0x08,
		0x01, 0x94, // height 404
		0x02, 0x80, // width 640
		0x03, 0x01, 0x22, 0x00, 0x02, 0x11, 0x01, 0x03, 0x11, 0x01}
	return append(out, 0xFF, 0xD9)
}

// XOR 包装的测试 JPEG。
func testXORWrappedJPEG() []byte {
	img := testJPEG640x404()
	const key = byte(0x97)
	raw := make([]byte, len(img)+1)
	raw[0] = key
	for i, b := range img {
		raw[i+1] = b ^ key
	}
	return raw
}

func testPlaylistFetch(resources map[string][]byte) probeFetcher {
	return func(_ context.Context, uri string) ([]byte, error) {
		body, ok := resources[uri]
		if !ok {
			return nil, errors.New("unexpected " + uri)
		}
		return body, nil
	}
}

func probeTestLimits() ProbeLimits {
	limits := DefaultProbeLimits()
	return limits
}

func TestProbeImageExtractsDimensions(t *testing.T) {
	resources := map[string][]byte{
		"https://img.example.test/a.jpg": testJPEG640x404(),
	}
	result := ProbeAssetMetadata(context.Background(), testPlaylistFetch(resources),
		"image", "https://img.example.test/a.jpg", probeTestLimits())
	if !result.HasDimensions() {
		t.Fatal("image probe must extract dimensions")
	}
	if result.Width != 640 || result.Height != 404 {
		t.Fatalf("dimensions = %dx%d, want 640x404", result.Width, result.Height)
	}
}

func TestProbeImageUnwrapsXOR(t *testing.T) {
	resources := map[string][]byte{
		"https://img.example.test/a.jpg": testXORWrappedJPEG(),
	}
	result := ProbeAssetMetadata(context.Background(), testPlaylistFetch(resources),
		"image", "https://img.example.test/a.jpg", probeTestLimits())
	if !result.HasDimensions() || result.Width != 640 || result.Height != 404 {
		t.Fatalf("XOR-wrapped image probe = %dx%d has=%v", result.Width, result.Height, result.HasDimensions())
	}
}

// probe 失败只省略 metadata,不报错(计划 #5 Probe 失败语义)。
func TestProbeImageFailureOmitsMetadata(t *testing.T) {
	resources := map[string][]byte{
		"https://img.example.test/a.jpg": []byte("HTML error page"),
	}
	result := ProbeAssetMetadata(context.Background(), testPlaylistFetch(resources),
		"image", "https://img.example.test/a.jpg", probeTestLimits())
	if result.HasDimensions() {
		t.Fatal("failed probe must omit dimensions")
	}
}

// probe disabled:完全跳过。
func TestProbeDisabledSkipsAll(t *testing.T) {
	calls := 0
	fetch := func(_ context.Context, uri string) ([]byte, error) {
		calls++
		return testJPEG640x404(), nil
	}
	limits := probeTestLimits()
	limits.Enabled = false
	result := ProbeAssetMetadata(context.Background(), fetch, "image", "https://img.example.test/a.jpg", limits)
	if calls != 0 {
		t.Fatalf("probe disabled must not fetch, got %d calls", calls)
	}
	if result.HasDimensions() {
		t.Fatal("disabled probe must omit metadata")
	}
}

// context cancellation 正常传播(计划 #5 Probe 失败语义)。
func TestProbeContextCancellationPropagates(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	fetch := func(ctx context.Context, uri string) ([]byte, error) {
		return nil, ctx.Err()
	}
	result := ProbeAssetMetadata(ctx, fetch, "image", "https://img.example.test/a.jpg", probeTestLimits())
	if result.HasDimensions() {
		t.Fatal("cancelled probe must omit metadata")
	}
}

// 视频 probe:duration 来自 EXTINF 求和;宽高来自 SPS(计划 #5)。
func TestProbeVideoExtractsDurationAndDimensions(t *testing.T) {
	segment := validTSSegmentAt(0)
	resources := map[string][]byte{
		"https://media.example.test/720p.m3u8": []byte(
			"#EXTM3U\n#EXT-X-TARGETDURATION:2\n#EXTINF:58.5255,\nseg1.ts\n#EXTINF:58.5255,\nseg2.ts\n#EXT-X-ENDLIST\n"),
		"https://media.example.test/seg1.ts": segment,
	}
	result := ProbeAssetMetadata(context.Background(), testPlaylistFetch(resources),
		"video", "https://media.example.test/720p.m3u8", probeTestLimits())
	if !result.HasDuration() {
		t.Fatal("video probe must extract duration from EXTINF")
	}
	wantDur := 58.5255 + 58.5255
	if diff := result.Duration - wantDur; diff > 0.01 || diff < -0.01 {
		t.Fatalf("duration = %f, want %f (EXTINF sum)", result.Duration, wantDur)
	}
	if !result.HasDimensions() {
		t.Fatal("video probe must extract dimensions from SPS")
	}
	if result.Width != 640 || result.Height != 480 {
		t.Fatalf("dimensions = %dx%d, want 640x480 (from SPS)", result.Width, result.Height)
	}
}

// 720p.m3u8 不能被错误报告成 1280x720(计划 #32 E2E)。
func TestProbeVideoNeverInfersFromURLName(t *testing.T) {
	segment := validTSSegmentAt(0)
	resources := map[string][]byte{
		"https://media.example.test/720p.m3u8": []byte(
			"#EXTM3U\n#EXT-X-TARGETDURATION:2\n#EXTINF:1.0,\nseg1.ts\n#EXT-X-ENDLIST\n"),
		"https://media.example.test/seg1.ts": segment,
	}
	result := ProbeAssetMetadata(context.Background(), testPlaylistFetch(resources),
		"video", "https://media.example.test/720p.m3u8", probeTestLimits())
	if result.HasDimensions() && result.Width == 1280 && result.Height == 720 {
		t.Fatal("video probe must not infer resolution from URL name")
	}
}

// 并发与去重:同 URL 只 probe 一次(计划 #6)。
func TestProbeAssetsConcurrentDeduplicatesURLs(t *testing.T) {
	var mu sync.Mutex
	calls := 0
	fetch := func(_ context.Context, uri string) ([]byte, error) {
		// probe 并发回调的计数访问需要互斥。
		mu.Lock()
		calls++
		mu.Unlock()
		return testJPEG640x404(), nil
	}
	urls := []string{
		"https://img.example.test/a.jpg",
		"https://img.example.test/a.jpg", // 重复
		"https://img.example.test/b.jpg",
	}
	types := []string{"image", "image", "image"}
	results := ProbeAssetsConcurrent(context.Background(), fetch, types, urls, probeTestLimits())
	mu.Lock()
	defer mu.Unlock()
	if calls != 2 {
		t.Fatalf("fetch calls = %d, want 2 (URL dedup)", calls)
	}
	if len(results) != 3 {
		t.Fatalf("results = %d, want 3", len(results))
	}
	if !results[0].HasDimensions() || !results[1].HasDimensions() {
		t.Fatal("deduplicated positions must reuse first result")
	}
}

// 并发上界:concurrency=2 时同时最多 2 个在飞请求(计划 #6)。
func TestProbeAssetsConcurrentRespectsLimit(t *testing.T) {
	var mu sync.Mutex
	var exceeded int
	inFlight := 0
	maxInFlight := 0
	fetch := func(_ context.Context, uri string) ([]byte, error) {
		mu.Lock()
		inFlight++
		if inFlight > maxInFlight {
			maxInFlight = inFlight
		}
		if inFlight > 2 {
			exceeded++
		}
		mu.Unlock()
		defer func() {
			mu.Lock()
			inFlight--
			mu.Unlock()
		}()
		return testJPEG640x404(), nil
	}
	urls := make([]string, 10)
	types := make([]string, 10)
	for i := range urls {
		urls[i] = fmt.Sprintf("https://img.example.test/%d.jpg", i)
		types[i] = "image"
	}
	limits := probeTestLimits()
	limits.Concurrency = 2
	ProbeAssetsConcurrent(context.Background(), fetch, types, urls, limits)
	if exceeded > 0 {
		t.Fatalf("concurrency limit exceeded %d times", exceeded)
	}
	mu.Lock()
	defer mu.Unlock()
	if maxInFlight > 2 {
		t.Fatalf("max in flight = %d, want <= 2", maxInFlight)
	}
}
