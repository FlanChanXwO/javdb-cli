package list

import (
	"bytes"
	"strings"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/app"
)

func TestNewBuildsListCommand(t *testing.T) {
	aio := app.NewIO(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&app.Flags{}, aio)
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
	aio := app.NewIO(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&app.Flags{}, aio)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "accepts 1 arg(s), received 0") {
		t.Fatalf("expected arg error, got %v", err)
	}
}
