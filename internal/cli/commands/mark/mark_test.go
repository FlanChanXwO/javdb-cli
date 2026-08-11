package mark

import (
	"bytes"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"strings"
	"testing"
)

func TestNewBuildsMarkCommand(t *testing.T) {
	streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{}, streams)
	if cmd.Name() != "mark" || cmd.Use != "mark NUMBER" {
		t.Fatalf("name=%q use=%q", cmd.Name(), cmd.Use)
	}
	for _, flag := range []string{"watched", "want", "score", "content", "id"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Fatalf("missing --%s", flag)
		}
	}
}

func TestNewRequiresExactlyOneFlagBeforeNetwork(t *testing.T) {
	streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{}, streams)
	var out, errb bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errb)
	cmd.SetArgs([]string{"ABC"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "specify exactly one of --watched or --want") {
		t.Fatalf("expected flag validation error, got %v", err)
	}
}
