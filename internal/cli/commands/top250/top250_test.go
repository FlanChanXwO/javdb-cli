package top250

import (
	"bytes"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"strings"
	"testing"
)

func TestNewBuildsTop250Command(t *testing.T) {
	streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{}, streams)
	if cmd.Name() != "top250" || cmd.Use != "top250" {
		t.Fatalf("name=%q use=%q", cmd.Name(), cmd.Use)
	}
	for _, flag := range []string{"zone", "year", "from", "page", "limit", "ignore-watched", "has-magnets", "json"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Fatalf("missing --%s", flag)
		}
	}
}

func TestNewHelp(t *testing.T) {
	streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{}, streams)
	var out, errb bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errb)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("help error = %v", err)
	}
	if !strings.Contains(out.String(), "TOP250 list (needs login)") {
		t.Fatalf("help: %s", out.String())
	}
}

func TestWriteRankedPrefixesRank(t *testing.T) {
	var out, errb bytes.Buffer
	if err := writeRanked(&out, &errb, []map[string]any{
		{"ranking": float64(3), "number": "N3", "id": "i3", "title": "T3", "release_date": "2026-03-04"},
		{"ranking": float64(4), "number": "N4", "id": "i4", "title": "T4"},
	}); err != nil {
		t.Fatal(err)
	}
	want := "#3\tN3\ti3\tT3\t2026-03-04\n#4\tN4\ti4\tT4\n"
	if out.String() != want {
		t.Fatalf("got %q want %q", out.String(), want)
	}
}

func TestWriteRankedEmpty(t *testing.T) {
	var out, errb bytes.Buffer
	if err := writeRanked(&out, &errb, nil); err != nil {
		t.Fatal(err)
	}
	if out.String() != "" || errb.String() != "(空列表)\n" {
		t.Fatalf("out=%q err=%q", out.String(), errb.String())
	}
}
