package watched

import (
	"bytes"
	"strings"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/internal/environment"
	"github.com/FlanChanXwO/javdb-cli/internal/session"
)

func TestNewBuildsWatchedCommand(t *testing.T) {
	streams := environment.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&session.Options{}, streams)
	if cmd.Name() != "watched" || cmd.Use != "watched" {
		t.Fatalf("name=%q use=%q", cmd.Name(), cmd.Use)
	}
	if cmd.Flags().Lookup("has-magnets") == nil {
		t.Fatal("missing --has-magnets")
	}
}

func TestNewHelp(t *testing.T) {
	streams := environment.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&session.Options{}, streams)
	var out, errb bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errb)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("help error = %v", err)
	}
	if !strings.Contains(out.String(), "List watched movies") {
		t.Fatalf("help: %s", out.String())
	}
}

func TestWriteMoviesEmpty(t *testing.T) {
	var out, errb bytes.Buffer
	if err := writeMovies(&out, &errb, nil); err != nil {
		t.Fatal(err)
	}
	if out.String() != "" || errb.String() != "(empty list)\n" {
		t.Fatalf("out=%q err=%q", out.String(), errb.String())
	}
}
