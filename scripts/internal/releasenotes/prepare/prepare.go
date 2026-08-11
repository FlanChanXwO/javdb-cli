// Package prepare 将已审阅的计划渲染为按版本归档的双语变更说明文件。
package prepare

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"

	"github.com/FlanChanXwO/javdb-cli/scripts/internal/releasenotes/document"
	"github.com/FlanChanXwO/javdb-cli/scripts/internal/releasenotes/model"
)

// Run 执行 prepare 子命令。
func Run(arguments []string) error {
	flags := flag.NewFlagSet("prepare", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	version := flags.String("version", "", "semantic version without v")
	previous := flags.String("previous", "", "previous v-prefixed tag; empty for the initial release")
	date := flags.String("date", "", "release date in YYYY-MM-DD form")
	changelogRoot := flags.String("changelog-root", "changelog", "changelog root directory")
	plan := flags.String("plan", "", "reviewed JSON release plan")
	audit := flags.String("audit", "", "optional JSON audit report whose sources must be covered")
	apply := flags.Bool("apply", false, "write the versioned notes and indexes after rendering")
	replace := flags.Bool("replace", false, "replace an existing versioned note pair during an authorized historical backfill")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("prepare accepts no positional arguments: %q", flags.Arg(0))
	}
	if *version == "" || *date == "" || *plan == "" {
		return errors.New("prepare requires --version, --date, and --plan")
	}
	return PrepareRelease(model.PrepareConfig{
		Version:       *version,
		Previous:      *previous,
		Date:          *date,
		ChangelogRoot: *changelogRoot,
		PlanPath:      *plan,
		AuditPath:     *audit,
		Apply:         *apply,
		Replace:       *replace,
	})
}

// PrepareRelease 校验并渲染发布计划；Apply=false 时只列出将要写入的路径。
func PrepareRelease(config model.PrepareConfig) error {
	if !document.IsSemanticVersion(config.Version) {
		return fmt.Errorf("invalid semantic version %q", config.Version)
	}
	if !datePattern.MatchString(config.Date) {
		return fmt.Errorf("invalid release date %q", config.Date)
	}
	if _, err := time.Parse("2006-01-02", config.Date); err != nil {
		return fmt.Errorf("invalid release date %q: %w", config.Date, err)
	}
	if config.ChangelogRoot == "" || config.PlanPath == "" {
		return errors.New("changelog root and plan path are required")
	}
	if config.Replace && !config.Apply {
		return errors.New("prepare --replace requires --apply")
	}
	plan, err := ReadPreparePlan(config.PlanPath)
	if err != nil {
		return err
	}
	if err := ValidatePreparePlan(plan); err != nil {
		return err
	}
	if err := ValidatePreparePlanMetadata(plan, config.Version, config.Previous); err != nil {
		return err
	}
	var report *model.AuditReport
	if config.AuditPath != "" {
		parsedReport, err := document.ReadAuditReport(config.AuditPath)
		if err != nil {
			return err
		}
		if err := ValidatePlanCoverage(plan, parsedReport); err != nil {
			return err
		}
		report = &parsedReport
	}
	files, err := renderPreparedRelease(config, plan)
	if err != nil {
		return err
	}
	if !config.Apply {
		for _, path := range sortedFilePaths(files) {
			fmt.Fprintln(os.Stdout, path)
		}
		return nil
	}
	for _, path := range sortedFilePaths(files) {
		if err := writePreparedFile(path, files[path], config.Replace); err != nil {
			return err
		}
	}
	directory := filepath.Join(config.ChangelogRoot, "v"+config.Version)
	if report != nil {
		return document.ValidateSourceCoverage(directory, config.Version, config.Previous, *report)
	}
	return document.ValidateReleaseDirectory(directory, config.Version, config.Previous)
}

// ReadPreparePlan 读取 JSON 发布计划。
func ReadPreparePlan(path string) (model.PreparePlan, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return model.PreparePlan{}, fmt.Errorf("read release plan: %w", err)
	}
	var plan model.PreparePlan
	if err := json.Unmarshal(body, &plan); err != nil {
		return model.PreparePlan{}, fmt.Errorf("parse release plan: %w", err)
	}
	return plan, nil
}

// ValidatePreparePlan 检查计划分类、双语文案、来源和贡献者记录。
func ValidatePreparePlan(plan model.PreparePlan) error {
	if len(plan.Entries) == 0 {
		return errors.New("release plan has no entries")
	}
	for index, entry := range plan.Entries {
		if !document.IsReleaseNoteCategory(entry.Category) || entry.Category == "None" {
			return fmt.Errorf("entry %d has unsupported category %q", index+1, entry.Category)
		}
		if entry.English == "" || entry.Chinese == "" {
			return fmt.Errorf("entry %d must contain both English and Simplified Chinese text", index+1)
		}
		if len(entry.Sources) == 0 {
			return fmt.Errorf("entry %d has no sources", index+1)
		}
		entrySources := make(map[string]struct{})
		for _, source := range entry.Sources {
			if _, err := document.RenderSourceLink(source); err != nil {
				return fmt.Errorf("entry %d source: %w", index+1, err)
			}
			if _, exists := entrySources[source]; exists {
				return fmt.Errorf("release plan entry %d repeats source %s", index+1, source)
			}
			entrySources[source] = struct{}{}
		}
	}
	contributors := make(map[string]struct{})
	for _, contributor := range plan.NewContributors {
		if contributor.Login == "" || contributor.ProfileURL == "" || contributor.PullNumber <= 0 || contributor.PullURL == "" {
			return errors.New("new contributor record is incomplete")
		}
		if _, err := document.RenderSourceLink(contributor.PullURL); err != nil {
			return fmt.Errorf("new contributor %q: %w", contributor.Login, err)
		}
		if _, exists := contributors[contributor.Login]; exists {
			return fmt.Errorf("release plan repeats new contributor %q", contributor.Login)
		}
		contributors[contributor.Login] = struct{}{}
	}
	return nil
}

