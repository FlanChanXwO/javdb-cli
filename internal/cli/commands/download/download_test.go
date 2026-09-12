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
	"github.com/FlanChanXwO/javdb-cli/internal/cli/pipeline"
)

func TestNewHelpDescribesLocalMovieAssets(t *testing.T) {
	streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{}, streams)
	var out, errb bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errb)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("help error = %v", err)
	}
	for _, want := range []string{
		"--thumbnail",
		"--preview-image",
		"--preview-video",
		"only the first preview image",
		"thumbnail and preview assets",
		"does not download full movies",
		"magnets",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("download help missing %q: %s", want, out.String())
		}
	}
}

func TestNewUsesAssetsAsCanonicalCommand(t *testing.T) {
	streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{}, streams)

	if got, want := cmd.Name(), "assets"; got != want {
		t.Fatalf("canonical command name = %q, want %q", got, want)
	}
	if got, want := cmd.Use, "assets NUMBER"; got != want {
		t.Fatalf("canonical command use = %q, want %q", got, want)
	}
	if !cmd.HasAlias("download") {
		t.Fatal("canonical assets command is missing download alias")
	}
	for _, name := range []string{"id", "thumbnail", "preview-image", "preview-video", "json", "ndjson"} {
		if cmd.LocalNonPersistentFlags().Lookup(name) == nil {
			t.Fatalf("canonical assets command missing flag %q", name)
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
	// 改用同 number 不同 id 输入避免重复：用 NDJSON 提供唯一 id。
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
// 完整走通下载链路：detail 返回本地 mock 媒体，断言请求 ID 正确、成功结果
// 与输出文件内容。
func TestDownloadPipelineIDDoesNotResolveAsNumber(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	var searchCalls int
	var detailIDs []string
	media := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		// 媒体下载要求响应是已识别图片（magic）或 XOR 编码图片。
		_, _ = writer.Write([]byte{0xFF, 0xD8, 0xFF, 0xE0, 0x01, 0x02, 0x03})
	}))
	defer media.Close()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch {
		case request.URL.Path == "/api/v2/search":
			searchCalls++
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{
				"movies": []map[string]any{{"number": "SSIS-589", "id": "WRONG-ID"}},
			}})
		case strings.HasPrefix(request.URL.Path, "/api/v4/movies/"):
			id := strings.TrimPrefix(request.URL.Path, "/api/v4/movies/")
			detailIDs = append(detailIDs, id)
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{
				"movie": map[string]any{"number": "SSIS-589", "id": id, "thumb_url": media.URL},
			}})
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	dir := t.TempDir()
	target := filepath.Join(dir, "out.jpg")
	streams := invocation.NewStreams(strings.NewReader("CORRECT-ID\n"), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{Host: server.URL}, streams)
	cmd.SetArgs([]string{"--id", "--thumbnail", target})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if searchCalls != 0 {
		t.Fatalf("--id pipeline path called ResolveMovieID %d time(s)", searchCalls)
	}
	if len(detailIDs) != 1 || detailIDs[0] != "CORRECT-ID" {
		t.Fatalf("detail requested for ids %v, want [CORRECT-ID]", detailIDs)
	}
	body, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("output file missing: %v", err)
	}
	if !bytes.Equal(body, []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x01, 0x02, 0x03}) {
		t.Fatalf("output content = %x", body)
	}
}

