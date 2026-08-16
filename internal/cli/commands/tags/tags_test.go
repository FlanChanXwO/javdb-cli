package tags

import (
	"bytes"
	"strings"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	storagetags "github.com/FlanChanXwO/javdb-cli/internal/storage/tags"
)

func TestNewHelpListsFlags(t *testing.T) {
	streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{}, streams)
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

func TestNewTextAndHumanOutputModes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	path, err := storagetags.Path("censored")
	if err != nil {
		t.Fatal(err)
	}
	if err := storagetags.Save(path, &storagetags.Doc{
		Zone:       "censored",
		Categories: []storagetags.Category{{ID: "cat-1", NameEN: "General", Tags: []storagetags.Tag{{ID: "tag-1", NameEN: "Tag One", NameZH: "标签一"}}}},
	}); err != nil {
		t.Fatal(err)
	}

	textStreams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	textCmd := New(&invocation.RootOptions{Host: "https://example.invalid"}, textStreams)
	if err := textCmd.Execute(); err != nil {
		t.Fatalf("text execute: %v", err)
	}
	if got := textStreams.Out.(*bytes.Buffer).String(); got != "Tag One\n" {
		t.Fatalf("non-TTY output = %q, want stable ref", got)
	}

	humanStreams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	humanStreams.OutIsTerminal = true
	humanCmd := New(&invocation.RootOptions{Host: "https://example.invalid"}, humanStreams)
	if err := humanCmd.Execute(); err != nil {
		t.Fatalf("human execute: %v", err)
	}
	if got := humanStreams.Out.(*bytes.Buffer).String(); !strings.Contains(got, "# cat-1\tGeneral") || !strings.Contains(got, "tag-1\tTag One\t标签一") {
		t.Fatalf("TTY output = %q, want taxonomy table", got)
	}
}
