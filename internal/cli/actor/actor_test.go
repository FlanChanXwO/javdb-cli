package actor

import (
	"bytes"
	"strings"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/internal/environment"
	"github.com/FlanChanXwO/javdb-cli/internal/session"
)

func TestNewBuildsActorCommand(t *testing.T) {
	streams := environment.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&session.Options{}, streams)
	if cmd.Name() != "actor" || cmd.Use != "actor REF" {
		t.Fatalf("name=%q use=%q", cmd.Name(), cmd.Use)
	}
	for _, flag := range []string{"zone", "tag", "main", "sort", "order", "page", "limit", "all", "has-magnets", "json"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Fatalf("missing --%s", flag)
		}
	}
	if cmd.Flags().Lookup("sort").DefValue != "release" || cmd.Flags().Lookup("limit").DefValue != "20" {
		t.Fatal("entity flag defaults changed")
	}
}

func TestNewRequiresRef(t *testing.T) {
	streams := environment.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&session.Options{}, streams)
	var out, errb bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errb)
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "actor: input required") {
		t.Fatalf("expected arg error, got %v", err)
	}
}
