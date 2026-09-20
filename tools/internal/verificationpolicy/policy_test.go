package verificationpolicy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseLinePreservesQuotedPipeAndSplitsPipeline(t *testing.T) {
	command, err := ParseLine(`echo "a | b" | javdb detail ABC-123`)
	if err != nil {
		t.Fatal(err)
	}
	if len(command.Stages) != 2 {
		t.Fatalf("stages = %d, want 2", len(command.Stages))
	}
	if got := command.Stages[0].Argv[1]; got != "a | b" {
		t.Fatalf("quoted pipe = %q", got)
	}
	if _, err := ParseLine("javdb search ABC-123 && echo nope"); err == nil {
		t.Fatal("expected shell operator rejection")
	}
}

func TestWhitelistDenyAndWorkspaceBoundary(t *testing.T) {
	root := t.TempDir()
	whitelistPath := filepath.Join(root, "whitelist.txt")
	if err := os.WriteFile(whitelistPath, []byte("javdb *\njavdb !auth\ncat @\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "image.jpg"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	w, err := LoadWhitelist(whitelistPath)
	if err != nil {
		t.Fatal(err)
	}
	allowed, _ := ParseLine("cat image.jpg | javdb search")
	if err := w.Validate(allowed, "javdb", root); err != nil {
		t.Fatalf("allowed pipeline rejected: %v", err)
	}
	denied, _ := ParseLine("javdb auth list")
	if err := w.Validate(denied, "javdb", root); err == nil {
		t.Fatal("expected auth rejection")
	}
	proxy, _ := ParseLine("javdb --proxy=https://example.invalid search ABC-123")
	if err := w.Validate(proxy, "javdb", root); err == nil {
		t.Fatal("expected inline proxy rejection")
	}
	escape, _ := ParseLine("cat ../outside")
	if err := w.Validate(escape, "javdb", root); err == nil {
		t.Fatal("expected workspace escape rejection")
	}
}
