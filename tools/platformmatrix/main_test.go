package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveFiltersCapabilityAndRejectsDuplicates(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "platforms.json")
	body := `{"platforms":[
{"goos":"linux","goarch":"amd64","runner":"linux","capabilities":["smoke","container"]},
{"goos":"darwin","goarch":"arm64","runner":"mac","capabilities":["smoke"]}
]}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := resolve(path, "container")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Include) != 1 || result.Include[0].GOOS != "linux" || result.Include[0].GOARCH != "amd64" || result.Include[0].Artifact != "linux-amd64" {
		t.Fatalf("unexpected matrix: %#v", result)
	}

	duplicate := `{"platforms":[
{"goos":"linux","goarch":"amd64","runner":"a","capabilities":["smoke"]},
{"goos":"linux","goarch":"amd64","runner":"b","capabilities":["smoke"]}
]}`
	if err := os.WriteFile(path, []byte(duplicate), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := resolve(path, "smoke"); err == nil {
		t.Fatal("expected duplicate platform rejection")
	}
}
