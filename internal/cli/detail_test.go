package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestDetailHelp(t *testing.T) {
	var out, errb bytes.Buffer
	code := Run([]string{"detail", "--help"}, strings.NewReader(""), &out, &errb)
	if code != 0 {
		t.Fatalf("%d %s", code, errb.String())
	}
	s := out.String()
	for _, want := range []string{"--id", "--magnets", "--json"} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %s", want)
		}
	}
	if strings.Contains(s, "needs login") {
		t.Fatalf("detail must not require login")
	}
}
