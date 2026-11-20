package detail

import (
	"bytes"
	"strings"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/internal/environment"
	"github.com/FlanChanXwO/javdb-cli/internal/session"
)

func TestNewHelpListsFlags(t *testing.T) {
	streams := environment.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&session.Options{}, streams)
	var out, errb bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errb)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("help error = %v", err)
	}
	for _, want := range []string{"--id", "--magnets", "--json"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("help missing %s:\n%s", want, out.String())
		}
	}
	if strings.Contains(out.String(), "needs login") {
		t.Fatalf("detail must not require login")
	}
}

func TestNewRequiresNumber(t *testing.T) {
	streams := environment.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&session.Options{}, streams)
	var out, errb bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errb)
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "detail: input required") {
		t.Fatalf("expected keyword/image error, got %v", err)
	}
}

func TestRenderDetailWritesGraphLines(t *testing.T) {
	var out bytes.Buffer
	renderDetail(&out, map[string]any{
		"number": "SSIS-001", "id": "x1", "title": "T", "score": float64(8),
		"release_date": "2026-01-02", "duration": float64(130), "magnets_count": float64(3),
		"actors": []any{map[string]any{"id": "a1", "name": "山手"}},
	})
	for _, want := range []string{"Number\tSSIS-001", "id\tx1", "Title\tT", "Rating\t8", "Date\t2026-01-02", "Duration\t130 minutes", "Magnet count\t3", "Actor\ta1\t山手"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("detail output missing %q:\n%s", want, out.String())
		}
	}
}

func TestRenderMagnetsEmpty(t *testing.T) {
	var out, errb bytes.Buffer
	renderMagnets(&out, &errb, nil)
	if out.String() != "" || errb.String() != "(no magnets)\n" {
		t.Fatalf("out=%q err=%q", out.String(), errb.String())
	}
}
