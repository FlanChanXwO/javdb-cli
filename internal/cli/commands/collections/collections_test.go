package collections

import (
	"bytes"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"strings"
	"testing"
)

func TestNewBuildsCollectionsCommand(t *testing.T) {
	streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{}, streams)
	if cmd.Name() != "collections" || cmd.Use != "collections KIND" {
		t.Fatalf("name=%q use=%q", cmd.Name(), cmd.Use)
	}
}

func TestNewRequiresKind(t *testing.T) {
	streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{}, streams)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "keyword or an image") {
		t.Fatalf("expected arg error, got %v", err)
	}
}

func TestWriteNamedEmpty(t *testing.T) {
	var out, errb bytes.Buffer
	if err := writeNamed(&out, &errb, nil); err != nil {
		t.Fatal(err)
	}
	if out.String() != "" || errb.String() != "(空列表)\n" {
		t.Fatalf("out=%q err=%q", out.String(), errb.String())
	}
}
