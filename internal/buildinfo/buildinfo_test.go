package buildinfo

import "testing"

func TestNormalizeVersion(t *testing.T) {
	cases := map[string]string{
		"dev":           "dev",
		"":              "dev",
		"0.1.0":         "v0.1.0",
		"v0.1.0":        "v0.1.0",
		"v0.1.0-beta.1": "v0.1.0-beta.1",
	}
	for in, want := range cases {
		if got := NormalizeVersion(in); got != want {
			t.Fatalf("%q -> %q want %q", in, got, want)
		}
	}
}

func TestCurrentJSONShape(t *testing.T) {
	Version = "0.1.0"
	Commit = "abc"
	BuildDate = "2026-07-18"
	info := Current()
	if info.Version != "v0.1.0" || info.Commit != "abc" {
		t.Fatalf("%+v", info)
	}
	if info.IsDevelopment() {
		t.Fatal("should not be dev")
	}
	Version = "dev"
	if !Current().IsDevelopment() {
		t.Fatal("dev")
	}
}
