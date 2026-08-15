package search

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	magnetscmd "github.com/FlanChanXwO/javdb-cli/internal/cli/commands/magnets"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
)

// TestSearchTextStdinBatchConsumesKeywords 文本 stdin 批处理：逐关键词搜索并
// 按 fan-out 输出每部影片的番号，顺序保持。
func TestSearchTextStdinBatchConsumesKeywords(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	t.Setenv("HOMEDRIVE", filepath.VolumeName(t.TempDir()))
	t.Setenv("HOMEPATH", strings.TrimPrefix(t.TempDir(), filepath.VolumeName(t.TempDir())))
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		keyword := request.URL.Query().Get("q")
		_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{
			"movies": []map[string]any{{"number": strings.ToUpper(keyword), "id": "id-" + keyword}},
		}})
	}))
	defer server.Close()

	streams := invocation.NewStreams(strings.NewReader("ssis-589\nhzgd-246\n"), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{Host: server.URL}, streams)
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	out := streams.Out.(*bytes.Buffer).String()
	// fan-out 后每部影片输出番号（大写），而非原始关键词。
	if out != "SSIS-589\nHZGD-246\n" {
		t.Errorf("fan-out pipeline output = %q, want movie numbers", out)
	}
}

// TestSearchNDJSONBatchRejectsWrongKind NDJSON 批处理：不兼容 kind 原位错误。
func TestSearchNDJSONBatchRejectsWrongKind(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	t.Setenv("HOMEDRIVE", filepath.VolumeName(t.TempDir()))
	t.Setenv("HOMEPATH", strings.TrimPrefix(t.TempDir(), filepath.VolumeName(t.TempDir())))
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{"movies": []map[string]any{}}})
	}))
	defer server.Close()

	streams := invocation.NewStreams(strings.NewReader("{\"schema\":\"javdb.pipeline/v1\",\"kind\":\"actor\",\"ref\":\"x\"}\n"), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{Host: server.URL}, streams)
	cmd.SetArgs([]string{"--ndjson"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "1 of 1 items failed") {
		t.Fatalf("expected batch failure, got %v", err)
	}
	out := streams.Out.(*bytes.Buffer).String()
	if !strings.Contains(out, `"kind":"error"`) || !strings.Contains(out, "unsupported kind") {
		t.Errorf("error envelope missing:\n%s", out)
	}
}

// TestSearchNDJSONFanOutMultipleMovies NDJSON 批处理 fan-out：单关键词搜索返回
// 多部影片，每部输出一个 movie 信封。
func TestSearchNDJSONFanOutMultipleMovies(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	t.Setenv("HOMEDRIVE", filepath.VolumeName(t.TempDir()))
	t.Setenv("HOMEPATH", strings.TrimPrefix(t.TempDir(), filepath.VolumeName(t.TempDir())))
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{
			"movies": []map[string]any{
				{"number": "SSIS-001", "id": "id-1"},
				{"number": "SSIS-002", "id": "id-2"},
			},
		}})
	}))
	defer server.Close()

	streams := invocation.NewStreams(strings.NewReader("SSIS\n"), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{Host: server.URL}, streams)
	cmd.SetArgs([]string{"--ndjson"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	out := streams.Out.(*bytes.Buffer).String()
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 fan-out envelopes, got %d lines", len(lines))
	}
	for i, want := range []string{"SSIS-001", "SSIS-002"} {
		var envelope map[string]any
		if err := json.Unmarshal([]byte(lines[i]), &envelope); err != nil {
			t.Fatal(err)
		}
		if envelope["ref"] != want {
			t.Errorf("line %d ref = %v, want %s", i, envelope["ref"], want)
		}
		if envelope["kind"] != "movie" {
			t.Errorf("line %d kind = %v, want movie", i, envelope["kind"])
		}
	}
}

