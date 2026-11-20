package unmark

import (
	"bytes"
	"strings"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/internal/environment"
	"github.com/FlanChanXwO/javdb-cli/internal/session"
)

func TestNewBuildsUnmarkCommand(t *testing.T) {
	streams := environment.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&session.Options{}, streams)
	if cmd.Name() != "unmark" || cmd.Use != "unmark NUMBER" {
		t.Fatalf("name=%q use=%q", cmd.Name(), cmd.Use)
	}
	if cmd.Flags().Lookup("id") == nil {
		t.Fatal("missing --id")
	}
}

func TestNewRequiresNumber(t *testing.T) {
	streams := environment.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&session.Options{}, streams)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "unmark: input required") {
		t.Fatalf("expected arg error, got %v", err)
	}
}
