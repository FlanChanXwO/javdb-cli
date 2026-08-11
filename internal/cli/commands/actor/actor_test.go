package actor

import (
	"bytes"
	"strings"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/app"
)

func TestNewBuildsActorCommand(t *testing.T) {
	aio := app.NewIO(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&app.Flags{}, aio)
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
	aio := app.NewIO(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&app.Flags{}, aio)
	var out, errb bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errb)
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "accepts 1 arg(s), received 0") {
		t.Fatalf("expected arg error, got %v", err)
	}
}
