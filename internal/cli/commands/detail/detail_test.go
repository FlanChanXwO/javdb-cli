package detail

import (
	"bytes"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"strings"
	"testing"
)

func TestNewHelpListsFlags(t *testing.T) {
	streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{}, streams)
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
	streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{}, streams)
	var out, errb bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errb)
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "accepts 1 arg(s), received 0") {
		t.Fatalf("expected arg error, got %v", err)
	}
}

func TestRenderDetailWritesGraphLines(t *testing.T) {
	var out bytes.Buffer
	renderDetail(&out, map[string]any{
		"number": "SSIS-001", "id": "x1", "title": "T", "score": float64(8),
		"release_date": "2026-01-02", "magnets_count": float64(3),
		"actors": []any{map[string]any{"id": "a1", "name": "山手"}},
	})
	for _, want := range []string{"番号\tSSIS-001", "id\tx1", "标题\tT", "评分\t8", "日期\t2026-01-02", "磁力数\t3", "演员\ta1\t山手"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("detail output missing %q:\n%s", want, out.String())
		}
	}
}

func TestRenderMagnetsEmpty(t *testing.T) {
	var out, errb bytes.Buffer
	renderMagnets(&out, &errb, nil)
	if out.String() != "" || errb.String() != "(无磁力链)\n" {
		t.Fatalf("out=%q err=%q", out.String(), errb.String())
	}
}
