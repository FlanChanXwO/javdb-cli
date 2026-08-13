package director

import (
	"bytes"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"strings"
	"testing"
)

func TestNewBuildsDirectorCommand(t *testing.T) {
	streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{}, streams)
	if cmd.Name() != "director" || cmd.Use != "director REF" {
		t.Fatalf("name=%q use=%q", cmd.Name(), cmd.Use)
	}
	for _, flag := range []string{"zone", "tag", "main", "sort", "order", "page", "limit", "all", "has-magnets", "json"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Fatalf("missing --%s", flag)
		}
	}
}

func TestNewRequiresRef(t *testing.T) {
	streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{}, streams)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "keyword or an image") {
		t.Fatalf("expected arg error, got %v", err)
	}
}
