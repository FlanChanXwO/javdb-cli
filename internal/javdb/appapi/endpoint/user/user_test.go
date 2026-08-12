package user

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

// TestReviewIDExactDecimal 覆盖 review id 的精确十进制格式化：detail 里的大整数 id 经
// map[string]any 解码为 float64 后，若以 fmt.Sprint 输出会变成科学计数法（如
// 2.44850177e+08），导致 unmark 的 DELETE 打到不存在的 review。
func TestReviewIDExactDecimal(t *testing.T) {
	cases := []struct {
		name string
		rev  map[string]any
		want string
	}{
		{"nil", nil, ""},
		{"missing id", map[string]any{"status": "watched"}, ""},
		{"zero id", map[string]any{"id": float64(0)}, ""},
		{"float64 large", map[string]any{"id": float64(244850177)}, "244850177"},
		{"int", map[string]any{"id": 244850177}, "244850177"},
		{"string", map[string]any{"id": "244850177"}, "244850177"},
		{"non numeric", map[string]any{"id": "abc"}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := reviewID(tc.rev); got != tc.want {
				t.Fatalf("reviewID(%v) = %q, want %q", tc.rev, got, tc.want)
			}
		})
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
