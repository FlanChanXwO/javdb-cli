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

const listDetailFixture = `{"thumb_url":"https://m.example.test/thumb.jpg","cover_url":"https://m.example.test/cover.jpg","preview_video_url":"https://m.example.test/preview.m3u8","preview_images":[
	{"large_url":"https://m.example.test/p1-large.jpg","thumb_url":"https://m.example.test/p1-thumb.jpg"},
	{"thumb_url":"https://m.example.test/p2-thumb.jpg"},
	{"large_url":"https://m.example.test/p3-large.jpg"}]}`

func newListServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.URL.Path == "/api/v2/search":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"movies":[{"number":"SSIS-589","id":"m1"}]}}`))
		case request.URL.Path == "/api/v4/movies/m1":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"movie":` + listDetailFixture + `}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
}

func runList(t *testing.T, args ...string) (string, error) {
	t.Helper()
	return runListWithStreams(t, false, args...)
}

func runListWithStreams(t *testing.T, outIsTerminal bool, args ...string) (string, error) {
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
	if err != nil {
		return out, err
	}
	// TTY 模拟在真实终端下是 os.File,这里用 strings.Builder 断言纯文本。
	return out, nil
}

func TestListPipeOutputIsTypeTabURL(t *testing.T) {
	out, err := runList(t, "SSIS-589")
	if err != nil {
		t.Fatalf("execute error = %v", err)
	}
	want := strings.Join([]string{
		"image\thttps://m.example.test/thumb.jpg",
		"image\thttps://m.example.test/cover.jpg",
		"image\thttps://m.example.test/p1-large.jpg",
		"image\thttps://m.example.test/p2-thumb.jpg",
		"image\thttps://m.example.test/p3-large.jpg",
		"video\thttps://m.example.test/preview.m3u8",
		"",
	}, "\n")
	if out != want {
		t.Fatalf("pipe output mismatch:\n got  = %q\n want = %q", out, want)
	}
}

func TestListTTYOutputRendersNumberedTable(t *testing.T) {
	out, err := runListWithStreams(t, true, "SSIS-589")
	if err != nil {
		t.Fatalf("execute error = %v", err)
	}
	want := strings.Join([]string{
		"#  TYPE   DESCRIPTION",
		"1  image  thumbnail",
		"2  image  cover",
		"3  image  preview 1",
		"4  image  preview 2",
		"5  image  preview 3",
		"6  video  preview",
		"",
	}, "\n")
	if out != want {
		t.Fatalf("tty output mismatch:\n got  = %q\n want = %q", out, want)
	}
}

func TestListJSONOutputHasOnlyTypeAndURL(t *testing.T) {
	out, err := runList(t, "SSIS-589", "--json")
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
	if assets[0]["type"] != "image" || assets[0]["url"] != "https://m.example.test/thumb.jpg" {
		t.Fatalf("first asset = %v", assets[0])
	}
	if assets[5]["type"] != "video" {
		t.Fatalf("last asset = %v", assets[5])
	}
}

func TestListNDJSONOutputHasOnlyTypeAndURL(t *testing.T) {
	out, err := runList(t, "SSIS-589", "--ndjson")
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
	out, err := runList(t, "SSIS-589", "--type", "video")
	if err != nil {
		t.Fatalf("execute error = %v", err)
	}
	if out != "video\thttps://m.example.test/preview.m3u8\n" {
		t.Fatalf("video filter output = %q", out)
	}

	out, err = runList(t, "SSIS-589", "--type", "image", "1-2")
	if err != nil {
		t.Fatalf("execute error = %v", err)
	}
	want := "image\thttps://m.example.test/thumb.jpg\nimage\thttps://m.example.test/cover.jpg\n"
	if out != want {
		t.Fatalf("image filter + selector output:\n got  = %q\n want = %q", out, want)
	}
}

func TestListRejectsInvalidType(t *testing.T) {
	_, err := runList(t, "SSIS-589", "--type", "audio")
	if err == nil || !strings.Contains(err.Error(), `invalid --type "audio"`) {
		t.Fatalf("error = %v, want invalid --type", err)
	}
}

func TestListSelectorForms(t *testing.T) {
	cases := []struct {
		selector []string
		want     string
	}{
		{[]string{"1"}, "image\thttps://m.example.test/thumb.jpg\n"},
		{[]string{"1-2"}, "image\thttps://m.example.test/thumb.jpg\nimage\thttps://m.example.test/cover.jpg\n"},
		{[]string{"1,3"}, "image\thttps://m.example.test/thumb.jpg\nimage\thttps://m.example.test/p1-large.jpg\n"},
		// 空格分隔的多个 selector 参数:1 3 5。
		{[]string{"1", "3", "5"}, "image\thttps://m.example.test/thumb.jpg\nimage\thttps://m.example.test/p1-large.jpg\nimage\thttps://m.example.test/p3-large.jpg\n"},
		{[]string{"3-4,6"}, "image\thttps://m.example.test/p1-large.jpg\nimage\thttps://m.example.test/p2-thumb.jpg\nvideo\thttps://m.example.test/preview.m3u8\n"},
	}
	for _, tc := range cases {
		out, err := runList(t, append([]string{"SSIS-589"}, tc.selector...)...)
		if err != nil {
			t.Fatalf("selector %v: execute error = %v", tc.selector, err)
		}
		if out != tc.want {
			t.Fatalf("selector %v output:\n got  = %q\n want = %q", tc.selector, out, tc.want)
		}
	}
}

func TestListRejectsInvalidAndOutOfRangeSelector(t *testing.T) {
	_, err := runList(t, "SSIS-589", "4-1")
	if err == nil || !strings.Contains(err.Error(), `invalid selector "4-1"`) {
		t.Fatalf("error = %v, want invalid selector", err)
	}
	_, err = runList(t, "SSIS-589", "99")
	if err == nil || !strings.Contains(err.Error(), "out of range (1-6)") {
		t.Fatalf("error = %v, want out of range", err)
	}
}
