package movie

import (
	"testing"
)

func TestResolveNumberExactMatchesCaseInsensitive(t *testing.T) {
	movies := []map[string]any{
		{"number": "SSIS-589", "id": "id-a"},
		{"number": "HZGD-246", "id": "id-b"},
	}
	id, err := ResolveNumberExact(movies, "ssis-589")
	if err != nil {
		t.Fatalf("ResolveNumberExact: %v", err)
	}
	if id != "id-a" {
		t.Errorf("id = %q", id)
	}
}

func TestResolveNumberExactRejectsZeroAndMultiple(t *testing.T) {
	for _, tc := range []struct {
		name   string
		movies []map[string]any
	}{
		{name: "no match", movies: []map[string]any{{"number": "HZGD-246", "id": "id-b"}}},
		{name: "two exact", movies: []map[string]any{
			{"number": "SSIS-589", "id": "id-a"},
			{"number": "ssis-589", "id": "id-b"},
		}},
		{name: "empty input", movies: nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			number := "SSIS-589"
			if tc.name == "empty input" {
				number = "  "
			}
			if _, err := ResolveNumberExact(tc.movies, number); err == nil {
				t.Fatal("ResolveNumberExact accepted ambiguous input")
			}
		})
	}
}

func TestResolveNumberExactDoesNotFallBackToFirstHit(t *testing.T) {
	// 严格解析不得回退到搜索首项：首项是近似命中（非完整相等）时必须失败。
	movies := []map[string]any{
		{"number": "HZGD-246", "id": "id-b"},
		{"number": "SSIS-58X", "id": "id-near"},
	}
	if _, err := ResolveNumberExact(movies, "SSIS-589"); err == nil {
		t.Fatal("ResolveNumberExact fell back to the first search hit")
	}
}

func TestResolveNumberKeepsLegacyFallback(t *testing.T) {
	// 旧 ResolveNumber 保持首项回退行为，图片链路不使用它。
	movies := []map[string]any{
		{"number": "HZGD-246", "id": "id-b"},
	}
	id, err := ResolveNumber(movies, "SSIS-589")
	if err != nil {
		t.Fatalf("ResolveNumber: %v", err)
	}
	if id != "id-b" {
		t.Errorf("legacy fallback id = %q", id)
	}
}
