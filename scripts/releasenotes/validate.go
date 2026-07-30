// 校验负责离线 changelog 契约与 PR release-note 声明解析。
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func runValidate(arguments []string) error {
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
		return validateReleaseDirectory(*directory, *version, *previous)
	}
	report, err := readAuditReport(*auditPath)
	if err != nil {
		return err
	}
	return validateSourceCoverage(*directory, *version, *previous, report)
}

// validateReleaseDirectory 检查可发布的双语版本文件。它不依赖网络，因此既可用于
// 文档 CI，也可用于不可变 tag 的发布前门禁。
func validateReleaseDirectory(directory, version, previous string) error {
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

// validateSourceCoverage 将已经渲染的说明与审计报告逐项对照。它既能发现漏记的
// PR/历史 direct commit，也拒绝无法由该版本审计报告解释的额外来源链接。
func validateSourceCoverage(directory, version, previous string, report auditReport) error {
	if err := validateReleaseDirectory(directory, version, previous); err != nil {
		return err
	}
	english, err := readReleaseDocument(filepath.Join(directory, "en.md"), version, englishSectionOrder)
	if err != nil {
		return err
	}
	expected := make(map[string]struct{})
	for _, source := range report.Sources {
		if source.Kind == "pull_request" {
			if source.Note == nil || source.Issue != "" {
				return fmt.Errorf("audit source %s is not classified: %s", source.URL, source.Issue)
			}
			if source.Note.Category == "None" {
				continue
			}
		}
		expected[source.URL] = struct{}{}
	}
	for _, contributor := range report.NewContributors {
		expected[contributor.PullURL] = struct{}{}
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
		if _, ok := expected[source]; !ok {
			return fmt.Errorf("release notes source %s is not present in the audit report", source)
		}
	}
	return nil
}

var englishSectionOrder = []string{
	"Breaking changes",
	"Added",
	"Changed",
	"Fixed",
	"Security",
	"Documentation",
	"Maintenance",
	"New Contributors",
}

var chineseSectionOrder = []string{
	"破坏性变更",
	"新增",
	"变更",
	"修复",
	"安全",
	"文档",
	"维护",
	"新贡献者",
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
	if previous == "" {
		expected := "https://github.com/FlanChanXwO/javdb-cli/commits/v" + version
		if link != expected {
			return fmt.Errorf("got %q, want %q", link, expected)
		}
		return nil
	}
	expected := "https://github.com/FlanChanXwO/javdb-cli/compare/" + previous + "...v" + version
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

// parseReleaseNoteDeclaration 读取 PR 正文中的机器可读注释。注释不直接显示在
// GitHub 页面上，既避免重复面向读者的描述，也能让 CI 在离线事件载荷中稳定校验。
func parseReleaseNoteDeclaration(body string) (releaseNote, error) {
	match := releaseNotePattern.FindStringSubmatch(body)
	if len(match) != 2 {
		return releaseNote{}, errors.New("missing release-note declaration")
	}
	values := make(map[string]string)
	for _, rawLine := range strings.Split(match[1], "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return releaseNote{}, fmt.Errorf("invalid release-note line %q", line)
		}
		key = strings.TrimSpace(key)
		if _, exists := values[key]; exists {
			return releaseNote{}, fmt.Errorf("release-note repeats %q", key)
		}
		values[key] = strings.TrimSpace(value)
	}
	for _, key := range []string{"category", "breaking", "summary", "none_reason"} {
		if _, ok := values[key]; !ok {
			return releaseNote{}, fmt.Errorf("release-note is missing %s", key)
		}
	}
	for key := range values {
		switch key {
		case "category", "breaking", "summary", "none_reason":
		default:
			return releaseNote{}, fmt.Errorf("release-note contains unsupported key %q", key)
		}
	}
	if _, ok := releaseNoteCategories[values["category"]]; !ok {
		return releaseNote{}, fmt.Errorf("release-note has unsupported category %q", values["category"])
	}
	breaking, err := strconv.ParseBool(values["breaking"])
	if err != nil {
		return releaseNote{}, fmt.Errorf("release-note breaking: %w", err)
	}
	note := releaseNote{
		Category:   values["category"],
		Breaking:   breaking,
		Summary:    values["summary"],
		NoneReason: values["none_reason"],
	}
	if note.Summary == "" {
		return releaseNote{}, errors.New("release-note summary is required")
	}
	if note.Category == "None" {
		if note.Breaking {
			return releaseNote{}, errors.New("release-note None category cannot be breaking")
		}
		if note.NoneReason == "" {
			return releaseNote{}, errors.New("release-note none_reason is required for category None")
		}
	} else if note.NoneReason != "" {
		return releaseNote{}, errors.New("release-note none_reason is only valid for category None")
	}
	return note, nil
}

var releaseNoteCategories = map[string]struct{}{
	"Added":         {},
	"Changed":       {},
	"Fixed":         {},
	"Security":      {},
	"Documentation": {},
	"Maintenance":   {},
	"None":          {},
}

func recommendedVersionBump(notes []releaseNote) string {
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
