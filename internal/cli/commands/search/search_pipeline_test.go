package search

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	magnetscmd "github.com/FlanChanXwO/javdb-cli/internal/cli/commands/magnets"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/pipeline"
)

type failingWriter struct{ err error }

func (w failingWriter) Write([]byte) (int, error) { return 0, w.err }

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

// TestSearchPositionalNonTTYUsesStablePipelineRecord 验证单个位置参数在非 TTY
// stdout 下不再调用 Legacy 人类列表渲染，而是输出可供下游消费的番号记录。
func TestSearchPositionalNonTTYUsesStablePipelineRecord(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	t.Setenv("HOMEDRIVE", filepath.VolumeName(t.TempDir()))
	t.Setenv("HOMEPATH", strings.TrimPrefix(t.TempDir(), filepath.VolumeName(t.TempDir())))
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{
			"movies": []map[string]any{{"number": "SSIS-001", "id": "id-1", "title": "Mock"}},
		}})
	}))
	defer server.Close()

	streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{Host: server.URL}, streams)
	cmd.SetArgs([]string{"SSIS"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if got := streams.Out.(*bytes.Buffer).String(); got != "SSIS-001\n" {
		t.Fatalf("non-TTY positional output = %q, want stable ref record", got)
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

// TestSearchNamedFanOutStableRefs 验证命名实体搜索按实体逐项输出，非 TTY
// 文本与 NDJSON 的 ref 都来自实体本身，不回退为原始关键词。
func TestSearchNamedFanOutStableRefs(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	t.Setenv("HOMEDRIVE", filepath.VolumeName(t.TempDir()))
	t.Setenv("HOMEPATH", strings.TrimPrefix(t.TempDir(), filepath.VolumeName(t.TempDir())))
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{
			"actors": []map[string]any{
				{"id": "actor-1", "name": "Actor One"},
				{"id": "actor-2", "name_zht": "Actor Two"},
			},
		}})
	}))
	defer server.Close()

	textStreams := invocation.NewStreams(strings.NewReader("keyword\n"), &bytes.Buffer{}, &bytes.Buffer{})
	textCmd := New(&invocation.RootOptions{Host: server.URL}, textStreams)
	textCmd.SetArgs([]string{"--type", "actor"})
	if err := textCmd.Execute(); err != nil {
		t.Fatalf("text execute: %v", err)
	}
	if got := textStreams.Out.(*bytes.Buffer).String(); got != "Actor One\nActor Two\n" {
		t.Fatalf("named text output = %q", got)
	}

	ndjsonStreams := invocation.NewStreams(strings.NewReader("keyword\n"), &bytes.Buffer{}, &bytes.Buffer{})
	ndjsonCmd := New(&invocation.RootOptions{Host: server.URL}, ndjsonStreams)
	ndjsonCmd.SetArgs([]string{"--type", "actor", "--ndjson"})
	if err := ndjsonCmd.Execute(); err != nil {
		t.Fatalf("ndjson execute: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(ndjsonStreams.Out.(*bytes.Buffer).String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("named NDJSON lines = %d, want 2", len(lines))
	}
	for i, want := range []string{"Actor One", "Actor Two"} {
		var envelope map[string]any
		if err := json.Unmarshal([]byte(lines[i]), &envelope); err != nil {
			t.Fatal(err)
		}
		if envelope["kind"] != "actor" || envelope["ref"] != want {
			t.Errorf("line %d envelope = %v", i, envelope)
		}
	}
}

