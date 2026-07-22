package appapi

import "testing"

func TestAllPagesDedupAndStop(t *testing.T) {
	pages := map[int][]map[string]any{
		1: {{"id": "a"}, {"id": "b"}},
		2: {{"id": "b"}, {"id": "c"}}, // b dup
		3: {},
	}
	out, err := AllPages(func(p int) ([]map[string]any, error) {
		return pages[p], nil
	}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 3 {
		t.Fatalf("got %d want 3: %v", len(out), out)
	}
}

func TestCollectionSpecsNoLists(t *testing.T) {
	if _, ok := CollectionSpecs["lists"]; ok {
		t.Fatal("lists must not be exposed")
	}
	for _, k := range []string{"actors", "series", "codes", "makers", "directors"} {
		if _, ok := CollectionSpecs[k]; !ok {
			t.Fatalf("missing %s", k)
		}
	}
}
