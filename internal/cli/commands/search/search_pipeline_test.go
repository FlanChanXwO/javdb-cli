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
// 输出 movie 信封，顺序保持。
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
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 {
		t.Fatalf("envelope lines = %d:\n%s", len(lines), out)
	}
	var first struct {
		Kind string `json:"kind"`
		Ref  string `json:"ref"`
		ID   string `json:"id"`
	}
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatal(err)
	}
	if first.Kind != "movie" || first.Ref != "ssis-589" || first.ID != "id-ssis-589" {
		t.Errorf("first envelope = %+v", first)
	}
}

// TestSearchJSONLBatchRejectsWrongKind JSONL 批处理：不兼容 kind 原位错误。
func TestSearchJSONLBatchRejectsWrongKind(t *testing.T) {
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
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "1 of 1 items failed") {
		t.Fatalf("expected batch failure, got %v", err)
	}
	out := streams.Out.(*bytes.Buffer).String()
	if !strings.Contains(out, `"kind":"error"`) || !strings.Contains(out, "unsupported kind") {
		t.Errorf("error envelope missing:\n%s", out)
	}
}
