package lists

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/pipeline"
)

func TestNewRelatedNDJSONUsesAuthoritativeMovieIDAndFansOutLists(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	searchRequests := 0
	relatedRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/search":
			searchRequests++
			http.Error(w, "movie resolver must not be called", http.StatusInternalServerError)
		case "/api/v1/lists/related":
			relatedRequests++
			if got := r.URL.Query().Get("movie_id"); got != "movie-1" {
				http.Error(w, "unexpected movie id", http.StatusBadRequest)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"data": map[string]any{
					"lists": []map[string]any{
						{"id": "list-1", "name": "List One", "movies_count": 1},
						{"id": "list-2", "name": "", "movies_count": 2},
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	input := `{"schema":"javdb.pipeline/v1","kind":"movie","ref":"SSIS-001","id":"movie-1"}` + "\n"
	streams := invocation.NewStreams(strings.NewReader(input), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := NewRelated(&invocation.RootOptions{Host: server.URL}, streams)
	cmd.SetArgs([]string{"--ndjson"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("NDJSON execute: %v", err)
	}
	if searchRequests != 0 {
		t.Fatalf("movie resolver search requests = %d, want 0", searchRequests)
	}
	if relatedRequests != 1 {
		t.Fatalf("related requests = %d, want 1", relatedRequests)
	}

	lines := strings.Split(strings.TrimSpace(streams.Out.(*bytes.Buffer).String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("NDJSON lines = %d, want one per related list: %q", len(lines), streams.Out.(*bytes.Buffer).String())
	}
	want := []struct {
		id, ref string
	}{
		{id: "list-1", ref: "List One"},
		{id: "list-2", ref: "list-2"},
	}
	for index, line := range lines {
		envelope, err := pipeline.DecodeNDJSON(line)
		if err != nil {
			t.Fatalf("line %d: decode envelope: %v", index+1, err)
		}
		if envelope.Kind != pipeline.KindList || envelope.ID != want[index].id || envelope.Ref != want[index].ref {
			t.Fatalf("line %d: envelope = %#v, want kind=list id=%q ref=%q", index+1, envelope, want[index].id, want[index].ref)
		}
		list, ok := envelope.Data["list"].(map[string]any)
		if !ok || list["id"] != want[index].id {
			t.Fatalf("line %d: data.list = %#v, want original list", index+1, envelope.Data["list"])
		}
	}
}
