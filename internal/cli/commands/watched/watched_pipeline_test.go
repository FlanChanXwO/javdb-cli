package watched

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
)

// TestWatchedProducerJSONL 非 TTY producer：不消费 stdin，输出逐条 movie
// 信封；TTY 保持人类文本行。
func TestWatchedProducerJSONL(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{
			"movies": []map[string]any{
				{"number": "SSIS-589", "id": "id-a"},
				{"number": "HZGD-246", "id": "id-b"},
			},
		}})
	}))
	defer server.Close()

	streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{Host: server.URL}, streams)
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	out := streams.Out.(*bytes.Buffer).String()
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 {
		t.Fatalf("jsonl lines = %d:\n%s", len(lines), out)
	}
	var first struct {
		Kind string `json:"kind"`
		Ref  string `json:"ref"`
		ID   string `json:"id"`
	}
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatal(err)
	}
	if first.Kind != "movie" || first.Ref != "SSIS-589" || first.ID != "id-a" {
		t.Errorf("first envelope = %+v", first)
	}
}