// TestSearchMagnetsFlagOutputsMagnetURIs search --magnets 关键词路径：文本模式
// 输出磁力 URI，NDJSON 输出 kind=magnet 信封。
func TestSearchMagnetsFlagOutputsMagnetURIs(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	t.Setenv("HOMEDRIVE", filepath.VolumeName(t.TempDir()))
	t.Setenv("HOMEPATH", strings.TrimPrefix(t.TempDir(), filepath.VolumeName(t.TempDir())))
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/v2/search":
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{
				"movies": []map[string]any{{"number": "SSIS-001", "id": "id-1"}},
			}})
		case "/api/v1/movies/id-1/magnets":
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{
				"magnets": []map[string]any{
					{"name": "HD", "hash": "AAA", "size": float64(4096), "hd": true},
					{"name": "SD", "hash": "BBB", "size": float64(64)},
				},
			}})
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	// 文本模式：输出磁力 URI。
	streams := invocation.NewStreams(strings.NewReader("SSIS\n"), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{Host: server.URL}, streams)
	cmd.SetArgs([]string{"--magnets", "1"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	out := streams.Out.(*bytes.Buffer).String()
	if !strings.Contains(out, "magnet:?xt=urn:btih:AAA") {
		t.Errorf("text output should contain best magnet URI, got %q", out)
	}
	if strings.Contains(out, "BBB") {
		t.Errorf("text output should only contain top 1 magnet (AAA), got %q", out)
	}

	// NDJSON 模式：输出 kind=magnet 信封。
	streams2 := invocation.NewStreams(strings.NewReader("SSIS\n"), &bytes.Buffer{}, &bytes.Buffer{})
	cmd2 := New(&invocation.RootOptions{Host: server.URL}, streams2)
	cmd2.SetArgs([]string{"--magnets", "0", "--ndjson"})
	if err := cmd2.Execute(); err != nil {
		t.Fatalf("execute ndjson: %v", err)
	}
	out2 := streams2.Out.(*bytes.Buffer).String()
	var envelope map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out2)), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope["kind"] != "magnet" {
		t.Errorf("kind = %v, want magnet", envelope["kind"])
	}
	if envelope["ref"] != "SSIS-001" {
		t.Errorf("ref = %v, want SSIS-001", envelope["ref"])
	}
}

// TestSearchMagnetsRejectsNonMovieType search --magnets 与非 movie --type 互斥。
func TestSearchMagnetsRejectsNonMovieType(t *testing.T) {
	streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{}, streams)
	cmd.SetArgs([]string{"--magnets", "3", "--type", "actor"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "only supported for movie search") {
		t.Fatalf("expected type mismatch error, got %v", err)
	}
}

// TestSearchPipelineToMagnets end-to-end: search --ndjson | magnets --ndjson
// 验证搜索 fan-out 的 movie 信封（带 id）可被 magnets 消费并输出 magnet 信封。
func TestSearchPipelineToMagnets(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	t.Setenv("HOMEDRIVE", filepath.VolumeName(t.TempDir()))
	t.Setenv("HOMEPATH", strings.TrimPrefix(t.TempDir(), filepath.VolumeName(t.TempDir())))
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/v2/search":
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{
				"movies": []map[string]any{{"number": "SSIS-001", "id": "id-1"}},
			}})
		case "/api/v4/movies/id-1":
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{"movie": map[string]any{"magnets_count": float64(1)}}})
		case "/api/v1/movies/id-1/magnets":
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{
				"magnets": []map[string]any{{"name": "HD", "hash": "AAA", "size": float64(4096), "hd": true}},
			}})
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	// Step 1: search --ndjson produces movie envelopes with id.
	searchStreams := invocation.NewStreams(strings.NewReader("SSIS\n"), &bytes.Buffer{}, &bytes.Buffer{})
	searchCmd := New(&invocation.RootOptions{Host: server.URL}, searchStreams)
	searchCmd.SetArgs([]string{"--ndjson"})
	if err := searchCmd.Execute(); err != nil {
		t.Fatalf("search: %v", err)
	}
	searchOutput := searchStreams.Out.(*bytes.Buffer).String()

	// Step 2: magnets --ndjson consumes the search output.
	magnetsStreams := invocation.NewStreams(strings.NewReader(searchOutput), &bytes.Buffer{}, &bytes.Buffer{})
	magnetsCmd := magnetscmd.New(&invocation.RootOptions{Host: server.URL}, magnetsStreams)
	magnetsCmd.SetArgs([]string{"--ndjson"})
	if err := magnetsCmd.Execute(); err != nil {
		t.Fatalf("magnets: %v", err)
	}
	magnetsOutput := magnetsStreams.Out.(*bytes.Buffer).String()
	var envelope map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(magnetsOutput)), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope["kind"] != "magnet" {
		t.Errorf("kind = %v, want magnet", envelope["kind"])
	}
	if envelope["ref"] != "SSIS-001" {
		t.Errorf("ref = %v, want SSIS-001", envelope["ref"])
	}
}