// TestSearchMagnetsPassesThroughErrorEnvelope 验证上游错误不触发搜索请求，
// 且 NDJSON 中保留原错误信封内容。
func TestSearchMagnetsPassesThroughErrorEnvelope(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	t.Setenv("HOMEDRIVE", filepath.VolumeName(t.TempDir()))
	t.Setenv("HOMEPATH", strings.TrimPrefix(t.TempDir(), filepath.VolumeName(t.TempDir())))
	var requests atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		http.NotFound(writer, request)
	}))
	defer server.Close()

	input := `{"schema":"javdb.pipeline/v1","kind":"error","ref":"ABC-123","data":{"command":"upstream","stage":"resolve","code":"missing","message":"not found"}}` + "\n"
	streams := invocation.NewStreams(strings.NewReader(input), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{Host: server.URL}, streams)
	cmd.SetArgs([]string{"--magnets", "0", "--ndjson"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "1 of 1") {
		t.Fatalf("error = %v, want propagated failure summary", err)
	}
	if requests.Load() != 0 {
		t.Fatalf("requests = %d, want 0", requests.Load())
	}
	wantEnvelope, err := pipeline.DecodeNDJSON(strings.TrimSpace(input))
	if err != nil {
		t.Fatal(err)
	}
	gotEnvelope, err := pipeline.DecodeNDJSON(strings.TrimSpace(streams.Out.(*bytes.Buffer).String()))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotEnvelope, wantEnvelope) {
		t.Fatalf("propagated envelope = %#v, want %#v", gotEnvelope, wantEnvelope)
	}

	wrongKindInput := `{"schema":"javdb.pipeline/v1","kind":"actor","ref":"Actor One","id":"actor-1"}` + "\n"
	wrongKindStreams := invocation.NewStreams(strings.NewReader(wrongKindInput), &bytes.Buffer{}, &bytes.Buffer{})
	wrongKindCmd := New(&invocation.RootOptions{Host: server.URL}, wrongKindStreams)
	wrongKindCmd.SetArgs([]string{"--magnets", "0", "--ndjson"})
	if err := wrongKindCmd.Execute(); err == nil || !strings.Contains(err.Error(), "1 of 1") {
		t.Fatalf("wrong-kind error = %v", err)
	}
	if requests.Load() != 0 {
		t.Fatalf("wrong-kind requests = %d, want 0", requests.Load())
	}
	wrongKindEnvelope, err := pipeline.DecodeNDJSON(strings.TrimSpace(wrongKindStreams.Out.(*bytes.Buffer).String()))
	if err != nil {
		t.Fatal(err)
	}
	if wrongKindEnvelope.Kind != pipeline.KindError || wrongKindEnvelope.Data["code"] != "kind" {
		t.Fatalf("wrong-kind envelope = %#v", wrongKindEnvelope)
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

func TestSearchMagnetsUsesMovieEnvelopeIDDirectly(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	var searchRequests atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/v2/search":
			searchRequests.Add(1)
			http.Error(writer, "search must not be called", http.StatusInternalServerError)
		case "/api/v1/movies/id-1/magnets":
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{
				"magnets": []map[string]any{{"hash": "AAA"}},
			}})
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	input := `{"schema":"javdb.pipeline/v1","kind":"movie","ref":"SSIS-001","id":"id-1","data":{"movie":{"number":"SSIS-001","id":"id-1","magnets_count":1}}}` + "\n"
	streams := invocation.NewStreams(strings.NewReader(input), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{Host: server.URL}, streams)
	cmd.SetArgs([]string{"--magnets", "1", "--ndjson"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if searchRequests.Load() != 0 {
		t.Fatalf("search requests = %d, want 0", searchRequests.Load())
	}
	var envelope map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(streams.Out.(*bytes.Buffer).String())), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope["kind"] != "magnet" || envelope["id"] != "id-1" {
		t.Fatalf("envelope = %v, want direct movie id", envelope)
	}
}

