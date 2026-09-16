package assets

import (
	"bytes"
	"encoding/json"
	"errors"
	"image"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/pipeline"
	javdb "github.com/FlanChanXwO/javdb-cli/sdk"
)

// assets list 契约：资产获取 → --type 过滤 → selector → probe → 输出。
// TTY 和机器输出可包含 best-effort metadata；plain pipe 始终只有 TYPE<TAB>URL。

// listDetailFixture 的 <SERVER> 在 server 建立后替换为真实地址,
// 以便 list 的输出能直接喂给 download 测试(真管道)。
const listDetailFixture = `{"thumb_url":"<SERVER>/thumb.jpg","cover_url":"<SERVER>/cover.jpg","preview_video_url":"<SERVER>/preview.m3u8","preview_images":[
	{"large_url":"<SERVER>/p1-large.jpg","thumb_url":"<SERVER>/p1-thumb.jpg"},
	{"thumb_url":"<SERVER>/p2-thumb.jpg"},
	{"large_url":"<SERVER>/p3-large.jpg"}]}`

type listRequestLog struct {
	mu    sync.Mutex
	paths map[string]int
}

func (log *listRequestLog) record(path string) {
	log.mu.Lock()
	defer log.mu.Unlock()
	log.paths[path]++
}

func (log *listRequestLog) mediaPaths() map[string]int {
	log.mu.Lock()
	defer log.mu.Unlock()
	paths := make(map[string]int)
	for path, count := range log.paths {
		if !strings.HasPrefix(path, "/api/") {
			paths[path] = count
		}
	}
	return paths
}

func newListServer(t *testing.T) *httptest.Server {
	t.Helper()
	server, _ := newTrackedListServer(t)
	return server
}

func newTrackedListServer(t *testing.T) (*httptest.Server, *listRequestLog) {
	t.Helper()
	serverURL := ""
	var imageBody bytes.Buffer
	if err := jpeg.Encode(&imageBody, image.NewRGBA(image.Rect(0, 0, 13, 7)), nil); err != nil {
		t.Fatalf("encode list JPEG: %v", err)
	}
	encodedImage := imageBody.Bytes()
	requests := &listRequestLog{paths: make(map[string]int)}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests.record(request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.URL.Path == "/api/v2/search":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"movies":[{"number":"SSIS-589","id":"m1"}]}}`))
		case request.URL.Path == "/api/v4/movies/m1":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"movie":` + strings.ReplaceAll(listDetailFixture, "<SERVER>", serverURL) + `}}`))
		case strings.HasSuffix(request.URL.Path, ".m3u8"):
			_, _ = writer.Write([]byte("#EXTM3U\n#EXT-X-TARGETDURATION:2\n#EXTINF:1.0,\nseg.ts\n#EXT-X-ENDLIST\n"))
		case strings.HasSuffix(request.URL.Path, ".ts"):
			_, _ = writer.Write(validTSSegmentFixture())
		case strings.HasSuffix(request.URL.Path, ".jpg"):
			if request.URL.Path == "/p2-thumb.jpg" {
				_, _ = writer.Write(testJPEG)
			} else {
				_, _ = writer.Write(encodedImage)
			}
		default:
			http.NotFound(writer, request)
		}
	}))
	serverURL = server.URL
	return server, requests
}

func runList(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	return runListWithStreams(t, false, args...)
}

func runListWithStreams(t *testing.T, outIsTerminal bool, args ...string) (string, string, error) {
	t.Helper()
	out, serverURL, err, _ := runListConfigured(t, outIsTerminal, "", args...)
	return out, serverURL, err
}

func runListConfigured(t *testing.T, outIsTerminal bool, config string, args ...string) (string, string, error, *listRequestLog) {
	t.Helper()
	out, _, serverURL, err, requests := executeListConfigured(t, outIsTerminal, config, args...)
	return out, serverURL, err, requests
}

