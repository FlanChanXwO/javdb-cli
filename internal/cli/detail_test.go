package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrintDetailGraph(t *testing.T) {
	var out bytes.Buffer
	PrintDetail(&out, map[string]any{
		"id": "9DGB5X", "number": "SSIS-589", "title": "T", "score": "4.4",
		"release_date": "2023-01-25", "magnets_count": float64(16),
		"series_id": "dQ5", "series_name": "SSS-BODY",
		"maker_id": "7R", "maker_name": "S1",
		"director_id": "9dX", "director_name": "紋℃",
		"actors": []any{map[string]any{"id": "9Dqpw", "name": "山手梨愛"}},
		"tags":   []any{map[string]any{"id": "17", "name": "巨乳"}},
	})
	s := out.String()
	for _, want := range []string{"9DGB5X", "SSIS-589", "dQ5", "7R", "9Dqpw", "山手梨愛", "17", "巨乳"} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %s in %q", want, s)
		}
	}
}

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
}
