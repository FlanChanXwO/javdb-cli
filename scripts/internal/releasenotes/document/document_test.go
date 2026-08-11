package document

import (
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

func TestParseReleaseNoteDeclaration(t *testing.T) {
	t.Parallel()

	note, err := ParseReleaseNoteDeclaration(`
## Summary

Implemented a user-visible change.

<!-- release-note
category: Changed
breaking: true
summary: Reworked the download request contract.
none_reason:
-->
`)
	if err != nil {
		t.Fatalf("parse declaration: %v", err)
	}
	if note.Category != "Changed" || !note.Breaking || note.Summary != "Reworked the download request contract." {
		t.Fatalf("release note = %#v", note)
	}
}

func TestParseReleaseNoteDeclarationRequiresNoneReason(t *testing.T) {
	t.Parallel()

	_, err := ParseReleaseNoteDeclaration(`<!-- release-note
category: None
breaking: false
summary: No release entry.
none_reason:
-->`)
	if err == nil || !strings.Contains(err.Error(), "none_reason") {
		t.Fatalf("parse error = %v, want none_reason validation", err)
	}
}

func TestParseReleaseNoteDeclarationRejectsTemplatePlaceholders(t *testing.T) {
	t.Parallel()

	_, err := ParseReleaseNoteDeclaration(`<!-- release-note
category: __REQUIRED__
breaking: __REQUIRED__
summary: __REQUIRED__
none_reason: __REQUIRED__
-->`)
	if err == nil || !strings.Contains(err.Error(), "unsupported category") {
		t.Fatalf("parse error = %v, want placeholder rejection", err)
	}
}

func TestRecommendedVersionBump(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name  string
		notes []model.ReleaseNote
		want  string
	}{
		{name: "maintenance", notes: []model.ReleaseNote{{Category: "Maintenance", Summary: "Refresh CI."}}, want: "patch"},
		{name: "feature", notes: []model.ReleaseNote{{Category: "Added", Summary: "Add APNG."}}, want: "minor"},
		{name: "breaking", notes: []model.ReleaseNote{{Category: "Changed", Breaking: true, Summary: "Change output."}}, want: "major"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := RecommendedVersionBump(test.notes); got != test.want {
				t.Fatalf("recommended bump = %q, want %q", got, test.want)
			}
		})
	}
}

func TestRenderSourceLink(t *testing.T) {
	t.Parallel()
	for _, test := range []struct{ source, want string }{
		{"https://github.com/FlanChanXwO/javdb-cli/pull/42", "[#42](https://github.com/FlanChanXwO/javdb-cli/pull/42)"},
		{"https://github.com/FlanChanXwO/javdb-cli/commit/abcdef0123456789", "[`abcdef0`](https://github.com/FlanChanXwO/javdb-cli/commit/abcdef0123456789)"},
	} {
		if got, err := RenderSourceLink(test.source); err != nil || got != test.want {
			t.Fatalf("render source %q = %q, %v; want %q", test.source, got, err, test.want)
		}
	}
	if _, err := RenderSourceLink("https://example.com/42"); err == nil {
		t.Fatal("unsupported source must fail")
	}
}

func TestValidateCoverageRejectsMissingAuditSource(t *testing.T) {
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
		{Kind: "pull_request", URL: "https://github.com/FlanChanXwO/javdb-cli/pull/42", Note: &model.ReleaseNote{Category: "Added", Summary: "Add one."}},
		{Kind: "pull_request", URL: "https://github.com/FlanChanXwO/javdb-cli/pull/43", Note: &model.ReleaseNote{Category: "Fixed", Summary: "Fix two."}},
	}}
	if err := ValidateSourceCoverage(root, "1.2.0", "v1.1.0", report); err == nil || !strings.Contains(err.Error(), "does not cover") {
		t.Fatalf("source coverage error = %v, want missing source error", err)
	}
}
