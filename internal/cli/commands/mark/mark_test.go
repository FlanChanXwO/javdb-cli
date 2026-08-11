package mark

import (
	"bytes"
	"strings"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/app"
)

func TestNewBuildsMarkCommand(t *testing.T) {
	aio := app.NewIO(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&app.Flags{}, aio)
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
	aio := app.NewIO(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&app.Flags{}, aio)
	var out, errb bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errb)
	cmd.SetArgs([]string{"ABC"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "specify exactly one of --watched or --want") {
		t.Fatalf("expected flag validation error, got %v", err)
	}
}
