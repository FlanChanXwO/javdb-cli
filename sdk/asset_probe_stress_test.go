package javdb

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type probeStressResult struct {
	assetCount      int
	requestCount    int64
	maxActive       int64
	invalidMetadata int
	// peakHeapAlloc 是 Go heap peak sample，不是 RSS。
	peakHeapAlloc uint64
	wallTime      time.Duration
}

func TestProbeMovieAssetsStress(t *testing.T) {
	if os.Getenv("JAVDB_ASSET_PROBE_STRESS") != "1" {
		t.Skip("set JAVDB_ASSET_PROBE_STRESS=1 to run the explicit probe stress matrix")
	}
	for _, assetCount := range []int{20, 200, 2000} {
		for _, concurrency := range []int{1, 2, 4, 8} {
			result := runProbeStressCase(t, assetCount, concurrency)
			if result.assetCount != assetCount || result.requestCount != int64(assetCount) {
				t.Fatalf("assets=%d concurrency=%d result=%+v, want one request/result per asset", assetCount, concurrency, result)
			}
			if result.maxActive > int64(concurrency) {
				t.Fatalf("assets=%d concurrency=%d max active=%d, want <= concurrency", assetCount, concurrency, result.maxActive)
			}
			if result.invalidMetadata != 0 {
				t.Fatalf("assets=%d concurrency=%d invalid metadata=%d", assetCount, concurrency, result.invalidMetadata)
			}
			t.Logf("assets=%d concurrency=%d wall=%s requests=%d max_active=%d peak_heap_alloc=%d", assetCount, concurrency, result.wallTime, result.requestCount, result.maxActive, result.peakHeapAlloc)
		}
	}
}

func runProbeStressCase(t *testing.T, assetCount, concurrency int) probeStressResult {
	t.Helper()
	payload := probePNGFixture(t, 31, 17)
	var active, maxActive, requestCount atomic.Int64
	var peakHeapAlloc atomic.Uint64
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		current := active.Add(1)
		defer active.Add(-1)
		requestCount.Add(1)
		updateAtomicMax(&maxActive, current)
		if !strings.HasPrefix(request.URL.Path, "/img/") {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write(payload)
	}))
	defer server.Close()

	client, err := New(WithHost(server.URL))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	assets := make([]MovieAsset, assetCount)
	paths := make(map[string]struct{}, assetCount)
	for index := range assets {
		path := "/img/" + strconv.Itoa(index) + ".png"
		paths[path] = struct{}{}
		assets[index] = MovieAsset{Type: assetTypeImage, URL: server.URL + path}
	}
	if len(paths) != assetCount {
		t.Fatalf("generated %d unique URLs, want %d", len(paths), assetCount)
	}

	readHeap := func() {
		var stats runtime.MemStats
		runtime.ReadMemStats(&stats)
		updateAtomicMaxUint(&peakHeapAlloc, stats.HeapAlloc)
	}
	readHeap()
	stopSampling := make(chan struct{})
	var sampler sync.WaitGroup
	sampler.Add(1)
	go func() {
		defer sampler.Done()
		ticker := time.NewTicker(time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				readHeap()
			case <-stopSampling:
				return
			}
		}
	}()
	start := time.Now()
	infos, err := client.ProbeMovieAssets(context.Background(), assets, MovieAssetProbeOptions{Concurrency: concurrency})
	wallTime := time.Since(start)
	close(stopSampling)
	sampler.Wait()
	readHeap()
	if err != nil {
		t.Fatalf("ProbeMovieAssets(assets=%d, concurrency=%d) error = %v", assetCount, concurrency, err)
	}
	invalidMetadata := 0
	for _, info := range infos {
		if info.Metadata != (MovieAssetMetadata{Width: 31, Height: 17}) {
			invalidMetadata++
		}
	}
	return probeStressResult{
		assetCount:      len(infos),
		requestCount:    requestCount.Load(),
		maxActive:       maxActive.Load(),
		invalidMetadata: invalidMetadata,
		peakHeapAlloc:   peakHeapAlloc.Load(),
		wallTime:        wallTime,
	}
}

func updateAtomicMax(target *atomic.Int64, value int64) {
	for {
		current := target.Load()
		if value <= current || target.CompareAndSwap(current, value) {
			return
		}
	}
}

func updateAtomicMaxUint(target *atomic.Uint64, value uint64) {
	for {
		current := target.Load()
		if value <= current || target.CompareAndSwap(current, value) {
			return
		}
	}
}
