package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestEntityHelp(t *testing.T) {
	for _, name := range []string{"actor", "series", "maker", "director", "code", "list"} {
		var out, errb bytes.Buffer
		code := Run([]string{name, "--help"}, strings.NewReader(""), &out, &errb)
		if code != 0 {
			t.Fatalf("%s help failed: %s", name, errb.String())
		}
		s := out.String()
		for _, want := range []string{"--main", "--tag", "--zone", "--has-magnets", "--json"} {
			if !strings.Contains(s, want) {
				t.Fatalf("%s missing %s", name, want)
			}
		}
	}
}
