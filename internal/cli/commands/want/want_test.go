package want

import (
	"bytes"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"strings"
	"testing"
)

func TestNewBuildsWantCommand(t *testing.T) {
	streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{}, streams)
	if cmd.Name() != "want" || cmd.Use != "want" {
		t.Fatalf("name=%q use=%q", cmd.Name(), cmd.Use)
	}
	if cmd.Flags().Lookup("has-magnets") == nil {
		t.Fatal("missing --has-magnets")
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
	if !strings.Contains(out.String(), "List want-to-watch (想看) movies") {
		t.Fatalf("help: %s", out.String())
	}
}
