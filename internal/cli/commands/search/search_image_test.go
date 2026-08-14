package search

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

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
	if !strings.Contains(out, "GHOST-999") || !strings.Contains(out, "失败") {
		t.Errorf("output lacks per-item failure:\n%s", out)
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
	// 管道只负责输入分类，未显式指定结构化输出时仍使用纯文本。
	if !strings.Contains(out, "SSIS-589") || !strings.Contains(out, "95.2%") {
		t.Errorf("stdin image text output lacks candidate:\n%s", out)
	}
	if !strings.Contains(out, "GHOST-999") || !strings.Contains(out, "失败") {
		t.Errorf("stdin image text output lacks failure:\n%s", out)
	}
	if strings.Contains(out, `"kind":`) {
		t.Errorf("stdin image unexpectedly emitted JSONL without --jsonl:\n%s", out)
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
	if !strings.Contains(streams.Out.(*bytes.Buffer).String(), "SSIS-589") {
		t.Errorf("proxy image result missing candidate")
	}
}
