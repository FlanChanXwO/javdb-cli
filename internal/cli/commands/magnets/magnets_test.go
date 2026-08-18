package magnets

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/authstore"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/storage/auth"
)

func TestNewHelpListsFlags(t *testing.T) {
	streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{}, streams)
	var out, errb bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errb)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("help error = %v", err)
	}
	for _, want := range []string{"--cnsub", "--hd", "--min-size", "--best", "--json", "--id"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("help missing %s", want)
		}
	}
	if strings.Contains(out.String(), "needs login") {
		t.Fatalf("magnets must not require login")
	}
}

func TestParseSizeMiB(t *testing.T) {
	n, err := ParseSizeMiB("4GB")
	if err != nil || n != 4096 {
		t.Fatalf("%d %v", n, err)
	}
	n, err = ParseSizeMiB("500MB")
	if err != nil || n != 500 {
		t.Fatalf("%d %v", n, err)
	}
	n, err = ParseSizeMiB("2000")
	if err != nil || n != 2000 {
		t.Fatalf("%d %v", n, err)
	}
}

func TestWriteMagnetsEmpty(t *testing.T) {
	var out, errb bytes.Buffer
	writeMagnets(&out, &errb, nil)
	if out.String() != "" || errb.String() != "(无磁力链)\n" {
		t.Fatalf("out=%q err=%q", out.String(), errb.String())
	}
}

