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

func TestPrintLists(t *testing.T) {
	var out, errb bytes.Buffer
	PrintLists(&out, &errb, []map[string]any{
		{"id": "x", "name": "N", "movies_count": float64(3), "privacy": "open", "views_count": float64(9)},
	})
	s := out.String()
	if !strings.Contains(s, "x") || !strings.Contains(s, "N") {
		t.Fatal(s)
	}
}
