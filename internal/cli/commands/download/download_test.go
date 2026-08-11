package download

import (
	"bytes"
	"strings"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/app"
)

func TestNewHelpDocumentsSinglePreviewImage(t *testing.T) {
	aio := app.NewIO(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&app.Flags{}, aio)
	var out, errb bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errb)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("help error = %v", err)
	}
	for _, want := range []string{"--thumbnail", "--preview-image", "--preview-video", "only the first preview image"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("download help missing %q: %s", want, out.String())
		}
	}
}

func TestNewRequiresSelectedOutputBeforeNetwork(t *testing.T) {
	aio := app.NewIO(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&app.Flags{}, aio)
	var out, errb bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errb)
	cmd.SetArgs([]string{"WTEX-15"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "set at least one") {
		t.Fatalf("expected media selection error, got %v", err)
	}
}

func TestNewRejectsWhitespaceOnlyOutput(t *testing.T) {
	aio := app.NewIO(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&app.Flags{}, aio)
	var out, errb bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errb)
	cmd.SetArgs([]string{"WTEX-15", "--thumbnail", " \t "})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "set at least one") {
		t.Fatalf("expected whitespace-only error, got %v", err)
	}
}
