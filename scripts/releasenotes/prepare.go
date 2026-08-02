// 发布准备将已审阅的计划渲染为按版本归档的双语变更说明文件。
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func runPrepare(arguments []string) error {
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
	return prepareRelease(prepareConfig{
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

func prepareRelease(config prepareConfig) error {
	if !semanticVersionPattern.MatchString(config.Version) {
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
	plan, err := readPreparePlan(config.PlanPath)
	if err != nil {
		return err
	}
	if err := validatePreparePlan(plan); err != nil {
		return err
	}
	if err := validatePreparePlanMetadata(plan, config.Version, config.Previous); err != nil {
		return err
	}
	var report *auditReport
	if config.AuditPath != "" {
		parsedReport, err := readAuditReport(config.AuditPath)
		if err != nil {
			return err
		}
		if err := validatePlanCoverage(plan, parsedReport); err != nil {
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
		return validateSourceCoverage(directory, config.Version, config.Previous, *report)
	}
	return validateReleaseDirectory(directory, config.Version, config.Previous)
}

func readPreparePlan(path string) (preparePlan, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return preparePlan{}, fmt.Errorf("read release plan: %w", err)
	}
	var plan preparePlan
	if err := json.Unmarshal(body, &plan); err != nil {
		return preparePlan{}, fmt.Errorf("parse release plan: %w", err)
	}
	return plan, nil
}

func readAuditReport(path string) (auditReport, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return auditReport{}, fmt.Errorf("read audit report: %w", err)
	}
	var report auditReport
	if err := json.Unmarshal(body, &report); err != nil {
		return auditReport{}, fmt.Errorf("parse audit report: %w", err)
	}
	return report, nil
}

func validatePreparePlan(plan preparePlan) error {
	if len(plan.Entries) == 0 {
		return errors.New("release plan has no entries")
	}
	for index, entry := range plan.Entries {
		if _, ok := releaseNoteCategories[entry.Category]; !ok || entry.Category == "None" {
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
			if _, err := renderSourceLink(source); err != nil {
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
		if _, err := renderSourceLink(contributor.PullURL); err != nil {
			return fmt.Errorf("new contributor %q: %w", contributor.Login, err)
		}
		if _, exists := contributors[contributor.Login]; exists {
			return fmt.Errorf("release plan repeats new contributor %q", contributor.Login)
		}
		contributors[contributor.Login] = struct{}{}
	}
	return nil
}

// validatePreparePlanMetadata 将版本清单绑定到 prepare 的命令行参数，避免复用错误版本
// 或错误比较范围的 plan 生成看似正常、实际归属错误的发布说明。
func validatePreparePlanMetadata(plan preparePlan, version, previous string) error {
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
	expected := changelogCompareLink(version, previous)
	if plan.CompareURL == nil || *plan.CompareURL != expected {
		return fmt.Errorf("release plan compare_url must be %q", expected)
	}
	return nil
}

func validatePlanCoverage(plan preparePlan, report auditReport) error {
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
	plannedContributors := make(map[string]newContributor, len(plan.NewContributors))
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

func renderPreparedRelease(config prepareConfig, plan preparePlan) (map[string][]byte, error) {
	english, err := renderReleaseDocument(config, plan, false)
	if err != nil {
		return nil, err
	}
	chinese, err := renderReleaseDocument(config, plan, true)
	if err != nil {
		return nil, err
	}
	englishIndex, err := updateChangelogIndex(filepath.Join(config.ChangelogRoot, "README.md"), config.Version, config.Previous, config.Date, config.Replace)
	if err != nil {
		return nil, err
	}
	chineseIndex, err := updateChangelogIndex(filepath.Join(config.ChangelogRoot, "README.zh-CN.md"), config.Version, config.Previous, config.Date, config.Replace)
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

func renderReleaseDocument(config prepareConfig, plan preparePlan, chinese bool) ([]byte, error) {
	var builder strings.Builder
	fmt.Fprintf(&builder, "# v%s — %s\n", config.Version, config.Date)
	sectionEntries := make(map[string][]string)
	for _, entry := range plan.Entries {
		section := entry.Category
		if entry.Breaking {
			section = "Breaking changes"
		}
		if chinese {
			section = chineseSectionName(section)
		}
		text := entry.English
		if chinese {
			text = entry.Chinese
		}
		links := make([]string, 0, len(entry.Sources))
		for _, source := range entry.Sources {
			link, err := renderSourceLink(source)
			if err != nil {
				return nil, err
			}
			links = append(links, link)
		}
		sectionEntries[section] = append(sectionEntries[section], "- "+text+" ("+strings.Join(links, ", ")+")")
	}
	order := releaseContentSections.English
	if chinese {
		order = releaseContentSections.Chinese
	}
	for _, section := range order {
		entries := sectionEntries[section]
		if len(entries) == 0 {
			continue
		}
		fmt.Fprintf(&builder, "\n## %s\n\n%s\n", section, strings.Join(entries, "\n"))
	}
	if len(plan.NewContributors) > 0 {
		name := "New Contributors"
		if chinese {
			name = "新贡献者"
		}
		fmt.Fprintf(&builder, "\n## %s\n\n", name)
		for _, contributor := range plan.NewContributors {
			link, err := renderSourceLink(contributor.PullURL)
			if err != nil {
				return nil, err
			}
			if chinese {
				fmt.Fprintf(&builder, "- [@%s](%s) 在 %s 中完成首次贡献。\n", contributor.Login, contributor.ProfileURL, link)
			} else {
				fmt.Fprintf(&builder, "- [@%s](%s) made their first contribution in %s.\n", contributor.Login, contributor.ProfileURL, link)
			}
		}
	}
	link := changelogCompareLink(config.Version, config.Previous)
	if chinese {
		fmt.Fprintf(&builder, "\n**完整变更**：[%s](%s)\n", changelogCompareLabel(config.Version, config.Previous), link)
	} else {
		fmt.Fprintf(&builder, "\n**Full Changelog**: [%s](%s)\n", changelogCompareLabel(config.Version, config.Previous), link)
	}
	return []byte(builder.String()), nil
}

func chineseSectionName(english string) string {
	for index, candidate := range releaseContentSections.English {
		if candidate == english {
			return releaseContentSections.Chinese[index]
		}
	}
	return english
}

func renderSourceLink(source string) (string, error) {
	if !sourcePattern.MatchString(source) || sourcePattern.FindString(source) != source {
		return "", fmt.Errorf("unsupported source URL %q", source)
	}
	if pull, ok := strings.CutPrefix(source, "https://github.com/FlanChanXwO/javdb-cli/pull/"); ok {
		return "[#" + pull + "](" + source + ")", nil
	}
	commit, _ := strings.CutPrefix(source, "https://github.com/FlanChanXwO/javdb-cli/commit/")
	if len(commit) < 7 {
		return "", fmt.Errorf("commit source URL has a short hash %q", source)
	}
	return "[`" + commit[:7] + "`](" + source + ")", nil
}

func changelogCompareLink(version, previous string) string {
	if previous == "" {
		return "https://github.com/FlanChanXwO/javdb-cli/commits/v" + version
	}
	return "https://github.com/FlanChanXwO/javdb-cli/compare/" + previous + "...v" + version
}

func changelogCompareLabel(version, previous string) string {
	if previous == "" {
		return "v" + version + " commits"
	}
	return previous + "...v" + version
}

func updateChangelogIndex(path, version, previous, date string, replace bool) ([]byte, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read changelog index: %w", err)
	}
	content := string(body)
	versionMarker := "v" + version + "]("
	if strings.Contains(content, versionMarker) {
		if replace {
			return body, nil
		}
		return nil, fmt.Errorf("changelog index already contains v%s", version)
	}
	anchor := "| Unreleased | — | [English](unreleased/en.md) · [简体中文](unreleased/zh-CN.md) |\n"
	position := strings.Index(content, anchor)
	if position < 0 {
		return nil, fmt.Errorf("changelog index %s has no Unreleased row", path)
	}
	row := fmt.Sprintf("| [v%s](%s) | %s | [English](v%s/en.md) · [简体中文](v%s/zh-CN.md) |\n", version, changelogCompareLink(version, previous), date, version, version)
	position += len(anchor)
	return []byte(content[:position] + row + content[position:]), nil
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