func executeListConfigured(t *testing.T, outIsTerminal bool, config string, args ...string) (string, string, string, error, *listRequestLog) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOMEDRIVE", filepath.VolumeName(t.TempDir()))
	t.Setenv("HOMEPATH", strings.TrimPrefix(t.TempDir(), filepath.VolumeName(t.TempDir())))
	if config != "" {
		configDir := filepath.Join(home, ".javdb-cli")
		if err := os.MkdirAll(configDir, 0o700); err != nil {
			t.Fatalf("create config dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(config), 0o600); err != nil {
			t.Fatalf("write config: %v", err)
		}
	}
	server, requests := newTrackedListServer(t)
	defer server.Close()

	streams := invocation.NewStreams(strings.NewReader(""), &strings.Builder{}, &strings.Builder{})
	streams.OutIsTerminal = outIsTerminal
	cmd := NewList(&invocation.RootOptions{Host: server.URL}, streams)
	cmd.SetArgs(args)
	err := cmd.Execute()
	out := streams.Out.(*strings.Builder).String()
	errOut := streams.Err.(*strings.Builder).String()
	return out, errOut, server.URL, err, requests
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
		"#  TYPE   SIZE       DURATION  DESCRIPTION",
		"1  image  13x7       -         thumbnail",
		"2  image  13x7       -         cover",
		"3  image  13x7       -         preview 1",
		"4  image  -          -         preview 2",
		"5  image  13x7       -         preview 3",
		"6  video  640x480    1s        preview",
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

func TestListJSONOutputAddsProbedMetadata(t *testing.T) {
	out, _, err := runList(t, "SSIS-589", "--type", "video", "--json")
	if err != nil {
		t.Fatalf("execute error = %v", err)
	}
	var assets []map[string]any
	if err := json.Unmarshal([]byte(out), &assets); err != nil {
		t.Fatalf("json decode: %v (out=%q)", err, out)
	}
	if len(assets) != 1 {
		t.Fatalf("asset count = %d, want 1", len(assets))
	}
	got := assets[0]
	if got["type"] != "video" || !strings.HasSuffix(got["url"].(string), "/preview.m3u8") ||
		got["width"] != float64(640) || got["height"] != float64(480) || got["duration"] != float64(1) {
		t.Fatalf("video asset = %v, want flattened probed metadata", got)
	}
}

func TestListNDJSONOutputAddsProbedMetadata(t *testing.T) {
	out, _, err := runList(t, "SSIS-589", "--type", "video", "--ndjson")
	if err != nil {
		t.Fatalf("execute error = %v", err)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 1 {
		t.Fatalf("line count = %d, want 1 (out=%q)", len(lines), out)
	}
	var first map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("ndjson decode: %v", err)
	}
	if first["type"] != "video" || first["width"] != float64(640) ||
		first["height"] != float64(480) || first["duration"] != float64(1) {
		t.Fatalf("first ndjson line = %v", first)
	}
}

func TestListPipeProbesOnlySelectedAssets(t *testing.T) {
	out, _, err, requests := runListConfigured(t, false, "", "SSIS-589", "--type", "image", "3")
	if err != nil {
		t.Fatalf("execute error = %v", err)
	}
	if !strings.HasPrefix(out, "image\t") || !strings.HasSuffix(strings.TrimSpace(out), "/p1-large.jpg") {
		t.Fatalf("pipe output = %q, want selected TYPE<TAB>URL", out)
	}
	if got := requests.mediaPaths(); len(got) != 1 || got["/p1-large.jpg"] != 1 {
		t.Fatalf("media requests = %v, want only selected asset", got)
	}
}

func TestListDisabledProbeSkipsMediaRequests(t *testing.T) {
	config := "[assets.probe]\nenabled = false\nconcurrency = 2\n"
	out, _, err, requests := runListConfigured(t, false, config, "SSIS-589", "--type", "video", "--json")
	if err != nil {
		t.Fatalf("execute error = %v", err)
	}
	var items []map[string]any
	if err := json.Unmarshal([]byte(out), &items); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	if len(items) != 1 || len(items[0]) != 2 || items[0]["type"] != "video" {
		t.Fatalf("disabled probe output = %v, want type/url only", items)
	}
	if got := requests.mediaPaths(); len(got) != 0 {
		t.Fatalf("media requests = %v, want none when disabled", got)
	}
}

func TestListProbeFailureKeepsAssetWithoutMetadata(t *testing.T) {
	out, errOut, _, err, _ := executeListConfigured(t, false, "", "SSIS-589", "--type", "image", "4", "--json")
	if err != nil {
		t.Fatalf("execute error = %v", err)
	}
	if errOut != "" {
		t.Fatalf("stderr = %q, want no per-item probe warning", errOut)
	}
	var items []map[string]any
	if err := json.Unmarshal([]byte(out), &items); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	if len(items) != 1 || len(items[0]) != 2 || !strings.HasSuffix(items[0]["url"].(string), "/p2-thumb.jpg") {
		t.Fatalf("failed probe output = %v, want selected type/url only", items)
	}
}

func TestListRejectsInvalidProbeConfig(t *testing.T) {
	config := "[assets.probe]\nconcurrency = 0\n"
	_, _, err, requests := runListConfigured(t, false, config, "SSIS-589", "--json")
	if err == nil || !strings.Contains(err.Error(), "assets.probe.concurrency must be positive") {
		t.Fatalf("error = %v, want invalid concurrency", err)
	}
	if got := requests.mediaPaths(); len(got) != 0 {
		t.Fatalf("media requests = %v, want none after config error", got)
	}
}

func TestListHelpDescribesMetadataAndStablePipe(t *testing.T) {
	cmd := NewList(&invocation.RootOptions{}, invocation.NewStreams(strings.NewReader(""), &strings.Builder{}, &strings.Builder{}))
	if !strings.Contains(cmd.Long, "best-effort width/height/duration metadata") {
		t.Fatalf("Long = %q, want metadata contract", cmd.Long)
	}
	if !strings.Contains(cmd.Long, "Pipe output remains TYPE<TAB>URL") {
		t.Fatalf("Long = %q, want stable pipe contract", cmd.Long)
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

var errListWriter = errors.New("list writer failed")

type listErrorWriter struct{}

func (listErrorWriter) Write([]byte) (int, error) { return 0, errListWriter }

func TestListHumanOutputPropagatesWriterError(t *testing.T) {
	err := renderAssetList(
		listErrorWriter{},
		pipeline.OutputHuman,
		[]javdb.MovieAssetInfo{{Asset: javdb.MovieAsset{Type: "image", URL: "https://media.example.test/image.jpg"}}},
		[]string{"thumbnail"},
	)
	if !errors.Is(err, errListWriter) {
		t.Fatalf("renderAssetList error = %v, want %v", err, errListWriter)
	}
}
