package javdb_test

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/sdk"
)

var testJPEG = []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x01}

// providerServer 模拟统一反搜响应。
func providerServer(t *testing.T, payload string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(payload))
	}))
}

// javdbServer 模拟 JavDB search 与 movie detail 端点（app api envelope）。
// search 按 q 参数大小写不敏感过滤 movies。
func javdbServer(t *testing.T, movies []map[string]any, details map[string]map[string]any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch {
		case request.URL.Path == "/api/v2/search":
			query := strings.ToUpper(request.URL.Query().Get("q"))
			filtered := make([]map[string]any, 0, len(movies))
			for _, movie := range movies {
				if query == "" || strings.ToUpper(fmt.Sprint(movie["number"])) == query {
					filtered = append(filtered, movie)
				}
			}
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{"movies": filtered}})
		case len(request.URL.Path) > len("/api/v4/movies/") && request.URL.Path[:len("/api/v4/movies/")] == "/api/v4/movies/":
			id := request.URL.Path[len("/api/v4/movies/"):]
			movie, ok := details[id]
			if !ok {
				http.NotFound(writer, request)
				return
			}
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{"movie": movie}})
		default:
			http.NotFound(writer, request)
		}
	}))
}

type memoryCache struct {
	mu      sync.Mutex
	entries map[string]javdb.ReverseSearchResponse
	calls   int
}

func newMemoryCache() *memoryCache {
	return &memoryCache{entries: map[string]javdb.ReverseSearchResponse{}}
}

func (c *memoryCache) Get(_ context.Context, key string) (javdb.ReverseSearchResponse, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls++
	response, ok := c.entries[key]
	return response, ok, nil
}

func (c *memoryCache) Put(_ context.Context, key string, response javdb.ReverseSearchResponse) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls++
	c.entries[key] = response
	return nil
}

