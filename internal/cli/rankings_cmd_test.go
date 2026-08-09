package cli

import (
	"bytes"
	"encoding/json"
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
		if len(args) == 3 && !strings.Contains(out.String(), "--json") {
			t.Fatalf("%v: missing --json", args)
		}
	}
}

func TestRenderRankingsMoviesJSONFiltersMagnets(t *testing.T) {
	aio := &appIO{out: &bytes.Buffer{}, err: &bytes.Buffer{}}
	movies := []map[string]any{
		{"number": "HAS", "magnets_count": float64(1)},
		{"number": "NONE", "magnets_count": float64(0)},
	}
	if err := renderRankingsMovies(aio, FilterHasMagnets(movies), true); err != nil {
		t.Fatal(err)
	}
	var got map[string][]map[string]any
	if err := json.Unmarshal(aio.out.(*bytes.Buffer).Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got["movies"]) != 1 || got["movies"][0]["number"] != "HAS" {
		t.Fatalf("unexpected JSON: %#v", got)
	}
}

func TestRenderRankingsActorsJSON(t *testing.T) {
	aio := &appIO{out: &bytes.Buffer{}, err: &bytes.Buffer{}}
	actors := []map[string]any{{"id": "actor-1", "name": "Actor"}}
	if err := renderRankingsActors(aio, actors, true); err != nil {
		t.Fatal(err)
	}
	var got map[string][]map[string]any
	if err := json.Unmarshal(aio.out.(*bytes.Buffer).Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got["actors"]) != 1 || got["actors"][0]["id"] != "actor-1" {
		t.Fatalf("unexpected JSON: %#v", got)
	}
}
