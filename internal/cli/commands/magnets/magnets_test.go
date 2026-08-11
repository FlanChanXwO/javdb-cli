package magnets

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
	for _, want := range []string{"--cnsub", "--hd", "--min-size", "--best", "--json", "--id"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("help missing %s", want)
		}
	}
	if strings.Contains(out.String(), "needs login") {
		t.Fatalf("magnets must not require login")
	}
}

func TestParseSizeMiB(t *testing.T) {
	n, err := ParseSizeMiB("4GB")
	if err != nil || n != 4096 {
		t.Fatalf("%d %v", n, err)
	}
	n, err = ParseSizeMiB("500MB")
	if err != nil || n != 500 {
		t.Fatalf("%d %v", n, err)
	}
	n, err = ParseSizeMiB("2000")
	if err != nil || n != 2000 {
		t.Fatalf("%d %v", n, err)
	}
}

func TestWriteMagnetsEmpty(t *testing.T) {
	var out, errb bytes.Buffer
	writeMagnets(&out, &errb, nil)
	if out.String() != "" || errb.String() != "(无磁力链)\n" {
		t.Fatalf("out=%q err=%q", out.String(), errb.String())
	}
}
