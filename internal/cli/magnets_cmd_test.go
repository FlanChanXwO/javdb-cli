package cli

import (
	"bytes"
	"strings"
	"testing"
)

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
	if strings.Contains(s, "needs login") {
		t.Fatalf("magnets must not require login")
	}
}
