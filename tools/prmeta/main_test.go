package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/tools/internal/verificationpolicy"
)

func TestValidateTemplateAndPipeline(t *testing.T) {
	root := t.TempDir()
	whitelistPath := filepath.Join(root, "command-whitelist.txt")
	if err := os.WriteFile(whitelistPath, []byte("javdb *\njavdb !auth\necho *\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	w, err := verificationpolicy.LoadWhitelist(whitelistPath)
	if err != nil {
		t.Fatal(err)
	}
	fence := string([]byte{96, 96, 96})
	body := "## 变更点 / Changes\n\n- change\n\n## 验证步骤 / Verification\n\n" +
		fence + "test\necho ABC-123 | javdb detail\n" + fence +
		"\n\n## 检查清单 / Checklist\n\n- [x] one\n- [x] two\n"
	got := validate(body, w, "javdb", root)
	if !got.TemplateOK || !got.TestOK || len(got.Commands.Commands) != 1 || got.Hash == "" {
		t.Fatalf("unexpected validation: %#v", got)
	}
}

func TestValidateRejectsSecondBlockAndUncheckedChecklist(t *testing.T) {
	root := t.TempDir()
	whitelistPath := filepath.Join(root, "command-whitelist.txt")
	if err := os.WriteFile(whitelistPath, []byte("javdb *\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	w, err := verificationpolicy.LoadWhitelist(whitelistPath)
	if err != nil {
		t.Fatal(err)
	}
	fence := string([]byte{96, 96, 96})
	body := "## 变更点 / Changes\n## 验证步骤 / Verification\n" +
		fence + "test\njavdb --version\n" + fence + "\n" +
		fence + "test\njavdb --version\n" + fence +
		"\n## 检查清单 / Checklist\n- [ ] unchecked\n"
	got := validate(body, w, "javdb", root)
	if got.TemplateOK || got.TestOK {
		t.Fatalf("expected template and test failure: %#v", got)
	}
}
