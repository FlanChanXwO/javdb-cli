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

func TestNewBuildsListsGroup(t *testing.T) {
	streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{}, streams)
	if cmd.Name() != "lists" {
		t.Fatalf("name=%q", cmd.Name())
	}
	for _, flag := range []string{"page", "limit", "sort-by", "json"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Fatalf("lists missing --%s", flag)
		}
	}
	got := map[string]bool{}
	for _, sub := range cmd.Commands() {
		got[sub.Name()] = true
	}
	for _, name := range []string{"show", "search", "related"} {
		if !got[name] {
			t.Fatalf("lists missing subcommand %q", name)
		}
	}
}

func TestNewTextAndHumanOutputModes(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/lists" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": map[string]any{
			"lists": []map[string]any{{"id": "list-1", "name": "My List", "movies_count": float64(2), "privacy": "public", "views_count": float64(5)}},
		}})
	}))
	defer server.Close()

	textStreams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	textCmd := New(&invocation.RootOptions{Host: server.URL}, textStreams)
	if err := textCmd.Execute(); err != nil {
		t.Fatalf("text execute: %v", err)
	}
	if got := textStreams.Out.(*bytes.Buffer).String(); got != "My List\n" {
		t.Fatalf("non-TTY output = %q, want stable ref", got)
	}

	humanStreams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	humanStreams.OutIsTerminal = true
	humanCmd := New(&invocation.RootOptions{Host: server.URL}, humanStreams)
	if err := humanCmd.Execute(); err != nil {
		t.Fatalf("human execute: %v", err)
	}
	if got := humanStreams.Out.(*bytes.Buffer).String(); got != "list-1\tMy List\t2\tpublic\t5\n" {
		t.Fatalf("TTY output = %q, want list table", got)
	}
}

func TestNewJSONOutputFetchesListsOnceAndPreservesShape(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/lists" {
			http.NotFound(w, r)
			return
		}
		requests++
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data": map[string]any{
				"lists": []map[string]any{{
					"id": "list-1", "name": "My List", "movies_count": 2,
				}},
				"current_page": "3",
			},
		})
	}))
	defer server.Close()

	streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{Host: server.URL}, streams)
	cmd.SetArgs([]string{"--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("JSON execute: %v", err)
	}
	if requests != 1 {
		t.Fatalf("MyLists requests = %d, want 1", requests)
	}

	var got map[string]any
	if err := json.Unmarshal(streams.Out.(*bytes.Buffer).Bytes(), &got); err != nil {
		t.Fatalf("JSON output = %q: %v", streams.Out.(*bytes.Buffer).String(), err)
	}
	if got["current_page"] != "3" {
		t.Fatalf("current_page = %#v, want %q", got["current_page"], "3")
	}
	items, ok := got["lists"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("lists = %#v, want one item", got["lists"])
	}
	item, ok := items[0].(map[string]any)
	if !ok || item["id"] != "list-1" || item["name"] != "My List" {
		t.Fatalf("list item = %#v", items[0])
	}
}

func TestNewSearchNDJSONFansOutLists(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/search" {
			http.NotFound(w, r)
			return
		}
		requests++
		var items []map[string]any
		switch r.URL.Query().Get("q") {
		case "alpha":
			items = []map[string]any{{"id": "list-1", "name": "Alpha List"}}
		case "beta":
			items = []map[string]any{{"id": "list-2", "name": ""}}
		default:
			http.Error(w, "unexpected query", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data":    map[string]any{"lists": items},
		})
	}))
	defer server.Close()

	streams := invocation.NewStreams(strings.NewReader("alpha\nbeta\n"), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := NewSearch(&invocation.RootOptions{Host: server.URL}, streams)
	cmd.SetArgs([]string{"--ndjson"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("NDJSON execute: %v", err)
	}
	if requests != 2 {
		t.Fatalf("search requests = %d, want one per query", requests)
	}

	lines := strings.Split(strings.TrimSpace(streams.Out.(*bytes.Buffer).String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("NDJSON lines = %d, want two list envelopes: %q", len(lines), streams.Out.(*bytes.Buffer).String())
	}
	want := []struct {
		id, ref string
	}{
		{id: "list-1", ref: "Alpha List"},
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

func TestWriteListRows(t *testing.T) {
	var out, errb bytes.Buffer
	if err := writeListRows(&out, &errb, []map[string]any{
		{"id": "L", "name": "合集", "movies_count": float64(2), "privacy": "public", "views_count": float64(5)},
	}); err != nil {
		t.Fatal(err)
	}
	want := "L\t合集\t2\tpublic\t5\n"
	if out.String() != want {
		t.Fatalf("got %q want %q", out.String(), want)
	}
}

func TestWriteListRowsEmpty(t *testing.T) {
	var out, errb bytes.Buffer
	if err := writeListRows(&out, &errb, nil); err != nil {
		t.Fatal(err)
	}
	if out.String() != "" || errb.String() != "(空列表)\n" {
		t.Fatalf("out=%q err=%q", out.String(), errb.String())
	}
}

func TestNewShowHelp(t *testing.T) {
	streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := NewShow(&invocation.RootOptions{}, streams)
	var out, errb bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errb)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("help error = %v", err)
	}
	if !strings.Contains(out.String(), "Show 合集 meta") {
		t.Fatalf("help: %s", out.String())
	}
}
