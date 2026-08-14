package mark

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

// TestMarkBatchPartialFailure 批量 mark：部分远端写失败逐项可见并最终非零。
func TestMarkBatchPartialFailure(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	t.Setenv("HOMEDRIVE", filepath.VolumeName(t.TempDir()))
	t.Setenv("HOMEPATH", strings.TrimPrefix(t.TempDir(), filepath.VolumeName(t.TempDir())))
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch {
		case request.URL.Path == "/api/v2/search":
			number := request.URL.Query().Get("q")
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{
				"movies": []map[string]any{{"number": number, "id": "id-" + number}},
			}})
		case request.URL.Path == "/api/v1/movies/id-GHOST-999/reviews":
			http.Error(writer, "boom", http.StatusInternalServerError)
		case strings.Contains(request.URL.Path, "/reviews"):
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{"id": "rev-1"}})
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	streams := invocation.NewStreams(strings.NewReader("SSIS-589\nGHOST-999\n"), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{Host: server.URL}, streams)
	cmd.SetArgs([]string{"--watched", "--ndjson"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "1 of 2 items failed") {
		t.Fatalf("expected 1-of-2 failure, got %v", err)
	}
	out := streams.Out.(*bytes.Buffer).String()
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 {
		t.Fatalf("lines = %d:\n%s", len(lines), out)
	}
	if !strings.Contains(lines[0], `"kind":"movie"`) || !strings.Contains(lines[0], "id-SSIS-589") {
		t.Errorf("first envelope = %s", lines[0])
	}
	if !strings.Contains(lines[1], `"kind":"error"`) {
		t.Errorf("second envelope should be an error: %s", lines[1])
	}
}

// TestMarkWatchedSubmitsWatchedStatus 锁定 --watched 实际提交 watched
// 状态（回归：status 曾在 flag 解析前计算导致恒为 want_watch）。
func TestMarkWatchedSubmitsWatchedStatus(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	var submittedStatus string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch {
		case request.URL.Path == "/api/v2/search":
			number := request.URL.Query().Get("q")
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{
				"movies": []map[string]any{{"number": number, "id": "id-" + number}},
			}})
		case strings.HasSuffix(request.URL.Path, "/reviews"):
			_ = request.ParseForm()
			submittedStatus = request.Form.Get("status")
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{"id": "rev-1"}})
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	streams := invocation.NewStreams(strings.NewReader("SSIS-589\n"), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{Host: server.URL}, streams)
	cmd.SetArgs([]string{"--watched"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if submittedStatus != "watched" {
		t.Fatalf("submitted status = %q, want watched", submittedStatus)
	}

	submittedStatus = ""
	streams = invocation.NewStreams(strings.NewReader("SSIS-589\n"), &bytes.Buffer{}, &bytes.Buffer{})
	cmd = New(&invocation.RootOptions{Host: server.URL}, streams)
	cmd.SetArgs([]string{"--want"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute --want: %v", err)
	}
	if submittedStatus != "want_watch" {
		t.Fatalf("submitted status = %q, want want_watch", submittedStatus)
	}
}
