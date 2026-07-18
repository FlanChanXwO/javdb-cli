package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRankingsTop250Help(t *testing.T) {
	for _, args := range [][]string{
		{"rankings", "--help"},
		{"rankings", "movies", "--help"},
		{"rankings", "actors", "--help"},
		{"rankings", "playback", "--help"},
		{"top250", "--help"},
	} {
		var out, errb bytes.Buffer
		code := Run(args, strings.NewReader(""), &out, &errb)
		if code != 0 {
			t.Fatalf("%v: %s", args, errb.String())
		}
	}
}
