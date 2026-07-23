package update

import "testing"

func TestSemanticVersionOrdering(t *testing.T) {
	cases := []struct {
		name   string
		first  string
		second string
		want   int
	}{
		{name: "patch", first: "v1.0.1", second: "v1.0.0", want: 1},
		{name: "release after prerelease", first: "v1.0.0", second: "v1.0.0-rc.1", want: 1},
		{name: "numeric prerelease", first: "v1.0.0-rc.2", second: "v1.0.0-rc.10", want: -1},
		{name: "short prerelease", first: "v1.0.0-rc", second: "v1.0.0-rc.1", want: -1},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			first, err := parseSemanticVersion(test.first)
			if err != nil {
				t.Fatal(err)
			}
			second, err := parseSemanticVersion(test.second)
			if err != nil {
				t.Fatal(err)
			}
			if got := first.compare(second); got != test.want {
				t.Fatalf("%s compare %s = %d, want %d", test.first, test.second, got, test.want)
			}
		})
	}
}

func TestParseSemanticVersionRejectsInvalidTags(t *testing.T) {
	for _, tag := range []string{"1.2.3", "v1.2", "v01.2.3", "v1.2.3-01", "v1.2.3+"} {
		if _, err := parseSemanticVersion(tag); err == nil {
			t.Fatalf("parseSemanticVersion(%q) unexpectedly succeeded", tag)
		}
	}
}
