package prepare

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/scripts/internal/releasenotes/document"
	"github.com/FlanChanXwO/javdb-cli/scripts/internal/releasenotes/model"
)

func writeReleaseNote(t *testing.T, root, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func writeJSONFile(t *testing.T, path string, value any) {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal %s: %v", path, err)
	}
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestPrepareRendersBilingualNotesAndIndex(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeReleaseNote(t, root, "README.md", "# Changelog\n\n| Version | Date | Release notes |\n| --- | --- | --- |\n| Unreleased | — | [English](unreleased/en.md) · [简体中文](unreleased/zh-CN.md) |\n| [v1.0.0](https://github.com/FlanChanXwO/javdb-cli/commits/v1.0.0) | 2026-01-01 | [English](v1.0.0/en.md) · [简体中文](v1.0.0/zh-CN.md) |\n")
	writeReleaseNote(t, root, "README.zh-CN.md", "# 更新日志\n\n| 版本 | 日期 | 发布说明 |\n| --- | --- | --- |\n| Unreleased | — | [English](unreleased/en.md) · [简体中文](unreleased/zh-CN.md) |\n| [v1.0.0](https://github.com/FlanChanXwO/javdb-cli/commits/v1.0.0) | 2026-01-01 | [English](v1.0.0/en.md) · [简体中文](v1.0.0/zh-CN.md) |\n")
	planPath := filepath.Join(root, "plan.json")
	previous := "v1.0.0"
	compare := "https://github.com/FlanChanXwO/javdb-cli/compare/v1.0.0...v1.1.0"
	plan := model.PreparePlan{Version: "1.1.0", PreviousTag: &previous, CompareURL: &compare, Entries: []model.PreparedEntry{{
		Category: "Added",
		English:  "Add APNG downloads.",
		Chinese:  "新增 APNG 下载。",
		Sources:  []string{"https://github.com/FlanChanXwO/javdb-cli/pull/42"},
	}}, NewContributors: []model.NewContributor{{
		Login: "new-contributor", ProfileURL: "https://github.com/new-contributor", PullNumber: 42, PullURL: "https://github.com/FlanChanXwO/javdb-cli/pull/42",
	}}}
	writeJSONFile(t, planPath, plan)

	if err := PrepareRelease(model.PrepareConfig{Version: "1.1.0", Previous: "v1.0.0", Date: "2026-07-30", ChangelogRoot: root, PlanPath: planPath, Apply: true}); err != nil {
		t.Fatalf("prepare release: %v", err)
	}
	if err := document.ValidateReleaseDirectory(filepath.Join(root, "v1.1.0"), "1.1.0", "v1.0.0"); err != nil {
		t.Fatalf("validate prepared notes: %v", err)
	}
	english, err := os.ReadFile(filepath.Join(root, "v1.1.0", "en.md"))
	if err != nil {
		t.Fatalf("read English notes: %v", err)
	}
	if !strings.Contains(string(english), "## New Contributors") || !strings.Contains(string(english), "[@new-contributor](https://github.com/new-contributor) made their first contribution in [#42]") {
		t.Fatalf("English notes missing contributor: %s", english)
	}
	index, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatalf("read index: %v", err)
	}
	if !strings.Contains(string(index), "| [v1.1.0](https://github.com/FlanChanXwO/javdb-cli/compare/v1.0.0...v1.1.0) | 2026-07-30") {
		t.Fatalf("index missing new release row: %s", index)
	}
}

func TestPrepareRejectsMismatchedPlanMetadata(t *testing.T) {
	t.Parallel()

	previous := "v1.0.0"
	compare := "https://github.com/FlanChanXwO/javdb-cli/compare/v1.0.0...v1.1.0"
	plan := model.PreparePlan{Version: "1.1.0", PreviousTag: &previous, CompareURL: &compare, Entries: []model.PreparedEntry{{
		Category: "Added", English: "Add one.", Chinese: "新增一。", Sources: []string{"https://github.com/FlanChanXwO/javdb-cli/pull/42"},
	}}}
	if err := ValidatePreparePlanMetadata(plan, "1.1.1", "v1.1.0"); err == nil || !strings.Contains(err.Error(), "version") {
		t.Fatalf("metadata validation error = %v, want version mismatch", err)
	}
}

func TestPrepareRejectsRepeatedSource(t *testing.T) {
	t.Parallel()

	plan := model.PreparePlan{Entries: []model.PreparedEntry{{
		Category: "Added", English: "Add one.", Chinese: "新增一。", Sources: []string{
			"https://github.com/FlanChanXwO/javdb-cli/pull/42",
			"https://github.com/FlanChanXwO/javdb-cli/pull/42",
		},
	}}}
	if err := ValidatePreparePlan(plan); err == nil || !strings.Contains(err.Error(), "repeats source") {
		t.Fatalf("validate plan error = %v, want repeated source error", err)
	}
}

func TestPreparePlanRequiresAuditedNewContributor(t *testing.T) {
	t.Parallel()

	plan := model.PreparePlan{Entries: []model.PreparedEntry{{
		Category: "Added", English: "Add one.", Chinese: "新增一。", Sources: []string{"https://github.com/FlanChanXwO/javdb-cli/pull/42"},
	}}}
	report := model.AuditReport{
		Sources:         []model.AuditSource{{Kind: "pull_request", URL: "https://github.com/FlanChanXwO/javdb-cli/pull/42", Note: &model.ReleaseNote{Category: "Added", Summary: "Add one."}}},
		NewContributors: []model.NewContributor{{Login: "new-contributor", ProfileURL: "https://github.com/new-contributor", PullNumber: 42, PullURL: "https://github.com/FlanChanXwO/javdb-cli/pull/42"}},
	}
	if err := ValidatePlanCoverage(plan, report); err == nil || !strings.Contains(err.Error(), "new contributor") {
		t.Fatalf("plan coverage error = %v, want missing contributor error", err)
	}
}
