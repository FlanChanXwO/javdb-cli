package assets

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
)

// assets list 契约(input.md 计划 #1/#7/#9/#11):
// 资产获取 → --type 过滤 → 生成 1..N 编号 → selector;
// TTY 输出编号+类型+描述;非 TTY 默认 TYPE<TAB>URL;--json/--ndjson 只有 type/url。

// listDetailFixture 的 <SERVER> 在 server 建立后替换为真实地址,
// 以便 list 的输出能直接喂给 download 测试(真管道)。
const listDetailFixture = `{"thumb_url":"<SERVER>/thumb.jpg","cover_url":"<SERVER>/cover.jpg","preview_video_url":"<SERVER>/preview.m3u8","preview_images":[
	{"large_url":"<SERVER>/p1-large.jpg","thumb_url":"<SERVER>/p1-thumb.jpg"},
	{"thumb_url":"<SERVER>/p2-thumb.jpg"},
	{"large_url":"<SERVER>/p3-large.jpg"}]}`

func newListServer(t *testing.T) *httptest.Server {
	t.Helper()
	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.URL.Path == "/api/v2/search":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"movies":[{"number":"SSIS-589","id":"m1"}]}}`))
		case request.URL.Path == "/api/v4/movies/m1":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"movie":` + strings.ReplaceAll(listDetailFixture, "<SERVER>", serverURL) + `}}`))
		case strings.HasSuffix(request.URL.Path, ".jpg"):
			_, _ = writer.Write(testJPEG)
		default:
			http.NotFound(writer, request)
		}
	}))
	serverURL = server.URL
	return server
}

// runList 执行 list 命令,返回 (stdout, serverURL)。
func runList(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	return runListWithStreams(t, false, args...)
}

func runListWithStreams(t *testing.T, outIsTerminal bool, args ...string) (string, string, error) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	t.Setenv("HOMEDRIVE", filepath.VolumeName(t.TempDir()))
	t.Setenv("HOMEPATH", strings.TrimPrefix(t.TempDir(), filepath.VolumeName(t.TempDir())))
	server := newListServer(t)
	defer server.Close()

	streams := invocation.NewStreams(strings.NewReader(""), &strings.Builder{}, &strings.Builder{})
	streams.OutIsTerminal = outIsTerminal
	cmd := NewList(&invocation.RootOptions{Host: server.URL}, streams)
	cmd.SetArgs(args)
	err := cmd.Execute()
	out := streams.Out.(*strings.Builder).String()
	return out, server.URL, err
}

