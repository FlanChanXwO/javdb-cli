package javdb

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func probePNGFixture(t *testing.T, width, height int) []byte {
	t.Helper()
	var out bytes.Buffer
	if err := png.Encode(&out, image.NewRGBA(image.Rect(0, 0, width, height))); err != nil {
		t.Fatalf("encode PNG fixture: %v", err)
	}
	return out.Bytes()
}

// TestProbeMovieAssetsPreservesOrderDeduplicatesAndUsesBestEffort 是核心 SDK 契约：
// 输入顺序、Type+URL 去重、成功 metadata、单项 HTTP 失败、单项 transport 超时、
// unsupported type，最终整体仍然成功。
func TestProbeMovieAssetsPreservesOrderDeduplicatesAndUsesBestEffort(t *testing.T) {
	payload := probePNGFixture(t, 31, 17)
	// slowRelease 让挂起的 /slow.png handler 在测试完成后立即退出，
	// 避免 httptest.Server.Close 等待一个永远不会被取消的 handler。
	slowRelease := make(chan struct{})
	var (
		mu       sync.Mutex
		requests = map[string]int{}
	)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		mu.Lock()
		requests[request.URL.Path]++
		mu.Unlock()
		switch request.URL.Path {
		case "/good.png":
			_, _ = writer.Write(payload)
		case "/bad.png":
			http.Error(writer, "broken", http.StatusBadGateway)
		case "/slow.png":
			// 先发出 header 让 client 进入 body 读取，再挂起超过 client timeout。
			writer.WriteHeader(http.StatusOK)
			if flusher, ok := writer.(http.Flusher); ok {
				flusher.Flush()
			}
			<-slowRelease
		default:
			http.NotFound(writer, request)
		}
	}))
	// 先释放挂起的 handler，再关闭 server；单个 defer 保证顺序。
	defer func() {
		close(slowRelease)
		server.Close()
	}()

	// transport 的 timeout 以秒为单位取整，必须 >= 1s 才会生效。
	client, err := New(WithHost(server.URL), WithTimeout(1*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	assets := []MovieAsset{
		{Type: assetTypeImage, URL: server.URL + "/good.png"},
		{Type: assetTypeImage, URL: server.URL + "/bad.png"},
		{Type: assetTypeImage, URL: server.URL + "/slow.png"},
		{Type: assetTypeImage, URL: server.URL + "/good.png"},
		{Type: assetTypeVideo, URL: server.URL + "/good.png"},
		{Type: "audio", URL: server.URL + "/good.png"},
	}
	// 负数 concurrency 在发任何请求前被拒绝，属于同一 contract。
	if _, err := client.ProbeMovieAssets(context.Background(), assets, MovieAssetProbeOptions{Concurrency: -1}); err == nil {
		t.Fatal("ProbeMovieAssets() accepted negative concurrency")
	}
	infos, err := client.ProbeMovieAssets(context.Background(), assets, MovieAssetProbeOptions{Concurrency: 2})
	if err != nil {
		t.Fatalf("ProbeMovieAssets() error = %v", err)
	}
	if len(infos) != len(assets) {
		t.Fatalf("infos length = %d, want %d", len(infos), len(assets))
	}
	for index := range assets {
		if !reflect.DeepEqual(infos[index].Asset, assets[index]) {
			t.Fatalf("infos[%d].Asset = %+v, want %+v", index, infos[index].Asset, assets[index])
		}
	}
	wantImage := MovieAssetMetadata{Width: 31, Height: 17}
	if infos[0].Metadata != wantImage || infos[3].Metadata != wantImage {
		t.Fatalf("deduplicated image metadata = %+v / %+v, want %+v", infos[0].Metadata, infos[3].Metadata, wantImage)
	}
	// 单项 HTTP failure、单项 transport timeout、unsupported type 只丢失 metadata。
	for _, index := range []int{1, 2, 4, 5} {
		if infos[index].Metadata != (MovieAssetMetadata{}) {
			t.Fatalf("infos[%d].Metadata = %+v, want zero value", index, infos[index].Metadata)
		}
	}
	mu.Lock()
	defer mu.Unlock()
	if requests["/good.png"] != 2 {
		t.Fatalf("good URL requests = %d, want 2 (one image + one video)", requests["/good.png"])
	}
	if requests["/bad.png"] != 1 {
		t.Fatalf("bad URL requests = %d, want 1", requests["/bad.png"])
	}
	if requests["/slow.png"] != 1 {
		t.Fatalf("slow URL requests = %d, want 1", requests["/slow.png"])
	}
}

// TestProbeMovieAssetsUsesBoundedWorkerPool 证明并发上限由 worker pool 约束：
// 同时在途请求数不超过 concurrency，且每个 asset 恰好一个请求。
func TestProbeMovieAssetsUsesBoundedWorkerPool(t *testing.T) {
	const (
		assetCount  = 16
		concurrency = 3
	)
	payload := probePNGFixture(t, 11, 7)
	started := make(chan struct{}, concurrency)
	release := make(chan struct{})
	var active, maxActive, requestCount atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		current := active.Add(1)
		defer active.Add(-1)
		requestCount.Add(1)
		updateAtomicMax(&maxActive, current)
		select {
		case started <- struct{}{}:
		default:
		}
		<-release
		_, _ = writer.Write(payload)
	}))
	defer server.Close()

	client, err := New(WithHost(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	assets := make([]MovieAsset, assetCount)
	for index := range assets {
		assets[index] = MovieAsset{Type: assetTypeImage, URL: server.URL + "/image/" + strconv.Itoa(index)}
	}
	type probeResult struct {
		infos []MovieAssetInfo
		err   error
	}
	result := make(chan probeResult, 1)
	go func() {
		infos, err := client.ProbeMovieAssets(context.Background(), assets, MovieAssetProbeOptions{Concurrency: concurrency})
		result <- probeResult{infos: infos, err: err}
	}()
	for range concurrency {
		<-started
	}
	close(release)
	got := <-result
	if got.err != nil {
		t.Fatalf("ProbeMovieAssets() error = %v", got.err)
	}
	if len(got.infos) != assetCount {
		t.Fatalf("infos length = %d, want %d", len(got.infos), assetCount)
	}
	if gotMax := maxActive.Load(); gotMax > concurrency {
		t.Fatalf("max active requests = %d, want <= %d", gotMax, concurrency)
	}
	if gotRequests := requestCount.Load(); gotRequests != assetCount {
		t.Fatalf("requests = %d, want %d", gotRequests, assetCount)
	}
}

func TestProbeMovieAssetsReturnsContextCancellation(t *testing.T) {
	started := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		started <- struct{}{}
		<-request.Context().Done()
	}))
	defer server.Close()
	client, err := New(WithHost(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, err := client.ProbeMovieAssets(ctx, []MovieAsset{{Type: assetTypeImage, URL: server.URL + "/blocked.png"}}, MovieAssetProbeOptions{})
		result <- err
	}()
	<-started
	cancel()
	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Fatalf("ProbeMovieAssets() error = %v, want context.Canceled", err)
	}
}

// updateAtomicMax 以 CAS 循环维护原子最大值：普通并发测试与 stress 测试共用。
func updateAtomicMax(target *atomic.Int64, value int64) {
	for {
		current := target.Load()
		if value <= current || target.CompareAndSwap(current, value) {
			return
		}
	}
}
