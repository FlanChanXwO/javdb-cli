package comments

import (
	"bytes"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"strings"
	"testing"
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
	for _, want := range []string{"--id", "--json", "--page", "--limit", "never fetches later pages"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("help missing %s:\n%s", want, out.String())
		}
	}
}

func TestNewValidatesPageBeforeNetwork(t *testing.T) {
	streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{}, streams)
	var out, errb bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errb)
	cmd.SetArgs([]string{"ABC", "--page", "0"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "--page must be positive") {
		t.Fatalf("expected page validation error, got %v", err)
	}
}

func TestWriteCommentsPreservesFields(t *testing.T) {
	var out, errb bytes.Buffer
	err := writeComments(&out, &errb, []map[string]any{
		{"id": "c1", "user_name": "alice", "score": float64(5), "created_at": "2026-01-02", "content": "hello"},
		{"id": "c2", "username": "bob", "comment": "cmt"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "c1\talice\t5\t2026-01-02\nhello\nc2\tbob\t\t\ncmt\n"
	if out.String() != want {
		t.Fatalf("got %q want %q", out.String(), want)
	}
}
