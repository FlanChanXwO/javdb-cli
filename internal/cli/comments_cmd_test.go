package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestCommentsHelpShowsOnePageFlags(t *testing.T) {
	var out, errb bytes.Buffer
	code := Run([]string{"comments", "--help"}, strings.NewReader(""), &out, &errb)
	if code != 0 {
		t.Fatalf("%d %s", code, errb.String())
	}
	for _, want := range []string{"--id", "--page", "--limit", "--json", "never fetches later pages"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("comments help missing %q: %s", want, out.String())
		}
	}
}
