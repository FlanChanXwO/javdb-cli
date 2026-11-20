package want

import (
	"bytes"
	"strings"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/internal/environment"
	"github.com/FlanChanXwO/javdb-cli/internal/session"
)

func TestNewBuildsWantCommand(t *testing.T) {
	streams := environment.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&session.Options{}, streams)
	if cmd.Name() != "want" || cmd.Use != "want" {
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
	if !strings.Contains(out.String(), "List want-to-watch movies") {
		t.Fatalf("help: %s", out.String())
	}
}