func TestReverseSearchExternalSource(t *testing.T) {
	server := providerServer(t, `{"results":[{"video_code":"SSIS-589","best_similarity":95.2,"frames":[
		{"image_name":"SSIS-589_01-04-53.jpg","similarity":95.2}]}]}`)
	defer server.Close()

	client, err := javdb.New(javdb.WithReverseSearch(javdb.ReverseSearchOptions{Retries: 1}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	response, err := client.ReverseSearch(context.Background(), javdb.ReverseSearchRequest{
		Image:    testJPEG,
		Filename: "frame.jpg",
		Source:   javdb.ReverseSearchSource{Name: "test", URL: server.URL},
	})
	if err != nil {
		t.Fatalf("ReverseSearch: %v", err)
	}
	if len(response.Candidates) != 1 || response.Candidates[0].VideoCode != "SSIS-589" {
		t.Fatalf("candidates = %+v", response.Candidates)
	}
	if response.Candidates[0].Frames[0].Timestamp != "" {
		t.Errorf("external source must not derive fields: %+v", response.Candidates[0].Frames[0])
	}
}

func TestReverseSearchCacheHitSkipsProvider(t *testing.T) {
	cache := newMemoryCache()
	server := providerServer(t, "never reached")
	defer server.Close()

	client, err := javdb.New(javdb.WithReverseSearch(javdb.ReverseSearchOptions{
		Cache:   cache,
		Retries: 1,
	}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	request := javdb.ReverseSearchRequest{
		Image:    testJPEG,
		Filename: "frame.jpg",
		Source:   javdb.ReverseSearchSource{Name: "test", URL: server.URL},
	}
	// 预填充缓存：key 必须含 source 前缀（source + 原图 SHA-256 隔离契约）。
	expected := javdb.ReverseSearchResponse{Source: "test", Candidates: []javdb.ReverseSearchCandidate{
		{VideoCode: "SSIS-589"},
	}}
	_ = cache.Put(context.Background(), "test:"+sha256Hex(t, testJPEG), expected)

	response, err := client.ReverseSearch(context.Background(), request)
	if err != nil {
		t.Fatalf("ReverseSearch: %v", err)
	}
	if response.Candidates[0].VideoCode != "SSIS-589" {
		t.Fatalf("cached response not returned: %+v", response.Candidates)
	}

	// BypassCache 必须真正走 provider（server 返回非 JSON 会被当作协议错误）。
	bypassRequest := request
	bypassRequest.BypassCache = true
	if _, err := client.ReverseSearch(context.Background(), bypassRequest); err == nil {
		t.Fatal("bypass cache still used the cache")
	}
}

func TestSearchByImageLinksCandidatesInOrder(t *testing.T) {
	provider := providerServer(t, `{"results":[
		{"video_code":"SSIS-589","best_similarity":95.2,"frames":[]},
		{"video_code":"HZGD-246","best_similarity":90.0,"frames":[]},
		{"video_code":"MIDV-854","best_similarity":88.0,"frames":[]}]}`)
	defer provider.Close()

	javdbServer := javdbServer(t,
		[]map[string]any{
			{"number": "SSIS-589", "id": "id-a"},
			{"number": "HZGD-246", "id": "id-b"},
			{"number": "midv-854", "id": "id-c"},
		},
		map[string]map[string]any{
			"id-a": {"number": "SSIS-589"},
			"id-b": {"number": "HZGD-246"},
			"id-c": {"number": "MIDV-854"},
		},
	)
	defer javdbServer.Close()

	client, err := javdb.New(javdb.WithHost(javdbServer.URL), javdb.WithReverseSearch(javdb.ReverseSearchOptions{Retries: 1}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	result, err := client.SearchByImage(context.Background(), javdb.ReverseSearchRequest{
		Image:    testJPEG,
		Filename: "frame.jpg",
		Source:   javdb.ReverseSearchSource{Name: "test", URL: provider.URL},
	}, javdb.ImageSearchOptions{})
	if err != nil {
		t.Fatalf("SearchByImage: %v", err)
	}
	if len(result.Matches) != 3 {
		t.Fatalf("matches = %d, want 3", len(result.Matches))
	}
	// 顺序必须与 provider 返回一致。
	want := []string{"SSIS-589", "HZGD-246", "MIDV-854"}
	for index, match := range result.Matches {
		if match.Error != nil {
			t.Errorf("match %d has error: %v", index, match.Error)
		}
		if match.Candidate.VideoCode != want[index] {
			t.Errorf("match %d order = %s, want %s", index, match.Candidate.VideoCode, want[index])
		}
		if match.MovieID != map[string]string{"SSIS-589": "id-a", "HZGD-246": "id-b", "MIDV-854": "id-c"}[want[index]] {
			t.Errorf("match %d movie id = %q", index, match.MovieID)
		}
		if match.Movie["number"] != want[index] {
			t.Errorf("match %d movie = %+v", index, match.Movie)
		}
	}
}

func TestSearchByImagePartialFailureKeepsOrder(t *testing.T) {
	provider := providerServer(t, `{"results":[
		{"video_code":"SSIS-589","best_similarity":95.2,"frames":[]},
		{"video_code":"GHOST-999","best_similarity":10.0,"frames":[]}]}`)
	defer provider.Close()

	javdbServer := javdbServer(t,
		[]map[string]any{{"number": "SSIS-589", "id": "id-a"}},
		map[string]map[string]any{"id-a": {"number": "SSIS-589"}},
	)
	defer javdbServer.Close()

	client, err := javdb.New(javdb.WithHost(javdbServer.URL), javdb.WithReverseSearch(javdb.ReverseSearchOptions{Retries: 1}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	result, err := client.SearchByImage(context.Background(), javdb.ReverseSearchRequest{
		Image:    testJPEG,
		Filename: "frame.jpg",
		Source:   javdb.ReverseSearchSource{Name: "test", URL: provider.URL},
	}, javdb.ImageSearchOptions{})
	if err != nil {
		t.Fatalf("SearchByImage: %v", err)
	}
	if result.Matches[0].Error != nil || result.Matches[0].MovieID != "id-a" {
		t.Errorf("first match should succeed: %+v", result.Matches[0])
	}
	second := result.Matches[1]
	if second.Error == nil || second.Error.Code != "resolve" {
		t.Errorf("second match should carry a resolve error: %+v", second.Error)
	}
	if second.MovieID != "" {
		t.Errorf("failed match must not carry a movie id: %+v", second)
	}
}

// TestSearchByImageSkipMovieDetail SkipMovieDetail=true 时仅做番号→ID 解析，
// 不请求完整详情；Match.MovieID 有值，Match.Movie 为 nil。
func TestSearchByImageSkipMovieDetail(t *testing.T) {
	provider := providerServer(t, `{"results":[
		{"video_code":"SSIS-589","best_similarity":95.2,"frames":[]}]}`)
	defer provider.Close()

	javdbServer := javdbServer(t,
		[]map[string]any{{"number": "SSIS-589", "id": "id-a"}},
		map[string]map[string]any{"id-a": {"number": "SSIS-589"}},
	)
	defer javdbServer.Close()

	client, err := javdb.New(javdb.WithHost(javdbServer.URL), javdb.WithReverseSearch(javdb.ReverseSearchOptions{Retries: 1}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	result, err := client.SearchByImage(context.Background(), javdb.ReverseSearchRequest{
		Image:    testJPEG,
		Filename: "frame.jpg",
		Source:   javdb.ReverseSearchSource{Name: "test", URL: provider.URL},
	}, javdb.ImageSearchOptions{SkipMovieDetail: true})
	if err != nil {
		t.Fatalf("SearchByImage: %v", err)
	}
	if len(result.Matches) != 1 {
		t.Fatalf("matches = %d, want 1", len(result.Matches))
	}
	match := result.Matches[0]
	if match.Error != nil {
		t.Fatalf("unexpected error: %v", match.Error)
	}
	if match.MovieID != "id-a" {
		t.Errorf("MovieID = %q, want id-a", match.MovieID)
	}
	if match.Movie != nil {
		t.Errorf("Movie should be nil when SkipMovieDetail is true, got %+v", match.Movie)
	}
}

func TestResolveMovieIDExactCaseInsensitive(t *testing.T) {
	javdbServer := javdbServer(t,
		[]map[string]any{{"number": "midv-854", "id": "id-c"}},
		nil,
	)
	defer javdbServer.Close()
	client, err := javdb.New(javdb.WithHost(javdbServer.URL))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	id, err := client.ResolveMovieIDExact(context.Background(), "MIDV-854")
	if err != nil {
		t.Fatalf("ResolveMovieIDExact: %v", err)
	}
	if id != "id-c" {
		t.Errorf("id = %q", id)
	}
}

func TestResolveMovieIDExactRejectsZeroAndMultiple(t *testing.T) {
	for _, tc := range []struct {
		name   string
		movies []map[string]any
	}{
		{name: "no exact match", movies: []map[string]any{{"number": "ABCD-123", "id": "id-other"}}},
		{name: "multiple exact matches", movies: []map[string]any{
			{"number": "SSIS-589", "id": "id-a"},
			{"number": "ssis-589", "id": "id-b"},
		}},
		{name: "exact match without id", movies: []map[string]any{{"number": "SSIS-589"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := javdbServer(t, tc.movies, nil)
			defer server.Close()
			client, err := javdb.New(javdb.WithHost(server.URL))
			if err != nil {
				t.Fatalf("New: %v", err)
			}
			if _, err := client.ResolveMovieIDExact(context.Background(), "SSIS-589"); err == nil {
				t.Fatal("ResolveMovieIDExact accepted ambiguous input")
			}
		})
	}
}

func TestReverseSearchRejectsEmptyImage(t *testing.T) {
	client, err := javdb.New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := client.ReverseSearch(context.Background(), javdb.ReverseSearchRequest{}); err == nil {
		t.Fatal("ReverseSearch accepted empty image bytes")
	}
}

func sha256Hex(t *testing.T, raw []byte) string {
	t.Helper()
	sum := sha256.Sum256(raw)
	return fmt.Sprintf("%x", sum)
}

// TestReverseSearchCacheIsolatedBySource 同一 Client + 同一缓存实例切换
// provider 时，第二个 source 不得命中第一个 source 的缓存。
func TestReverseSearchCacheIsolatedBySource(t *testing.T) {
	cache := newMemoryCache()
	first := providerServer(t, `{"results":[{"video_code":"SSIS-589","best_similarity":95.2,"frames":[]}]}`)
	defer first.Close()
	second := providerServer(t, `{"results":[{"video_code":"HZGD-246","best_similarity":90.0,"frames":[]}]}`)
	defer second.Close()

	client, err := javdb.New(javdb.WithReverseSearch(javdb.ReverseSearchOptions{
		Cache:   cache,
		Retries: 1,
	}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	request := func(sourceURL, name string) javdb.ReverseSearchRequest {
		return javdb.ReverseSearchRequest{
			Image:    testJPEG,
			Filename: "frame.jpg",
			Source:   javdb.ReverseSearchSource{Name: name, URL: sourceURL},
		}
	}
	firstResult, err := client.ReverseSearch(context.Background(), request(first.URL, "src-a"))
	if err != nil {
		t.Fatalf("first ReverseSearch: %v", err)
	}
	if firstResult.Candidates[0].VideoCode != "SSIS-589" {
		t.Fatalf("first source result = %+v", firstResult.Candidates)
	}
	// 同一图片 + 不同 source：必须真正请求第二个 provider，不得命中缓存。
	secondResult, err := client.ReverseSearch(context.Background(), request(second.URL, "src-b"))
	if err != nil {
		t.Fatalf("second ReverseSearch: %v", err)
	}
	if secondResult.Candidates[0].VideoCode != "HZGD-246" {
		t.Fatalf("second source leaked the first source cache: %+v", secondResult.Candidates)
	}
	// 同一 source 再次请求应命中缓存：不产生新的 Put（entries 不增长）。
	cache.mu.Lock()
	entriesBefore := len(cache.entries)
	cache.mu.Unlock()
	if _, err := client.ReverseSearch(context.Background(), request(first.URL, "src-a")); err != nil {
		t.Fatalf("cached ReverseSearch: %v", err)
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if len(cache.entries) != entriesBefore {
		t.Errorf("same-source repeat should hit cache without writing a new entry")
	}
}

// TestReverseSearchRejectsInvalidImages SDK 必须拒绝空/未知格式/超限图片，
// 与 CLI 共用同一校验契约，且在缓存与上传之前执行。
func TestReverseSearchRejectsInvalidImages(t *testing.T) {
	client, err := javdb.New(javdb.WithReverseSearch(javdb.ReverseSearchOptions{Retries: 1}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	oversize := make([]byte, 8<<20+1)
	copy(oversize, testJPEG)
	for _, tc := range []struct {
		name string
		raw  []byte
	}{
		{name: "empty", raw: nil},
		{name: "plain text", raw: []byte("not an image")},
		{name: "oversize", raw: oversize},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := client.ReverseSearch(context.Background(), javdb.ReverseSearchRequest{Image: tc.raw}); err == nil {
				t.Fatal("ReverseSearch accepted invalid image bytes")
			}
		})
	}
}
