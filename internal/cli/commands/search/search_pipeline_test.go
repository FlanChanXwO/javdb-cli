package search

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

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
