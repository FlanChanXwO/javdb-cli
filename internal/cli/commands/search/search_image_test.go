package search

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
)

var testJPEG = []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x01, 0x02}

// imageSearchFixture 组装反搜 provider 与 JavDB mock，返回 (server, javdb server, cleanup)。
func imageSearchFixture(t *testing.T) (*httptest.Server, *httptest.Server) {
	t.Helper()
	provider := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(`{"results":[
			{"video_code":"SSIS-589","best_similarity":95.2,"frames":[{"image_name":"SSIS-589_01-04-53.jpg","similarity":95.2,"timestamp":"01:04:53","thumbnail_url":"https://cdn.example/t.webp"}]},
			{"video_code":"GHOST-999","best_similarity":10.0,"frames":[]}]}`))
	}))
	javdb := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch {
		case request.URL.Path == "/api/v2/search":
			if request.URL.Query().Get("q") == "GHOST-999" {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"movies":[]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"movies":[
				{"number":"SSIS-589","id":"9DGB5X","title":"Test Title","number_remarks":""}]}}`))
		case strings.HasPrefix(request.URL.Path, "/api/v4/movies/9DGB5X"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"movie":{
				"id":"9DGB5X","number":"SSIS-589","title":"Test Title","number_remarks":""}}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	return provider, javdb
}

func executeSearch(t *testing.T, streams *invocation.Streams, options *invocation.RootOptions, args ...string) error {
	t.Helper()
	cmd := New(options, streams)
	cmd.SetArgs(args)
	return cmd.Execute()
}

// writeReverseSearchConfig 在隔离 HOME 下写入定义 test source 的配置。
func writeReverseSearchConfig(t *testing.T, providerURL string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOMEDRIVE", filepath.VolumeName(home))
	t.Setenv("HOMEPATH", strings.TrimPrefix(home, filepath.VolumeName(home)))
	configDir := filepath.Join(home, ".javdb-cli")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	config := "[reverse_search]\ndefault_source = \"test\"\n\n[[reverse_search.sources]]\nname = \"test\"\nurl = \"" + providerURL + "\"\n"
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestSearchImagePathAutoDetectsAndOutputsMatches(t *testing.T) {
	provider, javdbServer := imageSearchFixture(t)
	defer provider.Close()
	defer javdbServer.Close()
	writeReverseSearchConfig(t, provider.URL)

	imagePath := filepath.Join(t.TempDir(), "frame.jpg")
	if err := os.WriteFile(imagePath, testJPEG, 0o644); err != nil {
		t.Fatal(err)
	}

	streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	streams.InIsTerminal = true
	streams.OutIsTerminal = true
	err := executeSearch(t, streams, &invocation.RootOptions{Host: javdbServer.URL},
		imagePath, "--source", "test", "--no-cache")
	// 部分失败（GHOST-999 无法解析）必须在输出完成后非零。
	if err == nil {
		t.Fatal("partial candidate failure must exit non-zero")
	}
	out := streams.Out.(*bytes.Buffer).String()
	if !strings.Contains(out, "SSIS-589") || !strings.Contains(out, "95.2%") {
		t.Errorf("output lacks candidate line:\n%s", out)
	}
	if !strings.Contains(out, "01:04:53") {
		t.Errorf("output lacks frame timestamp:\n%s", out)
	}
	if !strings.Contains(out, "9DGB5X") {
		t.Errorf("output lacks matched movie projection:\n%s", out)
	}
	errOut := streams.Err.(*bytes.Buffer).String()
	if !strings.Contains(errOut, "GHOST-999") || !strings.Contains(errOut, "失败") {
		t.Errorf("stderr lacks per-item failure:\n%s", errOut)
	}
	if !strings.Contains(err.Error(), "1 of 2 candidates failed") {
		t.Errorf("error should count failures: %v", err)
	}
}

func TestSearchImageExplicitFlagAndJSON(t *testing.T) {
	provider, javdbServer := imageSearchFixture(t)
	defer provider.Close()
	defer javdbServer.Close()
	writeReverseSearchConfig(t, provider.URL)

	imagePath := filepath.Join(t.TempDir(), "frame.jpg")
	if err := os.WriteFile(imagePath, testJPEG, 0o644); err != nil {
		t.Fatal(err)
	}

	streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	err := executeSearch(t, streams, &invocation.RootOptions{Host: javdbServer.URL},
		"--image", imagePath, "--source", "test", "--json")
	if err == nil {
		t.Fatal("partial failure must exit non-zero even in JSON mode")
	}
	var got map[string]any
	if err := json.Unmarshal(streams.Out.(*bytes.Buffer).Bytes(), &got); err != nil {
		t.Fatalf("JSON output invalid: %v", err)
	}
	matches, _ := got["matches"].([]any)
	if len(matches) != 2 {
		t.Fatalf("matches = %d, want 2", len(matches))
	}
	first, _ := matches[0].(map[string]any)
	if first["movie_id"] != "9DGB5X" {
		t.Errorf("first match movie_id = %v", first["movie_id"])
	}
}

func TestSearchImageFromStdinMagicDefaultsToText(t *testing.T) {
	provider, javdbServer := imageSearchFixture(t)
	defer provider.Close()
	defer javdbServer.Close()
	writeReverseSearchConfig(t, provider.URL)

	streams := invocation.NewStreams(bytes.NewReader(testJPEG), &bytes.Buffer{}, &bytes.Buffer{})
	err := executeSearch(t, streams, &invocation.RootOptions{Host: javdbServer.URL})
	if err == nil {
		t.Fatal("partial failure must exit non-zero")
	}
	out := streams.Out.(*bytes.Buffer).String()
	// 非 TTY 纯文本 stdout 只保留成功番号，供 magnets 消费。
	if out != "SSIS-589\n" {
		t.Errorf("stdin image text output = %q, want stable movie ref", out)
	}
	errOut := streams.Err.(*bytes.Buffer).String()
	if !strings.Contains(errOut, "SSIS-589") || !strings.Contains(errOut, "01:04:53") || !strings.Contains(errOut, "GHOST-999") || !strings.Contains(errOut, "失败") {
		t.Errorf("stdin image stderr lacks diagnostics:\n%s", errOut)
	}
	if strings.Contains(out, `"kind":`) {
		t.Errorf("stdin image unexpectedly emitted NDJSON without --ndjson:\n%s", out)
	}
}

// TestSearchImageTextSkipsMovieDetail 验证非 TTY 稳定 ref 路径只解析影片 ID，
// 不为成功候选追加 MovieDetail 请求。
func TestSearchImageTextSkipsMovieDetail(t *testing.T) {
	provider := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(`{"results":[{"video_code":"SSIS-589","best_similarity":95.2,"frames":[]}]}`))
	}))
	defer provider.Close()
	var detailRequests atomic.Int64
	javdbServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch {
		case request.URL.Path == "/api/v2/search":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"movies":[{"number":"SSIS-589","id":"9DGB5X"}]}}`))
		case strings.HasPrefix(request.URL.Path, "/api/v4/movies/9DGB5X"):
			detailRequests.Add(1)
			_, _ = writer.Write([]byte(`{"success":true,"data":{"movie":{"id":"9DGB5X","number":"SSIS-589"}}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer javdbServer.Close()
	writeReverseSearchConfig(t, provider.URL)

	streams := invocation.NewStreams(bytes.NewReader(testJPEG), &bytes.Buffer{}, &bytes.Buffer{})
	if err := executeSearch(t, streams, &invocation.RootOptions{Host: javdbServer.URL}); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if got := streams.Out.(*bytes.Buffer).String(); got != "SSIS-589\n" {
		t.Fatalf("text output = %q", got)
	}
	if detailRequests.Load() != 0 {
		t.Fatalf("MovieDetail requests = %d, want 0", detailRequests.Load())
	}
}

// TestSearchImageMagnetsEndToEnd 覆盖 provider → 严格 ID 解析 → 磁力请求，
// 并锁定 text、NDJSON、JSON 三种输出；该快路径不得请求 MovieDetail。
func TestSearchImageMagnetsEndToEnd(t *testing.T) {
	provider := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(`{"results":[{"video_code":"SSIS-589","best_similarity":95.2,"frames":[]}]}`))
	}))
	defer provider.Close()
	var detailRequests atomic.Int64
	javdbServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch {
		case request.URL.Path == "/api/v2/search":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"movies":[{"number":"SSIS-589","id":"9DGB5X"}]}}`))
		case request.URL.Path == "/api/v1/movies/9DGB5X/magnets":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"magnets":[{"name":"HD","hash":"AAA","size":4096,"hd":true}]}}`))
		case strings.HasPrefix(request.URL.Path, "/api/v4/movies/9DGB5X"):
			detailRequests.Add(1)
			http.NotFound(writer, request)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer javdbServer.Close()
	writeReverseSearchConfig(t, provider.URL)

	for _, tc := range []struct {
		name string
		flag string
	}{
		{name: "text"},
		{name: "ndjson", flag: "--ndjson"},
		{name: "json", flag: "--json"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			streams := invocation.NewStreams(bytes.NewReader(testJPEG), &bytes.Buffer{}, &bytes.Buffer{})
			args := []string{"--magnets", "1", "--source", "test", "--no-cache"}
			if tc.flag != "" {
				args = append(args, tc.flag)
			}
			if err := executeSearch(t, streams, &invocation.RootOptions{Host: javdbServer.URL}, args...); err != nil {
				t.Fatalf("execute: %v", err)
			}
			out := streams.Out.(*bytes.Buffer).String()
			switch tc.flag {
			case "--ndjson":
				var envelope map[string]any
				if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &envelope); err != nil {
					t.Fatal(err)
				}
				if envelope["kind"] != "magnet" || envelope["ref"] != "SSIS-589" {
					t.Fatalf("NDJSON envelope = %v", envelope)
				}
			case "--json":
				var payload map[string]any
				if err := json.Unmarshal([]byte(out), &payload); err != nil {
					t.Fatal(err)
				}
				matches, _ := payload["matches"].([]any)
				if len(matches) != 1 {
					t.Fatalf("JSON matches = %v", payload["matches"])
				}
			default:
				if !strings.Contains(out, "magnet:?xt=urn:btih:AAA") {
					t.Fatalf("text output = %q", out)
				}
			}
		})
	}
	if detailRequests.Load() != 0 {
		t.Fatalf("MovieDetail requests = %d, want 0", detailRequests.Load())
	}
}

func TestSearchImageMagnetsRejectsInvalidMinSizeBeforeUpload(t *testing.T) {
	var uploads atomic.Int64
	provider := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		uploads.Add(1)
		_, _ = writer.Write([]byte(`{"results":[]}`))
	}))
	defer provider.Close()
	writeReverseSearchConfig(t, provider.URL)

	streams := invocation.NewStreams(bytes.NewReader(testJPEG), &bytes.Buffer{}, &bytes.Buffer{})
	err := executeSearch(t, streams, &invocation.RootOptions{}, "--magnets", "1", "--min-size", "invalid", "--source", "test", "--no-cache")
	if err == nil || !strings.Contains(err.Error(), "invalid --min-size") {
		t.Fatalf("error = %v, want local min-size validation", err)
	}
	if uploads.Load() != 0 {
		t.Fatalf("provider uploads = %d, want 0 before local validation", uploads.Load())
	}
}

func TestSearchImageMagnetsWorkerContinuesAfterLinkFailures(t *testing.T) {
	provider := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(`{"results":[
			{"video_code":"BAD-001","best_similarity":90},
			{"video_code":"BAD-002","best_similarity":89},
			{"video_code":"BAD-003","best_similarity":88},
			{"video_code":"BAD-004","best_similarity":87},
			{"video_code":"BAD-005","best_similarity":86}
		]}`))
	}))
	defer provider.Close()
	javdbServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/api/v2/search" {
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{"movies": []map[string]any{}}})
			return
		}
		http.NotFound(writer, request)
	}))
	defer javdbServer.Close()
	writeReverseSearchConfig(t, provider.URL)

	streams := invocation.NewStreams(bytes.NewReader(testJPEG), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{Host: javdbServer.URL}, streams)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--magnets", "1", "--source", "test", "--no-cache", "--ndjson"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "5 of 5 candidates failed") {
		t.Fatalf("error = %v, want complete candidate failure summary", err)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("image magnet worker pool timed out after link failures")
	}
	lines := strings.Split(strings.TrimSpace(streams.Out.(*bytes.Buffer).String()), "\n")
	if len(lines) != 5 {
		t.Fatalf("NDJSON lines = %d, want 5", len(lines))
	}
	for index, line := range lines {
		var envelope map[string]any
		if err := json.Unmarshal([]byte(line), &envelope); err != nil {
			t.Fatal(err)
		}
		data, _ := envelope["data"].(map[string]any)
		if envelope["kind"] != "error" || data["stage"] != "link" || data["code"] != "resolve" {
			t.Errorf("line %d envelope = %v, want link/resolve error", index, envelope)
		}
	}
}

func TestSearchAmbiguousArgumentAndStdin(t *testing.T) {
	// 位置参数（非图片路径）+ 非空 stdin 同时存在 → 歧义错误。
	streams := invocation.NewStreams(bytes.NewReader(testJPEG), &bytes.Buffer{}, &bytes.Buffer{})
	err := executeSearch(t, streams, &invocation.RootOptions{}, "frame.jpg")
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("expected ambiguity error, got %v", err)
	}
}

func TestSearchStdinRejectsNonImage(t *testing.T) {
	// 位置参数 + 非空 stdin → 歧义错误。
	streams := invocation.NewStreams(strings.NewReader("plain text"), &bytes.Buffer{}, &bytes.Buffer{})
	err := executeSearch(t, streams, &invocation.RootOptions{}, "keyword")
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("expected ambiguity error for keyword + stdin, got %v", err)
	}
}

func TestSearchTextKeywordStillWorksWithTTYStdin(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	t.Setenv("HOMEDRIVE", filepath.VolumeName(t.TempDir()))
	t.Setenv("HOMEPATH", strings.TrimPrefix(t.TempDir(), filepath.VolumeName(t.TempDir())))
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(`{"success":true,"data":{"movies":[{"number":"SSIS-589","id":"x"}]}}`))
	}))
	defer server.Close()

	streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	streams.InIsTerminal = true
	if err := executeSearch(t, streams, &invocation.RootOptions{Host: server.URL}, "keyword"); err != nil {
		t.Fatalf("text search with TTY stdin failed: %v", err)
	}
	if !strings.Contains(streams.Out.(*bytes.Buffer).String(), "SSIS-589") {
		t.Errorf("text search output lost movies")
	}
}

// TestSearchImageURLGoesThroughProxy URL 图片读取必须走最终代理配置
// （与 provider/JavDB 共用），而不是直连目标域名。
func TestSearchImageURLGoesThroughProxy(t *testing.T) {
	provider, javdbServer := imageSearchFixture(t)
	defer provider.Close()
	defer javdbServer.Close()
	writeReverseSearchConfig(t, provider.URL)

	var proxySawRequest atomic.Bool
	upstream, _ := url.Parse(provider.URL)
	providerProxy := httputil.NewSingleHostReverseProxy(upstream)
	proxy := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodGet {
			// 图片 URL 读取必须经代理到达这里。
			proxySawRequest.Store(true)
			_, _ = writer.Write(testJPEG)
			return
		}
		// provider 的 multipart POST 也走同一代理，转发给真实 provider。
		providerProxy.ServeHTTP(writer, request)
	}))
	defer proxy.Close()

	streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	streams.InIsTerminal = true
	err := executeSearch(t, streams, &invocation.RootOptions{Host: javdbServer.URL, Proxy: proxy.URL},
		"--image", "http://upstream.invalid/frame.jpg", "--source", "test")
	if err == nil {
		t.Fatal("partial candidate failure must exit non-zero")
	}
	if !proxySawRequest.Load() {
		t.Fatal("image URL request did not go through the configured proxy")
	}
	if !strings.Contains(streams.Out.(*bytes.Buffer).String(), "SSIS-589") && !strings.Contains(streams.Err.(*bytes.Buffer).String(), "SSIS-589") {
		t.Errorf("proxy image result missing candidate: stdout=%q stderr=%q", streams.Out.(*bytes.Buffer).String(), streams.Err.(*bytes.Buffer).String())
	}
}