// TestMagnetsNDJSONInputWithIDDirect NDJSON 信封携带 id 时直接按 ID 请求，
// 不做番号解析。
func TestMagnetsNDJSONInputWithIDDirect(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	t.Setenv("HOMEDRIVE", filepath.VolumeName(t.TempDir()))
	t.Setenv("HOMEPATH", strings.TrimPrefix(t.TempDir(), filepath.VolumeName(t.TempDir())))
	resolved := false
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/v2/search":
			resolved = true
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{"movies": []map[string]any{}}})
		case "/api/v4/movies/id-xyz":
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{"movie": map[string]any{"magnets_count": float64(2)}}})
		case "/api/v1/movies/id-xyz/magnets":
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

	// NDJSON 信封携带 id → 直接按 ID 请求，不调用 search。
	streams := invocation.NewStreams(
		strings.NewReader("{\"schema\":\"javdb.pipeline/v1\",\"kind\":\"movie\",\"ref\":\"SSIS-001\",\"id\":\"id-xyz\"}\n"),
		&bytes.Buffer{}, &bytes.Buffer{},
	)
	cmd := New(&invocation.RootOptions{Host: server.URL}, streams)
	cmd.SetArgs([]string{"--ndjson"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if resolved {
		t.Error("ResolveMovieID (search) must not be called when envelope carries id")
	}
	out := streams.Out.(*bytes.Buffer).String()
	var envelope map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope["kind"] != "magnet" {
		t.Errorf("kind = %v, want magnet", envelope["kind"])
	}
	if envelope["id"] != "id-xyz" {
		t.Errorf("id = %v, want id-xyz", envelope["id"])
	}
}

// TestMagnetsNDJSONInputOutputsMagnetEnvelope 验证 NDJSON 输入保持磁力信封输出，
// 避免名称误导为默认文本模式覆盖。
func TestMagnetsNDJSONInputOutputsMagnetEnvelope(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	t.Setenv("HOMEDRIVE", filepath.VolumeName(t.TempDir()))
	t.Setenv("HOMEPATH", strings.TrimPrefix(t.TempDir(), filepath.VolumeName(t.TempDir())))
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/v4/movies/id-1":
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{"movie": map[string]any{"magnets_count": float64(2)}}})
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

	streams := invocation.NewStreams(
		strings.NewReader("{\"schema\":\"javdb.pipeline/v1\",\"kind\":\"movie\",\"ref\":\"SSIS-001\",\"id\":\"id-1\"}\n"),
		&bytes.Buffer{}, &bytes.Buffer{},
	)
	cmd := New(&invocation.RootOptions{Host: server.URL}, streams)
	cmd.SetArgs([]string{"--ndjson"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	out := streams.Out.(*bytes.Buffer).String()
	var envelope map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope["kind"] != "magnet" {
		t.Errorf("kind = %v, want magnet", envelope["kind"])
	}
	data, _ := envelope["data"].(map[string]any)
	magnets, _ := data["magnets"].([]any)
	if len(magnets) != 2 {
		t.Errorf("magnets count = %d, want 2", len(magnets))
	}
}

// TestMagnetsPositionalTextModeOutputsMagnetURIs 覆盖纯文本位置参数的真实
// 非 TTY 路径，确保 stdout 是 magnet URI 而不是输入番号或人类表格行。
func TestMagnetsPositionalTextModeOutputsMagnetURIs(t *testing.T) {
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
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{"movie": map[string]any{"magnets_count": float64(2)}}})
		case "/api/v1/movies/id-1/magnets":
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{
				"magnets": []map[string]any{{"name": "HD", "hash": "AAA"}, {"name": "SD", "hash": "BBB"}},
			}})
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{Host: server.URL}, streams)
	cmd.SetArgs([]string{"SSIS-001"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if got := streams.Out.(*bytes.Buffer).String(); got != "magnet:?xt=urn:btih:AAA\nmagnet:?xt=urn:btih:BBB\n" {
		t.Fatalf("text output = %q, want magnet URIs", got)
	}
}

func TestMagnetsNonTTYFallsBackToAnonymousAfterTokenRejection(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOMEDRIVE", filepath.VolumeName(home))
	t.Setenv("HOMEPATH", strings.TrimPrefix(home, filepath.VolumeName(home)))
	fs, store, err := authstore.Open()
	if err != nil {
		t.Fatal(err)
	}
	store.Upsert(auth.Account{UserID: 1, Username: "u", Token: "expired-token"}, true)
	if err := fs.Commit(store); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("authorization") != "" {
			_ = json.NewEncoder(writer).Encode(map[string]any{
				"success": false, "action": "JWTVerificationError", "message": "expired",
			})
			return
		}
		switch request.URL.Path {
		case "/api/v2/search":
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{
				"movies": []map[string]any{{"number": "SSIS-001", "id": "id-1"}},
			}})
		case "/api/v4/movies/id-1":
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{
				"movie": map[string]any{"magnets_count": float64(1)},
			}})
		case "/api/v1/movies/id-1/magnets":
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{
				"magnets": []map[string]any{{"hash": "AAA"}},
			}})
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{Host: server.URL}, streams)
	cmd.SetArgs([]string{"SSIS-001"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if got := streams.Out.(*bytes.Buffer).String(); got != "magnet:?xt=urn:btih:AAA\n" {
		t.Fatalf("text output = %q, want anonymous magnet result", got)
	}
	if got := streams.Err.(*bytes.Buffer).String(); !strings.Contains(got, "匿名") {
		t.Fatalf("stderr = %q, want token fallback diagnostic", got)
	}
}

// TestMagnetsPartialFailureContinues NDJSON 批处理中部分项失败仍继续。
func TestMagnetsPartialFailureContinues(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	t.Setenv("HOMEDRIVE", filepath.VolumeName(t.TempDir()))
	t.Setenv("HOMEPATH", strings.TrimPrefix(t.TempDir(), filepath.VolumeName(t.TempDir())))
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/v4/movies/id-ok":
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{"movie": map[string]any{"magnets_count": float64(1)}}})
		case "/api/v1/movies/id-ok/magnets":
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{
				"magnets": []map[string]any{{"name": "HD", "hash": "AAA", "size": float64(4096)}},
			}})
		case "/api/v4/movies/id-fail":
			http.NotFound(writer, request)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	ndjsonInput := "{\"schema\":\"javdb.pipeline/v1\",\"kind\":\"movie\",\"ref\":\"OK\",\"id\":\"id-ok\"}\n" +
		"{\"schema\":\"javdb.pipeline/v1\",\"kind\":\"movie\",\"ref\":\"FAIL\",\"id\":\"id-fail\"}\n"
	streams := invocation.NewStreams(strings.NewReader(ndjsonInput), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{Host: server.URL}, streams)
	cmd.SetArgs([]string{"--ndjson"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "1 of 2 items failed") {
		t.Fatalf("expected 1-of-2 failure, got %v", err)
	}
	out := streams.Out.(*bytes.Buffer).String()
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 {
		t.Fatalf("lines = %d, want 2", len(lines))
	}
	var first, second map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(lines[1]), &second); err != nil {
		t.Fatal(err)
	}
	if first["kind"] != "magnet" {
		t.Errorf("line 0 kind = %v, want magnet", first["kind"])
	}
	if second["kind"] != "error" {
		t.Errorf("line 1 kind = %v, want error", second["kind"])
	}
}
