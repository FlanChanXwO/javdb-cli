package document

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

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
