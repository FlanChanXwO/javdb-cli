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
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
)

func probePNGFixture(t *testing.T, width, height int) []byte {
	t.Helper()
	var out bytes.Buffer
	if err := png.Encode(&out, image.NewRGBA(image.Rect(0, 0, width, height))); err != nil {
		t.Fatalf("encode PNG fixture: %v", err)
	}
	return out.Bytes()
}

func TestProbeMovieAssetsPreservesOrderDeduplicatesAndUsesBestEffort(t *testing.T) {
	payload := probePNGFixture(t, 31, 17)
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
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client, err := New(WithHost(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	assets := []MovieAsset{
		{Type: assetTypeImage, URL: server.URL + "/good.png"},
		{Type: assetTypeImage, URL: server.URL + "/bad.png"},
		{Type: assetTypeImage, URL: server.URL + "/good.png"},
		{Type: assetTypeVideo, URL: server.URL + "/good.png"},
		{Type: "audio", URL: server.URL + "/good.png"},
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
	if infos[0].Metadata != wantImage || infos[2].Metadata != wantImage {
		t.Fatalf("deduplicated image metadata = %+v / %+v, want %+v", infos[0].Metadata, infos[2].Metadata, wantImage)
	}
	for _, index := range []int{1, 3, 4} {
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
}

func TestProbeMovieAssetsUsesBoundedWorkerPool(t *testing.T) {
	const (
		assetCount  = 1000
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
		for {
			maximum := maxActive.Load()
			if current <= maximum || maxActive.CompareAndSwap(maximum, current) {
				break
			}
		}
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
	baselineGoroutines := runtime.NumGoroutine()
	result := make(chan probeResult, 1)
	go func() {
		infos, err := client.ProbeMovieAssets(context.Background(), assets, MovieAssetProbeOptions{Concurrency: concurrency})
		result <- probeResult{infos: infos, err: err}
	}()
	for range concurrency {
		<-started
	}
	if increase := runtime.NumGoroutine() - baselineGoroutines; increase >= assetCount/2 {
		t.Fatalf("goroutine increase = %d for %d assets, worker pool appears per-asset", increase, assetCount)
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

func TestProbeMovieAssetsNormalizesVideoDuration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/short.m3u8":
			_, _ = writer.Write([]byte("#EXTM3U\n#EXTINF:0.4,\nshort.ts\n#EXT-X-ENDLIST\n"))
		case "/rounded.m3u8":
			_, _ = writer.Write([]byte("#EXTM3U\n#EXTINF:1.6,\nrounded.ts\n#EXT-X-ENDLIST\n"))
		case "/short.ts", "/rounded.ts":
			http.Error(writer, "SPS unavailable", http.StatusBadGateway)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	client, err := New(WithHost(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	infos, err := client.ProbeMovieAssets(context.Background(), []MovieAsset{
		{Type: assetTypeVideo, URL: server.URL + "/short.m3u8"},
		{Type: assetTypeVideo, URL: server.URL + "/rounded.m3u8"},
	}, MovieAssetProbeOptions{})
	if err != nil {
		t.Fatalf("ProbeMovieAssets() error = %v", err)
	}
	if infos[0].Metadata.Duration != 1 || infos[1].Metadata.Duration != 2 {
		t.Fatalf("durations = %d/%d, want 1/2", infos[0].Metadata.Duration, infos[1].Metadata.Duration)
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

func TestProbeMovieAssetsRejectsNegativeConcurrencyBeforeNetwork(t *testing.T) {
	var requests atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		http.Error(writer, "unexpected", http.StatusInternalServerError)
	}))
	defer server.Close()
	client, err := New(WithHost(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.ProbeMovieAssets(context.Background(), []MovieAsset{{Type: assetTypeImage, URL: server.URL + "/image.png"}}, MovieAssetProbeOptions{Concurrency: -1})
	if err == nil {
		t.Fatal("ProbeMovieAssets() accepted negative concurrency")
	}
	if got := requests.Load(); got != 0 {
		t.Fatalf("negative concurrency performed %d media requests", got)
	}
}
