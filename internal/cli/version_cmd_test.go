package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/internal/buildinfo"
)

func TestVersionTextAndJSON(t *testing.T) {
	buildinfo.Version = "0.1.0"
	buildinfo.Commit = "deadbeef"
	buildinfo.BuildDate = "2026-07-18T00:00:00Z"

	var out, errb bytes.Buffer
	if code := Run([]string{"version"}, strings.NewReader(""), &out, &errb); code != 0 {
		t.Fatalf("text: %s", errb.String())
	}
	if !strings.Contains(out.String(), "javdb v0.1.0") {
		t.Fatalf("text out: %q", out.String())
	}

	out.Reset()
	errb.Reset()
	if code := Run([]string{"version", "--json"}, strings.NewReader(""), &out, &errb); code != 0 {
		t.Fatalf("json: %s", errb.String())
	}
	var got map[string]string
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["version"] != "v0.1.0" {
		t.Fatalf("want v0.1.0 got %q full=%v", got["version"], got)
	}
	if got["commit"] != "deadbeef" {
		t.Fatalf("%v", got)
	}
}
