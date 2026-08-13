package detail

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
)

// TestDetailJSONLStdinBatch 从 stdin 消费 movie 信封批，逐项解析并输出详情
// 信封；不兼容 kind 原位错误。
func TestDetailJSONLStdinBatch(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch {
		case request.URL.Path == "/api/v2/search":
			number := request.URL.Query().Get("q")
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{
				"movies": []map[string]any{{"number": number, "id": "id-" + number}},
			}})
		case strings.HasPrefix(request.URL.Path, "/api/v4/movies/"):
			id := strings.TrimPrefix(request.URL.Path, "/api/v4/movies/")
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{
				"movie": map[string]any{"number": "SSIS-589", "id": id},
			}})
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	stdin := `{"schema":"javdb.pipeline/v1","kind":"movie","ref":"SSIS-589"}
{"schema":"javdb.pipeline/v1","kind":"actor","ref":"X"}
`
	streams := invocation.NewStreams(strings.NewReader(stdin), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{Host: server.URL}, streams)
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "1 of 2 items failed") {
		t.Fatalf("expected batch failure, got %v", err)
	}
	out := streams.Out.(*bytes.Buffer).String()
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 {
		t.Fatalf("lines = %d:\n%s", len(lines), out)
	}
	var first struct {
		Kind string `json:"kind"`
		Ref  string `json:"ref"`
		ID   string `json:"id"`
	}
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatal(err)
	}
	if first.Kind != "movie" || first.Ref != "SSIS-589" || first.ID != "id-SSIS-589" {
		t.Errorf("first envelope = %+v", first)
	}
	if !strings.Contains(lines[1], `"kind":"error"`) {
		t.Errorf("second line should be an error envelope: %s", lines[1])
	}
}
