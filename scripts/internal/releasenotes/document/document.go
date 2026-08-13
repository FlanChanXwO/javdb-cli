// Package document 负责 release-note 文档、来源链接和 PR 声明的解析校验。
package document

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/FlanChanXwO/javdb-cli/scripts/internal/releasenotes/model"
)

var (
	releaseHeadingPattern  = regexp.MustCompile(`(?m)^# v([0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?)\s+[—-]\s+.+$`)
	sectionPattern         = regexp.MustCompile(`(?m)^##\s+(.+?)\s*$`)
	sourcePattern          = regexp.MustCompile(`https://github\.com/FlanChanXwO/javdb-cli/(?:pull/[0-9]+|commit/[0-9a-fA-F]{7,64})`)
	linkPattern            = regexp.MustCompile(`\[[^\]]+\]\((https://github\.com/FlanChanXwO/javdb-cli/(?:compare/[^)\s]+|commits/[^)\s]+))\)`)
	releaseNotePattern     = regexp.MustCompile(`(?s)<!--\s*release-note\s*\n(.*?)-->`)
	semanticVersionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?$`)
	datePattern            = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)
)

var releaseNoteCategories = map[string]struct{}{
	"Added":         {},
	"Changed":       {},
	"Fixed":         {},
	"Security":      {},
	"Documentation": {},
	"Maintenance":   {},
	"None":          {},
}

var (
	englishSectionOrder = []string{
		"Breaking changes",
		"Added",
		"Changed",
		"Fixed",
		"Security",
		"Documentation",
		"Maintenance",
		"New Contributors",
	}
	chineseSectionOrder = []string{
		"破坏性变更",
		"新增",
		"变更",
		"修复",
		"安全",
		"文档",
		"维护",
		"新贡献者",
	}
	releaseContentSections = struct {
		English []string
		Chinese []string
	}{
		English: []string{
			"Breaking changes",
			"Added",
			"Changed",
			"Fixed",
			"Security",
			"Documentation",
			"Maintenance",
		},
		Chinese: []string{
			"破坏性变更",
			"新增",
			"变更",
			"修复",
			"安全",
			"文档",
			"维护",
		},
	}
)

type releaseDocument struct {
	sections []releaseSection
	sources  []string
	compare  string
}

type releaseSection struct {
	name    string
	entries []string
}

// IsSemanticVersion 判断不带 v 前缀的版本字符串是否符合 release-note 契约。
func IsSemanticVersion(version string) bool { return semanticVersionPattern.MatchString(version) }

// IsReleaseNoteCategory 判断 PR 声明中的 category 是否受支持。
func IsReleaseNoteCategory(category string) bool {
	_, ok := releaseNoteCategories[category]
	return ok
}

// ParseReleaseNoteDeclaration 读取 PR 正文中的机器可读注释。
// 注释不直接显示在 GitHub 页面上，便于 CI 在离线事件载荷中稳定校验。
func ParseReleaseNoteDeclaration(body string) (model.ReleaseNote, error) {
	match := releaseNotePattern.FindStringSubmatch(body)
	if len(match) != 2 {
		return model.ReleaseNote{}, errors.New("missing release-note declaration")
	}
	values := make(map[string]string)
	for _, rawLine := range strings.Split(match[1], "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return model.ReleaseNote{}, fmt.Errorf("invalid release-note line %q", line)
		}
		key = strings.TrimSpace(key)
		if _, exists := values[key]; exists {
			return model.ReleaseNote{}, fmt.Errorf("release-note repeats %q", key)
		}
		values[key] = strings.TrimSpace(value)
	}
	for _, key := range []string{"category", "breaking", "summary", "none_reason"} {
		if _, ok := values[key]; !ok {
			return model.ReleaseNote{}, fmt.Errorf("release-note is missing %s", key)
		}
	}
	for key := range values {
		switch key {
		case "category", "breaking", "summary", "none_reason":
		default:
			return model.ReleaseNote{}, fmt.Errorf("release-note contains unsupported key %q", key)
		}
	}
	if !IsReleaseNoteCategory(values["category"]) {
		return model.ReleaseNote{}, fmt.Errorf("release-note has unsupported category %q", values["category"])
	}
	breaking, err := strconv.ParseBool(values["breaking"])
	if err != nil {
		return model.ReleaseNote{}, fmt.Errorf("release-note breaking: %w", err)
	}
	note := model.ReleaseNote{
		Category:   values["category"],
		Breaking:   breaking,
		Summary:    values["summary"],
		NoneReason: values["none_reason"],
	}
	if note.Summary == "" {
		return model.ReleaseNote{}, errors.New("release-note summary is required")
	}
	if note.Category == "None" {
		if note.Breaking {
			return model.ReleaseNote{}, errors.New("release-note None category cannot be breaking")
		}
		if note.NoneReason == "" {
			return model.ReleaseNote{}, errors.New("release-note none_reason is required for category None")
		}
	} else if note.NoneReason != "" {
		return model.ReleaseNote{}, errors.New("release-note none_reason is only valid for category None")
	}
	return note, nil
}

// RecommendedVersionBump 根据 release-note 声明计算建议的版本级别。
func RecommendedVersionBump(notes []model.ReleaseNote) string {
	bump := "none"
	for _, note := range notes {
		if note.Category == "None" {
			continue
		}
		if note.Breaking {
			return "major"
		}
		if note.Category == "Added" {
			bump = "minor"
			continue
		}
		if bump == "none" {
			bump = "patch"
		}
	}
	return bump
}

// RenderSourceLink 将受支持的 PR/commit URL 转为稳定的 changelog 链接。
func RenderSourceLink(source string) (string, error) {
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

// ChangelogCompareLink 返回版本说明底部的比较链接。
func ChangelogCompareLink(version, previous string) string {
	if previous == "" {
		return "https://github.com/FlanChanXwO/javdb-cli/commits/v" + version
	}
	return "https://github.com/FlanChanXwO/javdb-cli/compare/" + previous + "...v" + version
}

// ChangelogCompareLabel 返回版本说明底部的比较链接文字。
func ChangelogCompareLabel(version, previous string) string {
	if previous == "" {
		return "v" + version + " commits"
	}
	return previous + "...v" + version
}

// ValidateReleaseDirectory 检查可发布的双语版本文件。
func ValidateReleaseDirectory(directory, version, previous string) error {
	english, err := readReleaseDocument(filepath.Join(directory, "en.md"), version, englishSectionOrder)
	if err != nil {
		return fmt.Errorf("English changelog: %w", err)
	}
	chinese, err := readReleaseDocument(filepath.Join(directory, "zh-CN.md"), version, chineseSectionOrder)
	if err != nil {
		return fmt.Errorf("Simplified Chinese changelog: %w", err)
	}
	if !sameStrings(english.sources, chinese.sources) {
		return fmt.Errorf("bilingual source sets differ: English=%v SimplifiedChinese=%v", english.sources, chinese.sources)
	}
	if err := validateCompareLink(english.compare, version, previous); err != nil {
		return fmt.Errorf("English Full Changelog: %w", err)
	}
	if err := validateCompareLink(chinese.compare, version, previous); err != nil {
		return fmt.Errorf("Simplified Chinese Full Changelog: %w", err)
	}
	if english.compare != chinese.compare {
		return fmt.Errorf("bilingual compare links differ: English=%q SimplifiedChinese=%q", english.compare, chinese.compare)
	}
	return nil
}

// ValidateSourceCoverage 将已渲染的说明与审计报告逐项对照。
func ValidateSourceCoverage(directory, version, previous string, report model.AuditReport) error {
	if err := ValidateReleaseDirectory(directory, version, previous); err != nil {
		return err
	}
	english, err := readReleaseDocument(filepath.Join(directory, "en.md"), version, englishSectionOrder)
	if err != nil {
		return err
	}
	// 只要求 PR 来源被分类覆盖；直接 push 的 commit 由审计报告人工核对
	// （release workflow 记录在案），不强制要求 notes 逐一引用。
	expected := make(map[string]struct{})
	for _, source := range report.Sources {
		if source.Kind != "pull_request" {
			continue
		}
		if source.Note == nil || source.Issue != "" {
			return fmt.Errorf("audit source %s is not classified: %s", source.URL, source.Issue)
		}
		if source.Note.Category == "None" {
			continue
		}
		expected[source.URL] = struct{}{}
	}
	for _, contributor := range report.NewContributors {
		expected[contributor.PullURL] = struct{}{}
	}
	// notes 中引用的每个 source 都必须真实存在于审计报告。
	present := make(map[string]struct{}, len(report.Sources)+len(report.NewContributors))
	for _, source := range report.Sources {
		present[source.URL] = struct{}{}
	}
	for _, contributor := range report.NewContributors {
		present[contributor.PullURL] = struct{}{}
	}
	actual := make(map[string]struct{}, len(english.sources))
	for _, source := range english.sources {
		actual[source] = struct{}{}
	}
	for source := range expected {
		if _, ok := actual[source]; !ok {
			return fmt.Errorf("release notes does not cover audited source %s", source)
		}
	}
	for source := range actual {
		if _, ok := present[source]; !ok {
			return fmt.Errorf("release notes source %s is not present in the audit report", source)
		}
	}
	return nil
}

// ReadAuditReport 读取 prepare/validate 共用的 audit JSON 报告。
func ReadAuditReport(path string) (model.AuditReport, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return model.AuditReport{}, fmt.Errorf("read audit report: %w", err)
	}
	var report model.AuditReport
	if err := json.Unmarshal(body, &report); err != nil {
		return model.AuditReport{}, fmt.Errorf("parse audit report: %w", err)
	}
	return report, nil
}

// RunValidate 执行 validate 子命令。
func RunValidate(arguments []string) error {
	flags := flag.NewFlagSet("validate", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	version := flags.String("version", "", "semantic version without v")
	directory := flags.String("dir", "", "directory containing en.md and zh-CN.md")
	previous := flags.String("previous", "", "previous v-prefixed tag; empty for the initial release")
	auditPath := flags.String("audit", "", "optional audit JSON report whose source set must match")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("validate accepts no positional arguments: %q", flags.Arg(0))
	}
	if *version == "" || *directory == "" {
		return errors.New("validate requires --version and --dir")
	}
	if *auditPath == "" {
		return ValidateReleaseDirectory(*directory, *version, *previous)
	}
	report, err := ReadAuditReport(*auditPath)
	if err != nil {
		return err
	}
	return ValidateSourceCoverage(*directory, *version, *previous, report)
}

func readReleaseDocument(path, version string, sectionOrder []string) (releaseDocument, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return releaseDocument{}, err
	}
	content := strings.TrimSpace(string(body))
	heading := releaseHeadingPattern.FindStringSubmatch(content)
	if len(heading) != 2 || heading[1] != version {
		return releaseDocument{}, fmt.Errorf("must start with heading for v%s", version)
	}

	matches := sectionPattern.FindAllStringSubmatchIndex(content, -1)
	if len(matches) == 0 {
		return releaseDocument{}, errors.New("has no release-note sections")
	}
	allowed := make(map[string]int, len(sectionOrder))
	for index, name := range sectionOrder {
		allowed[name] = index
	}
	document := releaseDocument{}
	previousIndex := -1
	for index, match := range matches {
		name := strings.TrimSpace(content[match[2]:match[3]])
		order, ok := allowed[name]
		if !ok {
			return releaseDocument{}, fmt.Errorf("contains unsupported section %q", name)
		}
		if order <= previousIndex {
			return releaseDocument{}, fmt.Errorf("section %q is out of order", name)
		}
		previousIndex = order
		end := len(content)
		if index+1 < len(matches) {
			end = matches[index+1][0]
		}
		entries := bulletEntries(content[match[1]:end])
		if len(entries) == 0 {
			return releaseDocument{}, fmt.Errorf("section %q has no entries", name)
		}
		for _, entry := range entries {
			sources := sourcePattern.FindAllString(entry, -1)
			if len(sources) == 0 {
				return releaseDocument{}, fmt.Errorf("entry in section %q has no source link", name)
			}
			document.sources = append(document.sources, sources...)
		}
		document.sections = append(document.sections, releaseSection{name: name, entries: entries})
	}

	links := linkPattern.FindAllStringSubmatch(content, -1)
	if len(links) != 1 {
		return releaseDocument{}, errors.New("must contain exactly one Full Changelog link")
	}
	document.compare = links[0][1]
	document.sources = sortedUnique(document.sources)
	return document, nil
}

func bulletEntries(section string) []string {
	lines := strings.Split(section, "\n")
	entries := make([]string, 0)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "- ") {
			entries = append(entries, strings.TrimSpace(strings.TrimPrefix(line, "- ")))
		}
	}
	return entries
}

func validateCompareLink(link, version, previous string) error {
	expected := ChangelogCompareLink(version, previous)
	if link != expected {
		return fmt.Errorf("got %q, want %q", link, expected)
	}
	return nil
}

func sortedUnique(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		seen[value] = struct{}{}
	}
	result := make([]string, 0, len(seen))
	for value := range seen {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