// ValidatePreparePlanMetadata 将计划版本和比较范围绑定到命令行参数。
func ValidatePreparePlanMetadata(plan model.PreparePlan, version, previous string) error {
	if plan.Version != version {
		return fmt.Errorf("release plan version %q does not match requested version %q", plan.Version, version)
	}
	if previous == "" {
		if plan.PreviousTag != nil || plan.CompareURL != nil {
			return errors.New("initial release plan must use null previous_tag and compare_url")
		}
		return nil
	}
	if plan.PreviousTag == nil || *plan.PreviousTag != previous {
		return fmt.Errorf("release plan previous_tag does not match requested previous tag %q", previous)
	}
	expected := document.ChangelogCompareLink(version, previous)
	if plan.CompareURL == nil || *plan.CompareURL != expected {
		return fmt.Errorf("release plan compare_url must be %q", expected)
	}
	return nil
}

// ValidatePlanCoverage 确认计划覆盖 audit 报告中的所有有效来源和贡献者。
func ValidatePlanCoverage(plan model.PreparePlan, report model.AuditReport) error {
	planned := make(map[string]struct{})
	for _, entry := range plan.Entries {
		for _, source := range entry.Sources {
			planned[source] = struct{}{}
		}
	}
	for _, contributor := range plan.NewContributors {
		planned[contributor.PullURL] = struct{}{}
	}
	for _, source := range report.Sources {
		if source.Kind == "pull_request" && (source.Note == nil || source.Issue != "") {
			return fmt.Errorf("audit source %s is not classified: %s", source.URL, source.Issue)
		}
		if source.Note != nil && source.Note.Category == "None" {
			continue
		}
		if _, ok := planned[source.URL]; !ok {
			return fmt.Errorf("release plan does not cover audited source %s", source.URL)
		}
	}
	plannedContributors := make(map[string]model.NewContributor, len(plan.NewContributors))
	for _, contributor := range plan.NewContributors {
		plannedContributors[contributor.Login] = contributor
	}
	for _, contributor := range report.NewContributors {
		planned, ok := plannedContributors[contributor.Login]
		if !ok {
			return fmt.Errorf("release plan does not include audited new contributor %q", contributor.Login)
		}
		if planned.ProfileURL != contributor.ProfileURL || planned.PullNumber != contributor.PullNumber || planned.PullURL != contributor.PullURL {
			return fmt.Errorf("release plan new contributor %q differs from the audit report", contributor.Login)
		}
		delete(plannedContributors, contributor.Login)
	}
	for login := range plannedContributors {
		return fmt.Errorf("release plan includes new contributor %q absent from the audit report", login)
	}
	return nil
}

var datePattern = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)

func renderPreparedRelease(config model.PrepareConfig, plan model.PreparePlan) (map[string][]byte, error) {
	english, err := document.RenderPreparedReleaseDocument(config, plan, false)
	if err != nil {
		return nil, err
	}
	chinese, err := document.RenderPreparedReleaseDocument(config, plan, true)
	if err != nil {
		return nil, err
	}
	englishIndex, err := document.UpdateChangelogIndex(filepath.Join(config.ChangelogRoot, "README.md"), config.Version, config.Previous, config.Date, config.Replace)
	if err != nil {
		return nil, err
	}
	chineseIndex, err := document.UpdateChangelogIndex(filepath.Join(config.ChangelogRoot, "README.zh-CN.md"), config.Version, config.Previous, config.Date, config.Replace)
	if err != nil {
		return nil, err
	}
	directory := filepath.Join(config.ChangelogRoot, "v"+config.Version)
	return map[string][]byte{
		filepath.Join(directory, "en.md"):                      english,
		filepath.Join(directory, "zh-CN.md"):                   chinese,
		filepath.Join(config.ChangelogRoot, "README.md"):       englishIndex,
		filepath.Join(config.ChangelogRoot, "README.zh-CN.md"): chineseIndex,
	}, nil
}

func sortedFilePaths(files map[string][]byte) []string {
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func writePreparedFile(path string, body []byte, replace bool) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if _, err := os.Lstat(path); err == nil {
		base := filepath.Base(path)
		if base != "README.md" && base != "README.zh-CN.md" && !replace {
			return fmt.Errorf("refusing to replace existing file %s", path)
		}
		return os.WriteFile(path, body, 0o644)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	if _, err := file.Write(body); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return nil
}
