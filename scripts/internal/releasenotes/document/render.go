package document

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/FlanChanXwO/javdb-cli/scripts/internal/releasenotes/model"
)

// RenderPreparedReleaseDocument 将审核后的计划渲染为一份双语版本说明。
func RenderPreparedReleaseDocument(config model.PrepareConfig, plan model.PreparePlan, chinese bool) ([]byte, error) {
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
			link, err := RenderSourceLink(source)
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
			link, err := RenderSourceLink(contributor.PullURL)
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
	link := ChangelogCompareLink(config.Version, config.Previous)
	if chinese {
		fmt.Fprintf(&builder, "\n**完整变更**：[%s](%s)\n", ChangelogCompareLabel(config.Version, config.Previous), link)
	} else {
		fmt.Fprintf(&builder, "\n**Full Changelog**: [%s](%s)\n", ChangelogCompareLabel(config.Version, config.Previous), link)
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

// UpdateChangelogIndex 在 Unreleased 行之后插入版本索引行。
func UpdateChangelogIndex(path, version, previous, date string, replace bool) ([]byte, error) {
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
	row := fmt.Sprintf("| [v%s](%s) | %s | [English](v%s/en.md) · [简体中文](v%s/zh-CN.md) |\n", version, ChangelogCompareLink(version, previous), date, version, version)
	position += len(anchor)
	return []byte(content[:position] + row + content[position:]), nil
}

// RenderGitHubReleaseBody 从双语 changelog 生成稳定的 GitHub Release 正文。
func RenderGitHubReleaseBody(directory, version string) (string, error) {
	english, err := notesFromChangelog(filepath.Join(directory, "en.md"), version)
	if err != nil {
		return "", fmt.Errorf("English changelog: %w", err)
	}
	chinese, err := notesFromChangelog(filepath.Join(directory, "zh-CN.md"), version)
	if err != nil {
		return "", fmt.Errorf("Simplified Chinese changelog: %w", err)
	}
	return string(bilingualBody(english, chinese)), nil
}

// RunRender 执行 render 子命令。
func RunRender(arguments []string) error {
	flags := flag.NewFlagSet("render", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	version := flags.String("version", "", "semantic version without v")
	directory := flags.String("dir", "", "version directory containing en.md and zh-CN.md")
	output := flags.String("output", "", "optional output file; stdout when omitted")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 || !IsSemanticVersion(*version) || *directory == "" {
		return errors.New("render requires --version and --dir")
	}
	body, err := RenderGitHubReleaseBody(*directory, *version)
	if err != nil {
		return err
	}
	if *output == "" {
		_, err = fmt.Fprint(os.Stdout, body)
		return err
	}
	return os.WriteFile(*output, []byte(body), 0o644)
}

func notesFromChangelog(path, version string) ([]byte, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read changelog: %w", err)
	}
	header := regexp.MustCompile(`(?m)^# v` + regexp.QuoteMeta(version) + `(?:\s+[—-]\s+.+)?\s*$`)
	location := header.FindIndex(body)
	if location == nil {
		return nil, fmt.Errorf("changelog has no release heading for v%s", version)
	}
	if second := header.FindIndex(body[location[1]:]); second != nil {
		return nil, fmt.Errorf("changelog has more than one release heading for v%s", version)
	}
	notes := strings.Trim(string(body[location[1]:]), "\n")
	if notes == "" {
		return nil, fmt.Errorf("changelog release v%s has no release notes", version)
	}
	return []byte(notes + "\n"), nil
}

func bilingualBody(english, chinese []byte) []byte {
	return []byte("# English\n\n" + strings.TrimSpace(string(english)) + "\n\n---\n\n# 简体中文\n\n" + strings.TrimSpace(string(chinese)) + "\n")
}
