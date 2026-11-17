package document

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/scripts/internal/releasenotes/model"
)

func writeReleaseNote(t *testing.T, root, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func TestValidateAcceptsMatchingBilingualReleaseNotes(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeReleaseNote(t, root, "en.md", `# v1.2.0 — 2026-07-30

## Added

- Added APNG output. ([#42](https://github.com/FlanChanXwO/javdb-cli/pull/42))

**Full Changelog**: [v1.1.0...v1.2.0](https://github.com/FlanChanXwO/javdb-cli/compare/v1.1.0...v1.2.0)
`)
	writeReleaseNote(t, root, "zh-CN.md", `# v1.2.0 — 2026-07-30

## 新增

- 新增 APNG 输出。([#42](https://github.com/FlanChanXwO/javdb-cli/pull/42))

**完整变更**：[v1.1.0...v1.2.0](https://github.com/FlanChanXwO/javdb-cli/compare/v1.1.0...v1.2.0)
`)

	if err := ValidateReleaseDirectory(root, "1.2.0", "v1.1.0"); err != nil {
		t.Fatalf("validate release directory: %v", err)
	}
}

func TestValidateRejectsMissingEntrySource(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeReleaseNote(t, root, "en.md", `# v1.2.0 — 2026-07-30

## Fixed

- Fixed an issue without an attribution.

**Full Changelog**: [v1.1.0...v1.2.0](https://github.com/FlanChanXwO/javdb-cli/compare/v1.1.0...v1.2.0)
`)
	writeReleaseNote(t, root, "zh-CN.md", `# v1.2.0 — 2026-07-30

## 修复

- 修复未标注来源的问题。

**完整变更**：[v1.1.0...v1.2.0](https://github.com/FlanChanXwO/javdb-cli/compare/v1.1.0...v1.2.0)
`)

	err := ValidateReleaseDirectory(root, "1.2.0", "v1.1.0")
	if err == nil || !strings.Contains(err.Error(), "source") {
		t.Fatalf("validate error = %v, want missing source error", err)
	}
}

func TestValidateRejectsMismatchedBilingualSources(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeReleaseNote(t, root, "en.md", `# v1.2.0 — 2026-07-30

## Changed

- Changed output. ([#42](https://github.com/FlanChanXwO/javdb-cli/pull/42))

**Full Changelog**: [v1.1.0...v1.2.0](https://github.com/FlanChanXwO/javdb-cli/compare/v1.1.0...v1.2.0)
`)
	writeReleaseNote(t, root, "zh-CN.md", `# v1.2.0 — 2026-07-30

## 变更

- 修改输出。([#43](https://github.com/FlanChanXwO/javdb-cli/pull/43))

**完整变更**：[v1.1.0...v1.2.0](https://github.com/FlanChanXwO/javdb-cli/compare/v1.1.0...v1.2.0)
`)

	err := ValidateReleaseDirectory(root, "1.2.0", "v1.1.0")
	if err == nil || !strings.Contains(err.Error(), "source sets differ") {
		t.Fatalf("validate error = %v, want bilingual source mismatch", err)
	}
}

func TestValidateAcceptsInitialReleaseCommitLink(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	entry := "([`abcdef0`](https://github.com/FlanChanXwO/javdb-cli/commit/abcdef0123456789))"
	writeReleaseNote(t, root, "en.md", "# v1.0.0 — 2026-07-30\n\n## Added\n\n- First release. "+entry+"\n\n**Full Changelog**: [v1.0.0 commits](https://github.com/FlanChanXwO/javdb-cli/commits/v1.0.0)\n")
	writeReleaseNote(t, root, "zh-CN.md", "# v1.0.0 — 2026-07-30\n\n## 新增\n\n- 首次发布。"+entry+"\n\n**完整变更**：[v1.0.0 commits](https://github.com/FlanChanXwO/javdb-cli/commits/v1.0.0)\n")

	if err := ValidateReleaseDirectory(root, "1.0.0", ""); err != nil {
		t.Fatalf("validate initial release: %v", err)
	}
}

func TestRenderWritesBilingualGitHubReleaseBody(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	entry := "([`abcdef0`](https://github.com/FlanChanXwO/javdb-cli/commit/abcdef0123456789))"
	writeReleaseNote(t, directory, "en.md", "# v1.0.0 — 2026-07-30\n\n## Added\n\n- First release. "+entry+"\n\n**Full Changelog**: [v1.0.0 commits](https://github.com/FlanChanXwO/javdb-cli/commits/v1.0.0)\n")
	writeReleaseNote(t, directory, "zh-CN.md", "# v1.0.0 — 2026-07-30\n\n## 新增\n\n- 首次发布。"+entry+"\n\n**完整变更**：[v1.0.0 commits](https://github.com/FlanChanXwO/javdb-cli/commits/v1.0.0)\n")
	output := filepath.Join(t.TempDir(), "release-notes.md")
	if err := RunRender([]string{"--version", "1.0.0", "--dir", directory, "--output", output}); err != nil {
		t.Fatalf("RunRender() error = %v", err)
	}
	body, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read rendered body: %v", err)
	}
	if got := string(body); !strings.Contains(got, "# English\n\n## Added") || !strings.Contains(got, "# 简体中文\n\n## 新增") {
		t.Fatalf("rendered bilingual body missing language sections: %s", body)
	}
}

