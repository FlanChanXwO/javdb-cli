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

func TestPrintMovieCommentsDoesNotDropContent(t *testing.T) {
	var out, errb bytes.Buffer
	PrintMovieComments(&out, &errb, []map[string]any{{
		"id": "review-1", "user": map[string]any{"nickname": "alice"}, "score": 4.5,
		"created_at": "2026-08-04", "content": "完整评论内容",
	}})
	for _, want := range []string{"review-1", "alice", "4.5", "完整评论内容"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("comment output missing %q: %s", want, out.String())
		}
	}
}