func TestDownloadPipelineWritesSelectedAssetsAndNDJSON(t *testing.T) {
	isolateDownloadTestHome(t)
	var detailIDs []string
	server, fixtures := newDownloadAssetServer(t, &detailIDs, nil)
	defer server.Close()

	dir := t.TempDir()
	thumbnailPath := filepath.Join(dir, "{number}-{id}.jpg")
	previewImagePath := filepath.Join(dir, "{number}-{id}.png")
	previewVideoPath := filepath.Join(dir, "{number}-{id}.ts")
	streams := invocation.NewStreams(strings.NewReader("{\"schema\":\"javdb.pipeline/v1\",\"kind\":\"movie\",\"ref\":\"SSIS-589\",\"id\":\"movie-id\"}\n"), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{Host: server.URL}, streams)
	cmd.SetArgs([]string{
		"--thumbnail", thumbnailPath,
		"--preview-image", previewImagePath,
		"--preview-video", previewVideoPath,
		"--ndjson",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(detailIDs) != 1 || detailIDs[0] != "movie-id" {
		t.Fatalf("detail requested for ids %v, want [movie-id]", detailIDs)
	}

	var output pipeline.Envelope
	if err := json.Unmarshal([]byte(strings.TrimSpace(streams.Out.(*bytes.Buffer).String())), &output); err != nil {
		t.Fatalf("decode NDJSON output: %v; output=%q", err, streams.Out.(*bytes.Buffer).String())
	}
	if output.Schema != pipeline.Schema || output.Kind != pipeline.KindDownload || output.Ref != "SSIS-589" || output.ID != "movie-id" {
		t.Fatalf("output envelope = %+v", output)
	}
	expandedThumbnail := filepath.Join(dir, "SSIS-589-movie-id.jpg")
	expandedPreviewImage := filepath.Join(dir, "SSIS-589-movie-id.png")
	expandedPreviewVideo := filepath.Join(dir, "SSIS-589-movie-id.ts")
	for _, asset := range []struct {
		name string
		path string
		want []byte
	}{
		{name: "thumbnail", path: expandedThumbnail, want: fixtures.thumbnail},
		{name: "preview image", path: expandedPreviewImage, want: fixtures.previewImage},
		{name: "preview video", path: expandedPreviewVideo, want: fixtures.previewVideo},
	} {
		got, err := os.ReadFile(asset.path)
		if err != nil {
			t.Errorf("read %s: %v", asset.name, err)
		} else if !bytes.Equal(got, asset.want) {
			t.Errorf("%s content = %x, want %x", asset.name, got, asset.want)
		}
	}
	for key, want := range map[string]string{
		"thumbnail":     expandedThumbnail,
		"preview_image": expandedPreviewImage,
		"preview_video": expandedPreviewVideo,
	} {
		if got, ok := output.Data[key].(string); !ok || got != want {
			t.Errorf("output data[%q] = %#v, want %q", key, output.Data[key], want)
		}
	}
}

func TestDownloadPositionalIDWritesJSONEnvelope(t *testing.T) {
	isolateDownloadTestHome(t)
	var detailIDs []string
	var searchCalls int
	server, fixtures := newDownloadAssetServer(t, &detailIDs, &searchCalls)
	defer server.Close()

	target := filepath.Join(t.TempDir(), "positional.jpg")
	streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	streams.InIsTerminal = true
	cmd := New(&invocation.RootOptions{Host: server.URL}, streams)
	cmd.SetArgs([]string{"movie-id", "--id", "--thumbnail", target, "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if searchCalls != 0 {
		t.Fatalf("positional --id called ResolveMovieID %d time(s)", searchCalls)
	}
	if len(detailIDs) != 1 || detailIDs[0] != "movie-id" {
		t.Fatalf("detail requested for ids %v, want [movie-id]", detailIDs)
	}
	var output pipeline.Envelope
	if err := json.Unmarshal(streams.Out.(*bytes.Buffer).Bytes(), &output); err != nil {
		t.Fatalf("decode JSON output: %v; output=%q", err, streams.Out.(*bytes.Buffer).String())
	}
	if output.Schema != pipeline.Schema || output.Kind != pipeline.KindDownload || output.Ref != "movie-id" || output.ID != "movie-id" {
		t.Fatalf("output envelope = %+v", output)
	}
	if got, err := os.ReadFile(target); err != nil {
		t.Fatalf("read output: %v", err)
	} else if !bytes.Equal(got, fixtures.thumbnail) {
		t.Fatalf("thumbnail content = %x, want %x", got, fixtures.thumbnail)
	}
}

func TestDownloadBatchRejectsMissingParentDirectory(t *testing.T) {
	isolateDownloadTestHome(t)
	var detailIDs []string
	server, _ := newDownloadAssetServer(t, &detailIDs, nil)
	defer server.Close()

	root := t.TempDir()
	target := filepath.Join(root, "missing", "{id}.jpg")
	stdin := "{\"schema\":\"javdb.pipeline/v1\",\"kind\":\"movie\",\"ref\":\"SSIS-589\",\"id\":\"movie-1\"}\n" +
		"{\"schema\":\"javdb.pipeline/v1\",\"kind\":\"movie\",\"ref\":\"HZGD-246\",\"id\":\"movie-2\"}\n"
	streams := invocation.NewStreams(strings.NewReader(stdin), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{Host: server.URL}, streams)
	cmd.SetArgs([]string{"--thumbnail", target, "--ndjson"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "parent directory does not exist") {
		t.Fatalf("expected missing-parent error, got %v", err)
	}
	if len(detailIDs) != 0 {
		t.Fatalf("preflight unexpectedly downloaded details for ids %v", detailIDs)
	}
}

type downloadAssetFixtures struct {
	thumbnail    []byte
	previewImage []byte
	previewVideo []byte
}

func newDownloadAssetServer(t *testing.T, detailIDs *[]string, searchCalls *int) (*httptest.Server, downloadAssetFixtures) {
	t.Helper()
	fixtures := downloadAssetFixtures{
		thumbnail:    []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x01},
		previewImage: []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0x00},
		previewVideo: []byte("first preview segment\nsecond preview segment\n"),
	}
	firstSegment := []byte("first preview segment\n")
	secondSegment := []byte("second preview segment\n")
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writeJSON := func(value any) {
			writer.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(writer).Encode(value)
		}
		switch {
		case request.URL.Path == "/api/v2/search":
			if searchCalls != nil {
				(*searchCalls)++
			}
			http.NotFound(writer, request)
		case strings.HasPrefix(request.URL.Path, "/api/v4/movies/"):
			id := strings.TrimPrefix(request.URL.Path, "/api/v4/movies/")
			if detailIDs != nil {
				*detailIDs = append(*detailIDs, id)
			}
			writeJSON(map[string]any{"success": true, "data": map[string]any{
				"movie": map[string]any{
					"id":                id,
					"thumb_url":         server.URL + "/media/thumbnail.jpg",
					"preview_images":    []map[string]any{{"large_url": server.URL + "/media/preview-first.png"}, {"large_url": server.URL + "/media/preview-second.png"}},
					"preview_video_url": server.URL + "/media/preview/index.m3u8",
				},
			}})
		case request.URL.Path == "/media/thumbnail.jpg":
			_, _ = writer.Write(fixtures.thumbnail)
		case request.URL.Path == "/media/preview-first.png":
			_, _ = writer.Write(fixtures.previewImage)
		case request.URL.Path == "/media/preview-second.png":
			_, _ = writer.Write([]byte("unexpected second preview"))
		case request.URL.Path == "/media/preview/index.m3u8":
			_, _ = writer.Write([]byte("#EXTM3U\n#EXTINF:1.0,\npart-1.ts\n#EXTINF:1.0,\npart-2.ts\n#EXT-X-ENDLIST\n"))
		case request.URL.Path == "/media/preview/part-1.ts":
			_, _ = writer.Write(firstSegment)
		case request.URL.Path == "/media/preview/part-2.ts":
			_, _ = writer.Write(secondSegment)
		default:
			http.NotFound(writer, request)
		}
	}))
	return server, fixtures
}

func isolateDownloadTestHome(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOMEDRIVE", filepath.VolumeName(home))
	t.Setenv("HOMEPATH", strings.TrimPrefix(home, filepath.VolumeName(home)))
	t.Setenv("HTTP_PROXY", "")
	t.Setenv("HTTPS_PROXY", "")
	t.Setenv("ALL_PROXY", "")
}