func TestValidateCoverageAllowsUnlistedAuditSources(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeReleaseNote(t, root, "en.md", `# v1.2.0 — 2026-07-30

## Added

- Added one. ([#42](https://github.com/FlanChanXwO/javdb-cli/pull/42))

**Full Changelog**: [v1.1.0...v1.2.0](https://github.com/FlanChanXwO/javdb-cli/compare/v1.1.0...v1.2.0)
`)
	writeReleaseNote(t, root, "zh-CN.md", `# v1.2.0 — 2026-07-30

## 新增

- 新增一。([#42](https://github.com/FlanChanXwO/javdb-cli/pull/42))

**完整变更**：[v1.1.0...v1.2.0](https://github.com/FlanChanXwO/javdb-cli/compare/v1.1.0...v1.2.0)
`)
	report := model.AuditReport{Sources: []model.AuditSource{
		{Kind: "pull_request", URL: "https://github.com/FlanChanXwO/javdb-cli/pull/42"},
		{Kind: "pull_request", URL: "https://github.com/FlanChanXwO/javdb-cli/pull/43"},
	}}
	if err := ValidateSourceCoverage(root, "1.2.0", "v1.1.0", report); err != nil {
		t.Fatalf("source coverage error = %v, want omitted audit source to be allowed", err)
	}
}

func TestValidateCoverageRejectsUnlistedChangelogSource(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeReleaseNote(t, root, "en.md", `# v1.2.0 — 2026-07-30

## Added

- Added one. ([#42](https://github.com/FlanChanXwO/javdb-cli/pull/42))

**Full Changelog**: [v1.1.0...v1.2.0](https://github.com/FlanChanXwO/javdb-cli/compare/v1.1.0...v1.2.0)
`)
	writeReleaseNote(t, root, "zh-CN.md", `# v1.2.0 — 2026-07-30

## 新增

- 新增一。([#42](https://github.com/FlanChanXwO/javdb-cli/pull/42))

**完整变更**：[v1.1.0...v1.2.0](https://github.com/FlanChanXwO/javdb-cli/compare/v1.1.0...v1.2.0)
`)
	report := model.AuditReport{Sources: []model.AuditSource{
		{Kind: "pull_request", URL: "https://github.com/FlanChanXwO/javdb-cli/pull/43"},
	}}
	if err := ValidateSourceCoverage(root, "1.2.0", "v1.1.0", report); err == nil || !strings.Contains(err.Error(), "not present") {
		t.Fatalf("source coverage error = %v, want missing audit source error", err)
	}
}

func TestInternalSourcesStayAuditableButOutOfReleaseBody(t *testing.T) {
	t.Parallel()
	for _, visible := range []bool{false, true} {
		t.Run(fmt.Sprint(visible), func(t *testing.T) {
			root := t.TempDir()
			for _, locale := range []string{"en", "zh-CN"} {
				section := ""
				if visible {
					heading := "Fixed"
					if locale == "zh-CN" {
						heading = "修复"
					}
					section = "## " + heading + "\n\n- Fix output. ([#42](https://github.com/FlanChanXwO/javdb-cli/pull/42))\n\n"
				}
				writeReleaseNote(t, root, locale+".md", "# v1.2.0 — 2026-07-30\n\n"+section+"**Full Changelog**: [v1.1.0...v1.2.0](https://github.com/FlanChanXwO/javdb-cli/compare/v1.1.0...v1.2.0)\n\n<!-- internal:\n  [#43](https://github.com/FlanChanXwO/javdb-cli/pull/43)\n-->\n")
			}
			if err := ValidateReleaseDirectory(root, "1.2.0", "v1.1.0"); err != nil {
				t.Fatal(err)
			}
			report := model.AuditReport{Sources: []model.AuditSource{{URL: "https://github.com/FlanChanXwO/javdb-cli/pull/42"}}}
			if err := ValidateSourceCoverage(root, "1.2.0", "v1.1.0", report); err == nil {
				t.Fatal("internal source outside audit was accepted")
			}
			body, err := RenderGitHubReleaseBody(root, "1.2.0")
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(body, "internal:") || strings.Contains(body, "pull/43") || !strings.Contains(body, "Full Changelog") {
				t.Fatalf("internal source leaked or compare link missing: %s", body)
			}
		})
	}
}

func TestValidateRejectsMaintainerOnlySections(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	for _, locale := range []string{"en", "zh-CN"} {
		heading := "Maintenance"
		if locale == "zh-CN" {
			heading = "维护"
		}
		writeReleaseNote(t, root, locale+".md", "# v1.2.0 — 2026-07-30\n\n## "+heading+"\n\n- Refactor CI. ([#43](https://github.com/FlanChanXwO/javdb-cli/pull/43))\n\n**Full Changelog**: [v1.1.0...v1.2.0](https://github.com/FlanChanXwO/javdb-cli/compare/v1.1.0...v1.2.0)\n")
	}
	if err := ValidateReleaseDirectory(root, "1.2.0", "v1.1.0"); err == nil {
		t.Fatal("maintainer-only section accepted in public notes")
	}
}
