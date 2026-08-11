package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestSearchHelpListsFlags(t *testing.T) {
	var out, err bytes.Buffer
	code := Run([]string{"search", "--help"}, strings.NewReader(""), &out, &err)
	if code != 0 {
		t.Fatalf("code=%d err=%s", code, err.String())
	}
	s := out.String() + err.String()
	for _, want := range []string{"--zone", "--sort", "--filter-by", "--type", "--has-magnets", "--json", "--page", "--limit"} {
		if !strings.Contains(s, want) {
			t.Fatalf("help missing %s:\n%s", want, s)
		}
	}
}
