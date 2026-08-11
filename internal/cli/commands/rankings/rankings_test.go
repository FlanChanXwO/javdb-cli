package rankings

import (
	"bytes"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"strings"
	"testing"
)

func TestNewBuildsRankingsGroup(t *testing.T) {
	streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{}, streams)
	if cmd.Name() != "rankings" {
		t.Fatalf("name=%q", cmd.Name())
	}
	got := map[string]bool{}
	for _, sub := range cmd.Commands() {
		got[sub.Name()] = true
	}
	for _, name := range []string{"movies", "actors", "playback"} {
		if !got[name] {
			t.Fatalf("rankings missing subcommand %q", name)
		}
	}
}

func TestNewMoviesHasNoJSONFlag(t *testing.T) {
	streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := NewMovies(&invocation.RootOptions{}, streams)
	for _, flag := range []string{"type", "period", "has-magnets"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Fatalf("movies missing --%s", flag)
		}
	}
	if cmd.Flags().Lookup("json") != nil {
		t.Fatal("movies must not have --json in this baseline")
	}
}

func TestNewActorsFlags(t *testing.T) {
	streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := NewActors(&invocation.RootOptions{}, streams)
	if cmd.Flags().Lookup("period") == nil {
		t.Fatal("actors missing --period")
	}
}

func TestWriteMoviesEmpty(t *testing.T) {
	var out, errb bytes.Buffer
	if err := writeMovies(&out, &errb, nil); err != nil {
		t.Fatal(err)
	}
	if out.String() != "" || errb.String() != "(空列表)\n" {
		t.Fatalf("out=%q err=%q", out.String(), errb.String())
	}
}

func TestWriteNamedNoCount(t *testing.T) {
	var out, errb bytes.Buffer
	if err := writeNamedNoCount(&out, &errb, []map[string]any{
		{"id": "A", "name_zht": "中文", "videos_count": float64(9)},
		{"id": "B", "name": "Plain"},
	}); err != nil {
		t.Fatal(err)
	}
	want := "A\t中文\nB\tPlain\n"
	if out.String() != want {
		t.Fatalf("got %q want %q", out.String(), want)
	}
}
