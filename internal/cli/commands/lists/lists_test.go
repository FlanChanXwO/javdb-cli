package lists

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
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
