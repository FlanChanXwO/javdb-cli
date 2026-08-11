package browse

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
	for _, want := range []string{"--zone", "--tag", "--main", "--year", "--month", "--sort", "--order", "--page", "--limit", "--has-magnets", "--json"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("help missing %s", want)
		}
	}
}

func TestWriteMovieRowsEmpty(t *testing.T) {
	var out, errb bytes.Buffer
	if err := writeMovieRows(&out, &errb, nil); err != nil {
		t.Fatal(err)
	}
	if out.String() != "" || errb.String() != "(空列表)\n" {
		t.Fatalf("out=%q err=%q", out.String(), errb.String())
	}
}
