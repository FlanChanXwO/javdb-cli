package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestListsHelp(t *testing.T) {
	for _, args := range [][]string{
		{"lists", "--help"},
		{"lists", "show", "--help"},
		{"lists", "search", "--help"},
		{"lists", "related", "--help"},
	} {
		var out, errb bytes.Buffer
		code := Run(args, strings.NewReader(""), &out, &errb)
		if code != 0 {
			t.Fatalf("%v: %s", args, errb.String())
		}
	}
}
