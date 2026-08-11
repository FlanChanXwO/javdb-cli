package result

import (
	"encoding/json"
	"testing"
)

func TestProjectMovieFields(t *testing.T) {
	row := ProjectMovie(map[string]any{
		"number": "SSIS-001", "id": "x1", "title": "A", "release_date": "2026-01-02",
	})
	if row.Number != "SSIS-001" || row.ID != "x1" || row.Title != "A" || row.ReleaseDate != "2026-01-02" {
		t.Fatalf("row = %+v", row)
	}
	if got := row.Line(); got != "SSIS-001\tx1\tA\t2026-01-02" {
		t.Fatalf("Line = %q", got)
	}
}

func TestProjectMovieMissingReleaseDate(t *testing.T) {
	row := ProjectMovie(map[string]any{"number": "ABC-9", "id": "x2", "title": "B"})
	if row.ReleaseDate != "" {
		t.Fatalf("release date = %q, want empty", row.ReleaseDate)
	}
	if got := row.Line(); got != "ABC-9\tx2\tB" {
		t.Fatalf("Line = %q", got)
	}
}

func TestProjectMovieFloatIDTruncation(t *testing.T) {
	row := ProjectMovie(map[string]any{"number": "N", "id": float64(123.9), "title": "T"})
	if row.ID != "123" {
		t.Fatalf("float id = %q, want 123", row.ID)
	}
}

func TestProjectMovies(t *testing.T) {
	rows := ProjectMovies([]map[string]any{
		{"number": "A", "id": "a", "title": "T1", "release_date": "2026-01-01"},
		{"number": "B", "id": "b", "title": "T2"},
	})
	if len(rows) != 2 || rows[0].ReleaseDate != "2026-01-01" || rows[1].ReleaseDate != "" {
		t.Fatalf("rows = %+v", rows)
	}
}

func TestFilterMoviesWithMagnetsPreservesMissingField(t *testing.T) {
	in := []map[string]any{
		{"number": "A", "magnets_count": float64(3)},
		{"number": "B", "magnets_count": float64(0)},
		{"number": "C"},
	}
	out := FilterMoviesWithMagnets(in)
	want := []map[string]any{
		{"number": "A", "magnets_count": float64(3)},
		{"number": "C"},
	}
	if len(out) != 2 {
		t.Fatalf("len = %d, want 2: %v", len(out), out)
	}
	for i := range out {
		if out[i]["number"] != want[i]["number"] {
			t.Fatalf("out[%d] = %v, want %v", i, out[i], want[i])
		}
	}
}

func TestFilterMoviesWithMagnetsNumericVariants(t *testing.T) {
	in := []map[string]any{
		{"number": "A", "magnets_count": int64(2)},
		{"number": "B", "magnets_count": "0"},
		{"number": "C", "magnets_count": json.Number("3")},
	}
	out := FilterMoviesWithMagnets(in)
	if len(out) != 2 || out[0]["number"] != "A" || out[1]["number"] != "C" {
		t.Fatalf("out = %v", out)
	}
}