func TestSearchMagnetsBatchJSONEmitsSourceEnvelopes(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/v2/search":
			keyword := request.URL.Query().Get("q")
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{
				"movies": []map[string]any{{"number": strings.ToUpper(keyword), "id": "id-" + keyword}},
			}})
		case "/api/v1/movies/id-alpha/magnets", "/api/v1/movies/id-beta/magnets":
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{
				"magnets": []map[string]any{{"hash": "AAA"}},
			}})
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	streams := invocation.NewStreams(strings.NewReader("alpha\nbeta\n"), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{Host: server.URL}, streams)
	cmd.SetArgs([]string{"--magnets", "1", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	var envelopes []map[string]any
	if err := json.Unmarshal(streams.Out.(*bytes.Buffer).Bytes(), &envelopes); err != nil {
		t.Fatalf("batch JSON = %q: %v", streams.Out.(*bytes.Buffer).String(), err)
	}
	if len(envelopes) != 2 {
		t.Fatalf("envelope count = %d, want 2", len(envelopes))
	}
	for i, want := range []string{"alpha", "beta"} {
		if envelopes[i]["kind"] != "magnet" {
			t.Errorf("envelope %d kind = %v, want magnet", i, envelopes[i]["kind"])
		}
		meta, _ := envelopes[i]["meta"].(map[string]any)
		if meta["input_ref"] != want {
			t.Errorf("envelope %d input_ref = %v, want %s", i, meta["input_ref"], want)
		}
	}
}

// TestSearchMagnetsJSONPartialFailureReturnsError 验证 JSON 保留成功/失败明细，
// 但部分磁力请求失败时命令仍以非零错误结束。
func TestSearchMagnetsJSONPartialFailureReturnsError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	t.Setenv("HOMEDRIVE", filepath.VolumeName(t.TempDir()))
	t.Setenv("HOMEPATH", strings.TrimPrefix(t.TempDir(), filepath.VolumeName(t.TempDir())))
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/v2/search":
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{
				"movies": []map[string]any{
					{"number": "SSIS-001", "id": "id-ok"},
					{"number": "SSIS-002", "id": "id-fail"},
				},
			}})
		case "/api/v1/movies/id-ok/magnets":
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{
				"magnets": []map[string]any{{"name": "HD", "hash": "AAA"}},
			}})
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	streams := invocation.NewStreams(strings.NewReader("SSIS\n"), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{Host: server.URL}, streams)
	cmd.SetArgs([]string{"--magnets", "0", "--json"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "1 of 2 movies failed") {
		t.Fatalf("error = %v, output = %q, want partial-failure summary", err, streams.Out.(*bytes.Buffer).String())
	}
	var output map[string]any
	if err := json.Unmarshal(streams.Out.(*bytes.Buffer).Bytes(), &output); err != nil {
		t.Fatalf("JSON output invalid: %v", err)
	}
	movies, _ := output["movies"].([]any)
	if len(movies) != 2 {
		t.Fatalf("movies = %d, want 2", len(movies))
	}
	failed, _ := movies[1].(map[string]any)
	if failed["error"] == nil || failed["error"] == "" {
		t.Fatalf("failed movie missing error detail: %v", failed)
	}
}

func TestSearchMagnetsWorkerContinuesAfterAllInitialFailures(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/v2/search":
			movies := make([]map[string]any, 0, 5)
			for i := 1; i <= 5; i++ {
				movies = append(movies, map[string]any{
					"number": fmt.Sprintf("SSIS-%03d", i),
					"id":     fmt.Sprintf("id-%d", i),
				})
			}
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{"movies": movies}})
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	streams := invocation.NewStreams(strings.NewReader("SSIS\n"), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{Host: server.URL}, streams)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--magnets", "0", "--json"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "5 of 5 movies failed") {
		t.Fatalf("error = %v, want complete five-item failure summary", err)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("magnet worker pool timed out after workers failed")
	}
	var payload map[string]any
	if err := json.Unmarshal(streams.Out.(*bytes.Buffer).Bytes(), &payload); err != nil {
		t.Fatalf("JSON output invalid: %v", err)
	}
	movies, _ := payload["movies"].([]any)
	if len(movies) != 5 {
		t.Fatalf("movies = %d, want 5 partial results", len(movies))
	}
}

