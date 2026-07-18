package cli

import "testing"

func TestFilterHasMagnets(t *testing.T) {
	in := []map[string]any{
		{"number": "A", "magnets_count": float64(3)},
		{"number": "B", "magnets_count": float64(0)},
		{"number": "C"}, // missing → keep
	}
	out := FilterHasMagnets(in)
	if len(out) != 2 || anyString(out[0]["number"]) != "A" || anyString(out[1]["number"]) != "C" {
		t.Fatalf("%v", out)
	}
}

func TestSearchTypeKey(t *testing.T) {
	if SearchTypeKey("actor") != "actors" {
		t.Fatal(SearchTypeKey("actor"))
	}
	if SearchTypeKey("list") != "lists" {
		t.Fatal(SearchTypeKey("list"))
	}
	if SearchTypeKey("") != "movies" {
		t.Fatal(SearchTypeKey(""))
	}
}
