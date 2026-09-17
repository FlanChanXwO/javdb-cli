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

// mediaPaths 只返回非 /api/ 的请求，用于断言 probe 是否真的发起媒体请求。
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

// newListServer 返回列表/详情 API 与媒体资源，并记录全部请求路径。
func newListServer(t *testing.T) (*httptest.Server, *listRequestLog) {
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

// listTestOptions 描述一次 assets list 调用环境；测试 helper 只暴露实际用到
// 的两个维度，避免多层 wrapper。
type listTestOptions struct {
	TTY    bool
	Config string
}

// runListCase 在隔离 HOME 下执行一次 assets list，返回 stdout/stderr/错误与请求记录。
func runListCase(t *testing.T, options listTestOptions, args ...string) (string, string, error, *listRequestLog) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOMEDRIVE", filepath.VolumeName(home))
	t.Setenv("HOMEPATH", strings.TrimPrefix(home, filepath.VolumeName(home)))
	if options.Config != "" {
		configDir := filepath.Join(home, ".javdb-cli")
		if err := os.MkdirAll(configDir, 0o700); err != nil {
			t.Fatalf("create config dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(options.Config), 0o600); err != nil {
			t.Fatalf("write config: %v", err)
		}
	}
	server, requests := newListServer(t)
	defer server.Close()

	streams := invocation.NewStreams(strings.NewReader(""), &strings.Builder{}, &strings.Builder{})
	streams.OutIsTerminal = options.TTY
	cmd := NewList(&invocation.RootOptions{Host: server.URL}, streams)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return streams.Out.(*strings.Builder).String(), streams.Err.(*strings.Builder).String(), err, requests
}

func TestListPipeOutputIsTypeTabURL(t *testing.T) {
	out, _, err, _ := runListCase(t, listTestOptions{}, "SSIS-589")
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

// TestListTTYOutputRendersNumberedTable 只锁定用户可观察的列与取值，
// 不锁定空格 padding：表格布局不是稳定机器协议。
func TestListTTYOutputRendersNumberedTable(t *testing.T) {
	out, _, err, _ := runListCase(t, listTestOptions{TTY: true}, "SSIS-589")
	if err != nil {
		t.Fatalf("execute error = %v", err)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 7 {
		t.Fatalf("line count = %d, want 1 header + 6 assets (out=%q)", len(lines), out)
	}
	for _, column := range []string{"#", "TYPE", "SIZE", "DURATION", "DESCRIPTION"} {
		if !strings.Contains(lines[0], column) {
			t.Fatalf("header = %q, want %s column", lines[0], column)
		}
	}
	// 每行必须同时包含类型、尺寸、时长与描述。
	wantRows := [][4]string{
		{"image", "13x7", "-", "thumbnail"},
		{"image", "13x7", "-", "cover"},
		{"image", "13x7", "-", "preview 1"},
		{"image", "-", "-", "preview 2"},
		{"image", "13x7", "-", "preview 3"},
		{"video", "640x480", "1s", "preview"},
	}
	for index, want := range wantRows {
		for _, field := range want {
			if !strings.Contains(lines[index+1], field) {
				t.Fatalf("row %d = %q, want %q", index+1, lines[index+1], field)
			}
		}
	}
}

// TestListMachineOutputAddsProbedMetadata 合并 JSON/NDJSON 两条机器输出链路。
func TestListMachineOutputAddsProbedMetadata(t *testing.T) {
	for _, mode := range []string{"--json", "--ndjson"} {
		t.Run(mode[2:], func(t *testing.T) {
			out, _, err, _ := runListCase(t, listTestOptions{}, "SSIS-589", "--type", "video", mode)
			if err != nil {
				t.Fatalf("execute error = %v", err)
			}
			var item map[string]any
			if mode == "--json" {
				var items []map[string]any
				if err := json.Unmarshal([]byte(out), &items); err != nil {
					t.Fatalf("json decode: %v (out=%q)", err, out)
				}
				if len(items) != 1 {
					t.Fatalf("asset count = %d, want 1", len(items))
				}
				item = items[0]
			} else {
				lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
				if len(lines) != 1 {
					t.Fatalf("line count = %d, want 1 (out=%q)", len(lines), out)
				}
				if err := json.Unmarshal([]byte(lines[0]), &item); err != nil {
					t.Fatalf("ndjson decode: %v", err)
				}
			}
			if item["type"] != "video" || !strings.HasSuffix(item["url"].(string), "/preview.m3u8") ||
				item["width"] != float64(640) || item["height"] != float64(480) || item["duration"] != float64(1) {
				t.Fatalf("video asset = %v, want flattened probed metadata", item)
			}
		})
	}
}

func TestListPipeProbesOnlySelectedAssets(t *testing.T) {
	out, _, err, requests := runListCase(t, listTestOptions{}, "SSIS-589", "--type", "image", "3")
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
	out, _, err, requests := runListCase(t, listTestOptions{Config: config}, "SSIS-589", "--type", "video", "--json")
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
	out, errOut, err, _ := runListCase(t, listTestOptions{}, "SSIS-589", "--type", "image", "4", "--json")
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

func TestListTypeFilterRenumbersAfterFilter(t *testing.T) {
	out, _, err, _ := runListCase(t, listTestOptions{}, "SSIS-589", "--type", "video")
	if err != nil {
		t.Fatalf("execute error = %v", err)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 1 || !strings.HasPrefix(lines[0], "video\t") || !strings.HasSuffix(lines[0], "/preview.m3u8") {
		t.Fatalf("video filter output = %q", out)
	}

	out, _, err, _ = runListCase(t, listTestOptions{}, "SSIS-589", "--type", "image", "1-2")
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
	_, _, err, _ := runListCase(t, listTestOptions{}, "SSIS-589", "--type", "audio")
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
		out, _, err, _ := runListCase(t, listTestOptions{}, append([]string{"SSIS-589"}, tc.selector...)...)
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
	_, _, err, _ := runListCase(t, listTestOptions{}, "SSIS-589", "4-1")
	if err == nil || !strings.Contains(err.Error(), `invalid selector "4-1"`) {
		t.Fatalf("error = %v, want invalid selector", err)
	}
	// 越界 selector 不得发起任何媒体请求。
	_, _, err, requests := runListCase(t, listTestOptions{}, "SSIS-589", "99")
	if err == nil || !strings.Contains(err.Error(), "out of range (1-6)") {
		t.Fatalf("error = %v, want out of range", err)
	}
	if got := requests.mediaPaths(); len(got) != 0 {
		t.Fatalf("media requests = %v, want none after selector error", got)
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