func TestListPipeOutputIsTypeTabURL(t *testing.T) {
	out, _, err := runList(t, "SSIS-589")
	if err != nil {
		t.Fatalf("execute error = %v", err)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 6 {
		t.Fatalf("line count = %d, want 6 (out=%q)", len(lines), out)
	}
	wantTypes := []string{"image", "image", "image", "image", "image", "video"}
	wantPaths := []string{"/thumb.jpg", "/cover.jpg", "/p1-large.jpg", "/p2-thumb.jpg", "/p3-large.jpg", "/preview.m3u8"}
	for i, line := range lines {
		parts := strings.Split(line, "\t")
		if len(parts) != 2 {
			t.Fatalf("line %d = %q, want TYPE<TAB>URL", i+1, line)
		}
		if parts[0] != wantTypes[i] {
			t.Fatalf("line %d type = %q, want %q", i+1, parts[0], wantTypes[i])
		}
		if !strings.HasSuffix(parts[1], wantPaths[i]) {
			t.Fatalf("line %d url = %q, want suffix %q", i+1, parts[1], wantPaths[i])
		}
	}
}

func TestListTTYOutputRendersNumberedTable(t *testing.T) {
	out, _, err := runListWithStreams(t, true, "SSIS-589")
	if err != nil {
		t.Fatalf("execute error = %v", err)
	}
	wantLines := []string{
		"#  TYPE   DESCRIPTION",
		"1  image  thumbnail",
		"2  image  cover",
		"3  image  preview 1",
		"4  image  preview 2",
		"5  image  preview 3",
		"6  video  preview",
	}
	gotLines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(gotLines) != len(wantLines) {
		t.Fatalf("line count = %d, want %d (out=%q)", len(gotLines), len(wantLines), out)
	}
	for i := range wantLines {
		if gotLines[i] != wantLines[i] {
			t.Fatalf("line %d:\n got  = %q\n want = %q", i+1, gotLines[i], wantLines[i])
		}
	}
}

func TestListJSONOutputHasOnlyTypeAndURL(t *testing.T) {
	out, _, err := runList(t, "SSIS-589", "--json")
	if err != nil {
		t.Fatalf("execute error = %v", err)
	}
	var assets []map[string]any
	if err := json.Unmarshal([]byte(out), &assets); err != nil {
		t.Fatalf("json decode: %v (out=%q)", err, out)
	}
	if len(assets) != 6 {
		t.Fatalf("asset count = %d, want 6", len(assets))
	}
	for i, asset := range assets {
		if len(asset) != 2 || asset["type"] == nil || asset["url"] == nil {
			t.Fatalf("asset %d has fields beyond type/url: %v", i, asset)
		}
	}
	if assets[0]["type"] != "image" {
		t.Fatalf("first asset = %v", assets[0])
	}
	if assets[5]["type"] != "video" {
		t.Fatalf("last asset = %v", assets[5])
	}
}

func TestListNDJSONOutputHasOnlyTypeAndURL(t *testing.T) {
	out, _, err := runList(t, "SSIS-589", "--ndjson")
	if err != nil {
		t.Fatalf("execute error = %v", err)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 6 {
		t.Fatalf("line count = %d, want 6 (out=%q)", len(lines), out)
	}
	var first map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("ndjson decode: %v", err)
	}
	if len(first) != 2 || first["type"] != "image" {
		t.Fatalf("first ndjson line = %v", first)
	}
}

func TestListTypeFilterRenumbersAfterFilter(t *testing.T) {
	out, _, err := runList(t, "SSIS-589", "--type", "video")
	if err != nil {
		t.Fatalf("execute error = %v", err)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 1 || !strings.HasPrefix(lines[0], "video\t") || !strings.HasSuffix(lines[0], "/preview.m3u8") {
		t.Fatalf("video filter output = %q", out)
	}

	out, _, err = runList(t, "SSIS-589", "--type", "image", "1-2")
	if err != nil {
		t.Fatalf("execute error = %v", err)
	}
	lines = strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("image filter + selector lines = %d, want 2 (%q)", len(lines), out)
	}
	for i, line := range lines {
		if !strings.HasPrefix(line, "image\t") {
			t.Fatalf("line %d = %q, want image asset", i+1, line)
		}
	}
}

func TestListRejectsInvalidType(t *testing.T) {
	_, _, err := runList(t, "SSIS-589", "--type", "audio")
	if err == nil || !strings.Contains(err.Error(), `invalid --type "audio"`) {
		t.Fatalf("error = %v, want invalid --type", err)
	}
}

func TestListSelectorForms(t *testing.T) {
	cases := []struct {
		selector   []string
		wantLines  int
		wantSuffix []string
	}{
		{[]string{"1"}, 1, []string{"/thumb.jpg"}},
		{[]string{"1-2"}, 2, []string{"/thumb.jpg", "/cover.jpg"}},
		{[]string{"1,3"}, 2, []string{"/thumb.jpg", "/p1-large.jpg"}},
		// 空格分隔的多个 selector 参数:1 3 5。
		{[]string{"1", "3", "5"}, 3, []string{"/thumb.jpg", "/p1-large.jpg", "/p3-large.jpg"}},
		{[]string{"3-4,6"}, 3, []string{"/p1-large.jpg", "/p2-thumb.jpg", "/preview.m3u8"}},
	}
	for _, tc := range cases {
		out, _, err := runList(t, append([]string{"SSIS-589"}, tc.selector...)...)
		if err != nil {
			t.Fatalf("selector %v: execute error = %v", tc.selector, err)
		}
		lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
		if len(lines) != tc.wantLines {
			t.Fatalf("selector %v: lines = %d, want %d (out=%q)", tc.selector, len(lines), tc.wantLines, out)
		}
		for i, line := range lines {
			if !strings.HasSuffix(line, tc.wantSuffix[i]) {
				t.Fatalf("selector %v: line %d = %q, want suffix %q", tc.selector, i+1, line, tc.wantSuffix[i])
			}
		}
	}
}

func TestListRejectsInvalidAndOutOfRangeSelector(t *testing.T) {
	_, _, err := runList(t, "SSIS-589", "4-1")
	if err == nil || !strings.Contains(err.Error(), `invalid selector "4-1"`) {
		t.Fatalf("error = %v, want invalid selector", err)
	}
	_, _, err = runList(t, "SSIS-589", "99")
	if err == nil || !strings.Contains(err.Error(), "out of range (1-6)") {
		t.Fatalf("error = %v, want out of range", err)
	}
}
