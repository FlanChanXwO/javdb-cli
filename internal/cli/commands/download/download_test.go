package download

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
)

func TestNewHelpDocumentsSinglePreviewImage(t *testing.T) {
	streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{}, streams)
	var out, errb bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errb)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("help error = %v", err)
	}
	for _, want := range []string{"--thumbnail", "--preview-image", "--preview-video", "only the first preview image"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("download help missing %q: %s", want, out.String())
		}
	}
}

func TestNewRequiresSelectedOutputBeforeNetwork(t *testing.T) {
	streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{}, streams)
	var out, errb bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errb)
	cmd.SetArgs([]string{"WTEX-15"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "set at least one") {
		t.Fatalf("expected media selection error, got %v", err)
	}
}

func TestNewRejectsWhitespaceOnlyOutput(t *testing.T) {
	streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{}, streams)
	var out, errb bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errb)
	cmd.SetArgs([]string{"WTEX-15", "--thumbnail", " \t "})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "set at least one") {
		t.Fatalf("expected whitespace-only error, got %v", err)
	}
}

// TestDownloadBatchRequiresPlaceholders 批量下载目标必须包含 {number}/{id}。
func TestDownloadBatchRequiresPlaceholders(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	t.Setenv("HOMEDRIVE", filepath.VolumeName(t.TempDir()))
	t.Setenv("HOMEPATH", strings.TrimPrefix(t.TempDir(), filepath.VolumeName(t.TempDir())))
	streams := invocation.NewStreams(strings.NewReader("SSIS-589\nHZGD-246\n"), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{}, streams)
	cmd.SetArgs([]string{"--thumbnail", "/tmp/out.jpg"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "{number} or {id}") {
		t.Fatalf("expected placeholder error, got %v", err)
	}
}

// TestDownloadBatchPreflightRejectsDuplicateTargets 全量展开后重复目标在写入前失败。
func TestDownloadBatchPreflightRejectsDuplicateTargets(t *testing.T) {
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
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	dir := t.TempDir()
	// 两个输入展开为同一路径 → 冲突（同 id 输入）。
	duplicateStdin := "{\"schema\":\"javdb.pipeline/v1\",\"kind\":\"movie\",\"ref\":\"SSIS-589\",\"id\":\"SAME\"}\n{\"schema\":\"javdb.pipeline/v1\",\"kind\":\"movie\",\"ref\":\"HZGD-246\",\"id\":\"SAME\"}\n"
	streams := invocation.NewStreams(strings.NewReader(duplicateStdin), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{Host: server.URL}, streams)
	cmd.SetArgs([]string{"--thumbnail", dir + "/{id}.jpg"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("expected duplicate-target error, got %v", err)
	}
	// 已存在文件 → 冲突。
	existing := dir + "/EXISTING.jpg"
	if err := os.WriteFile(existing, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	streams = invocation.NewStreams(strings.NewReader("SSIS-589\nHZGD-246\n"), &bytes.Buffer{}, &bytes.Buffer{})
	cmd = New(&invocation.RootOptions{Host: server.URL}, streams)
	cmd.SetArgs([]string{"--thumbnail", dir + "/{number}.jpg"})
	// 改用同 number 不同 id 输入避免重复：用 JSONL 提供唯一 id。
	streams = invocation.NewStreams(strings.NewReader("{\"schema\":\"javdb.pipeline/v1\",\"kind\":\"movie\",\"ref\":\"SSIS-589\",\"id\":\"EXISTING\"}\n{\"schema\":\"javdb.pipeline/v1\",\"kind\":\"movie\",\"ref\":\"SSIS-589\",\"id\":\"OTHER\"}\n"), &bytes.Buffer{}, &bytes.Buffer{})
	cmd = New(&invocation.RootOptions{Host: server.URL}, streams)
	cmd.SetArgs([]string{"--thumbnail", dir + "/{id}.jpg"})
	err = cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected existing-target error, got %v", err)
	}
}

// TestDownloadPipelineIDDoesNotResolveAsNumber --id 在管道/非 TTY 路径也不得
// 调用番号搜索（resolver 无精确匹配时会回退首项，存在下载错影片风险）。
func TestDownloadPipelineIDDoesNotResolveAsNumber(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	var searchCalls int
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch {
		case request.URL.Path == "/api/v2/search":
			searchCalls++
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{
				"movies": []map[string]any{{"number": "SSIS-589", "id": "WRONG-ID"}},
			}})
		case strings.HasPrefix(request.URL.Path, "/api/v4/movies/CORRECT-ID"):
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{
				"movie": map[string]any{"number": "SSIS-589", "id": "CORRECT-ID", "thumb_url": "https://example.test/t.jpg"},
			}})
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	dir := t.TempDir()
	streams := invocation.NewStreams(strings.NewReader("CORRECT-ID\n"), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{Host: server.URL}, streams)
	cmd.SetArgs([]string{"--id", "--thumbnail", dir + "/{id}.jpg"})
	// 下载会失败（detail 无媒体字段时），但搜索绝不能发生。
	if err := cmd.Execute(); err == nil {
		// 若意外成功也可；核心断言是 search 未被调用。
	}
	if searchCalls != 0 {
		t.Fatalf("--id pipeline path called ResolveMovieID %d time(s)", searchCalls)
	}
}
