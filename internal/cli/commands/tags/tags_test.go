package tags

import (
	"bytes"
	"strings"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/app"
)

func TestNewHelpListsFlags(t *testing.T) {
	aio := app.NewIO(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&app.Flags{}, aio)
	var out, errb bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errb)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("help error = %v", err)
	}
	for _, want := range []string{"--refresh", "--zone"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("help missing %s", want)
		}
	}
}
