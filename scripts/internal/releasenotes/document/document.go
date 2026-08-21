// Package document 负责双语 release-note 文档、来源链接和 Release 正文校验。
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
	"strings"

	"github.com/FlanChanXwO/javdb-cli/scripts/internal/releasenotes/model"
)

var (
	releaseHeadingPattern  = regexp.MustCompile(`(?m)^# v([0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?)\s+[—-]\s+.+$`)
	sectionPattern         = regexp.MustCompile(`(?m)^##\s+(.+?)\s*$`)
	sourcePattern          = regexp.MustCompile(`https://github\.com/FlanChanXwO/javdb-cli/(?:pull/[0-9]+|commit/[0-9a-fA-F]{7,64})`)
	linkPattern            = regexp.MustCompile(`\[[^\]]+\]\((https://github\.com/FlanChanXwO/javdb-cli/(?:compare/[^)\s]+|commits/[^)\s]+))\)`)
	semanticVersionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?$`)
)

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

// ValidateSourceCoverage 将 changelog 中的来源逐项对照审计报告。
// 发布说明是人工维护的唯一内容来源，因此允许审计区间内存在未列入 notes 的来源；
// 但 notes 不能引用区间之外或无法解析的 PR/commit。
func ValidateSourceCoverage(directory, version, previous string, report model.AuditReport) error {
	if err := ValidateReleaseDirectory(directory, version, previous); err != nil {
		return err
	}
	english, err := readReleaseDocument(filepath.Join(directory, "en.md"), version, englishSectionOrder)
	if err != nil {
		return err
	}
	present := make(map[string]struct{}, len(report.Sources))
	for _, source := range report.Sources {
		present[source.URL] = struct{}{}
	}
	for _, source := range english.sources {
		if _, ok := present[source]; !ok {
			return fmt.Errorf("release notes source %s is not present in the audit report", source)
		}
	}
	return nil
}

// ReadAuditReport 读取 validate 共用的 audit JSON 报告。
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
	auditPath := flags.String("audit", "", "optional audit JSON report whose source set must contain all changelog sources")
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
