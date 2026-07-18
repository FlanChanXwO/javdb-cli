package cli

import (
	"bytes"
	"encoding/json"
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

func TestRenderSearchMoviesJSON(t *testing.T) {
	aio := &appIO{out: &bytes.Buffer{}, err: &bytes.Buffer{}}
	moviesRaw, _ := json.Marshal([]map[string]any{
		{"number": "SSIS-589", "id": "x", "title": "T", "magnets_count": float64(2)},
		{"number": "ZERO", "id": "y", "title": "Z", "magnets_count": float64(0)},
	})
	res := map[string]json.RawMessage{"movies": moviesRaw}
	if err := renderSearch(aio, res, "", true, true); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(aio.out.(*bytes.Buffer).Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	arr, _ := got["movies"].([]any)
	if len(arr) != 1 {
		t.Fatalf("has-magnets filter: %v", got)
	}
}

func TestRenderSearchNamed(t *testing.T) {
	aio := &appIO{out: &bytes.Buffer{}, err: &bytes.Buffer{}}
	raw, _ := json.Marshal([]map[string]any{
		{"id": "9Dqpw", "name": "山手梨愛", "videos_count": float64(10)},
	})
	res := map[string]json.RawMessage{"actors": raw}
	if err := renderSearch(aio, res, "actor", false, false); err != nil {
		t.Fatal(err)
	}
	s := aio.out.(*bytes.Buffer).String()
	if !strings.Contains(s, "9Dqpw") || !strings.Contains(s, "山手梨愛") {
		t.Fatalf("%q", s)
	}
}
