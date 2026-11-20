package list

import (
	"bytes"
	"strings"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/internal/environment"
	"github.com/FlanChanXwO/javdb-cli/internal/session"
)

func TestNewBuildsListCommand(t *testing.T) {
	streams := environment.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&session.Options{}, streams)
	if cmd.Name() != "list" || cmd.Use != "list REF" {
		t.Fatalf("name=%q use=%q", cmd.Name(), cmd.Use)
	}
	for _, flag := range []string{"zone", "tag", "main", "sort", "order", "page", "limit", "all", "has-magnets", "json"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Fatalf("missing --%s", flag)
		}
	}
}

func TestNewRequiresRef(t *testing.T) {
	streams := environment.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&session.Options{}, streams)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "list: input required") {
		t.Fatalf("expected keyword/image error, got %v", err)
	}
}
