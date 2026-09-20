package releaseversion

import "testing"

func TestParseStableTag(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		tag     string
		version string
		valid   bool
	}{
		{tag: "v1.2.3", version: "1.2.3", valid: true},
		{tag: "v0.0.0", version: "0.0.0", valid: true},
		{tag: "1.2.3", valid: false},
		{tag: "v01.2.3", valid: false},
		{tag: "v1.2.3-rc.1", valid: false},
	} {
		version, err := ParseStableTag(test.tag)
		if !test.valid {
			if err == nil {
				t.Errorf("ParseStableTag(%q) error = nil, want rejection", test.tag)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseStableTag(%q) error = %v", test.tag, err)
			continue
		}
		if version != test.version {
			t.Errorf("ParseStableTag(%q) = %q, want %q", test.tag, version, test.version)
		}
	}
}