// TestSearchMagnetsTextPropagatesWriteErrors 验证磁力文本路径向上返回 stdout
// 和 stderr 的写入错误，避免下游关闭 pipe 后仍报告正常成功。
func TestSearchMagnetsTextPropagatesWriteErrors(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	t.Setenv("HOMEDRIVE", filepath.VolumeName(t.TempDir()))
	t.Setenv("HOMEPATH", strings.TrimPrefix(t.TempDir(), filepath.VolumeName(t.TempDir())))
	for _, tc := range []struct {
		name       string
		magnets404 bool
		out        failingWriter
		errW       failingWriter
		want       string
	}{
		{name: "stdout", out: failingWriter{err: errors.New("stdout closed")}, want: "stdout closed"},
		{name: "stderr", magnets404: true, errW: failingWriter{err: errors.New("stderr closed")}, want: "stderr closed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				switch request.URL.Path {
				case "/api/v2/search":
					_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{
						"movies": []map[string]any{{"number": "SSIS-001", "id": "id-1"}},
					}})
				case "/api/v1/movies/id-1/magnets":
					if tc.magnets404 {
						http.NotFound(writer, request)
						return
					}
					_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{
						"magnets": []map[string]any{{"hash": "AAA"}},
					}})
				default:
					http.NotFound(writer, request)
				}
			}))
			defer server.Close()

			var out, errOut bytes.Buffer
			outWriter := &out
			if tc.out.err != nil {
				outWriter = nil
			}
			errWriter := &errOut
			if tc.errW.err != nil {
				errWriter = nil
			}
			streams := invocation.NewStreams(strings.NewReader("SSIS\n"), outWriter, errWriter)
			if tc.out.err != nil {
				streams.Out = tc.out
			}
			if tc.errW.err != nil {
				streams.Err = tc.errW
			}
			cmd := New(&invocation.RootOptions{Host: server.URL}, streams)
			cmd.SetArgs([]string{"--magnets", "0"})
			err := cmd.Execute()
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
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

// TestSearchTextPipelineToMagnets 验证默认非 TTY 文本链路只传递稳定番号，且
// magnets 最终只投影磁力 URI；这是用户实际 `search SSIS | magnets` 的路径。
func TestSearchTextPipelineToMagnets(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	t.Setenv("HOMEDRIVE", filepath.VolumeName(t.TempDir()))
	t.Setenv("HOMEPATH", strings.TrimPrefix(t.TempDir(), filepath.VolumeName(t.TempDir())))
	searchQueries := make([]string, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/v2/search":
			searchQueries = append(searchQueries, request.URL.Query().Get("q"))
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{
				"movies": []map[string]any{{"number": "SSIS-001", "id": "id-1", "title": "Mock"}},
			}})
		case "/api/v4/movies/id-1":
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{
				"movie": map[string]any{"magnets_count": float64(1)},
			}})
		case "/api/v1/movies/id-1/magnets":
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{
				"magnets": []map[string]any{{"name": "HD", "hash": "AAA", "size": float64(4096), "hd": true}},
			}})
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	// Step 1: 单个位置参数且非 TTY 时仅产生稳定番号，不包含标题或表格列。
	searchStreams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	searchCmd := New(&invocation.RootOptions{Host: server.URL}, searchStreams)
	searchCmd.SetArgs([]string{"SSIS"})
	if err := searchCmd.Execute(); err != nil {
		t.Fatalf("search: %v", err)
	}
	searchOutput := searchStreams.Out.(*bytes.Buffer).String()
	if searchOutput != "SSIS-001\n" {
		t.Fatalf("search output = %q, want only stable movie record", searchOutput)
	}
	if got := searchStreams.Err.(*bytes.Buffer).String(); got != "" {
		t.Fatalf("search stderr = %q, want empty", got)
	}

	// Step 2: 默认文本 magnets 从上一步 stdout 读取番号，并仅输出磁力 URI。
	magnetsStreams := invocation.NewStreams(strings.NewReader(searchOutput), &bytes.Buffer{}, &bytes.Buffer{})
	magnetsCmd := magnetscmd.New(&invocation.RootOptions{Host: server.URL}, magnetsStreams)
	if err := magnetsCmd.Execute(); err != nil {
		t.Fatalf("magnets: %v", err)
	}
	if got := magnetsStreams.Out.(*bytes.Buffer).String(); got != "magnet:?xt=urn:btih:AAA\n" {
		t.Fatalf("magnets output = %q, want only magnet URI", got)
	}
	if got := magnetsStreams.Err.(*bytes.Buffer).String(); got != "" {
		t.Fatalf("magnets stderr = %q, want empty", got)
	}
	if got, want := strings.Join(searchQueries, ","), "SSIS,SSIS-001"; got != want {
		t.Fatalf("search queries = %q, want %q", got, want)
	}
}
