package result

import (
	"testing"
)

func TestProjectNamedPrefersChineseName(t *testing.T) {
	row := ProjectNamed(map[string]any{"id": "9D", "name_zht": "山手", "name": "Yam", "videos_count": float64(10)})
	if row.ID != "9D" || row.Name != "山手" || !row.HasCount || row.Count != float64(10) {
		t.Fatalf("row = %+v", row)
	}
	if got := row.Line(); got != "9D\t山手\t10" {
		t.Fatalf("Line = %q", got)
	}
}

func TestProjectNamedFallsBackToNameAndMoviesCount(t *testing.T) {
	row := ProjectNamed(map[string]any{"id": "B", "name": "N", "movies_count": float64(3)})
	if row.Name != "N" || !row.HasCount || row.Count != float64(3) {
		t.Fatalf("row = %+v", row)
	}
	if got := row.Line(); got != "B\tN\t3" {
		t.Fatalf("Line = %q", got)
	}
}

func TestProjectNamedNoCount(t *testing.T) {
	row := ProjectNamed(map[string]any{"id": "A", "name": "X"})
	if row.HasCount || row.Count != nil {
		t.Fatalf("row = %+v", row)
	}
	if got := row.Line(); got != "A\tX" {
		t.Fatalf("Line = %q", got)
	}
}

func TestProjectNamedAll(t *testing.T) {
	rows := ProjectNamedAll([]map[string]any{
		{"id": "A", "name_zht": "中文", "videos_count": float64(2)},
		{"id": "B", "name": "Plain"},
	})
	if len(rows) != 2 || rows[0].Name != "中文" || rows[1].HasCount {
		t.Fatalf("rows = %+v", rows)
	}
}
