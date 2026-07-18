package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestTagsBrowseHelp(t *testing.T) {
	for _, cmd := range []string{"tags", "browse"} {
		var out, errb bytes.Buffer
		code := Run([]string{cmd, "--help"}, strings.NewReader(""), &out, &errb)
		if code != 0 {
			t.Fatalf("%s help %d %s", cmd, code, errb.String())
		}
	}
	var out, errb bytes.Buffer
	if code := Run([]string{"browse", "--help"}, strings.NewReader(""), &out, &errb); code != 0 {
		t.Fatal(errb.String())
	}
	s := out.String()
	for _, want := range []string{"--tag", "--main", "--year", "--zone", "--has-magnets"} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %s", want)
		}
	}
}
