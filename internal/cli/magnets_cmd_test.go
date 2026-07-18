package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseSizeMiB(t *testing.T) {
	n, err := parseSizeMiB("4GB")
	if err != nil || n != 4096 {
		t.Fatalf("%d %v", n, err)
	}
	n, err = parseSizeMiB("500MB")
	if err != nil || n != 500 {
		t.Fatalf("%d %v", n, err)
	}
	n, err = parseSizeMiB("2000")
	if err != nil || n != 2000 {
		t.Fatalf("%d %v", n, err)
	}
}

func TestMagnetsHelp(t *testing.T) {
	var out, errb bytes.Buffer
	code := Run([]string{"magnets", "--help"}, strings.NewReader(""), &out, &errb)
	if code != 0 {
		t.Fatalf("%d %s", code, errb.String())
	}
	s := out.String()
	for _, want := range []string{"--cnsub", "--hd", "--min-size", "--best", "--json", "--id"} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %s", want)
		}
	}
}
