// Command releasenotes audits and validates the versioned changelog contract.
package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"time"
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

// releaseNote 是 PR 模板中 release-note 注释的稳定语义模型。它不包含 PR 编号，
// 因为 audit 会将同一声明与 GitHub 返回的 PR 元数据关联。
type releaseNote struct {
	Category   string `json:"category"`
	Breaking   bool   `json:"breaking"`
	Summary    string `json:"summary"`
	NoneReason string `json:"none_reason"`
}

// githubClient 只封装 release-note 流程需要的 GitHub REST 读取。写入历史
// Release 的操作在 sync-history 子命令中单独、显式地处理。
type githubClient struct {
	baseURL string
	token   string
	client  *http.Client
}

type githubUser struct {
	Login   string `json:"login"`
	Type    string `json:"type"`
	HTMLURL string `json:"html_url"`
}

type githubPullRequest struct {
	Number   int        `json:"number"`
	Title    string     `json:"title"`
	Body     string     `json:"body"`
	HTMLURL  string     `json:"html_url"`
	MergedAt *time.Time `json:"merged_at"`
	User     githubUser `json:"user"`
}

type githubPullRequestSearchResult struct {
	Items []githubPullRequest `json:"items"`
}

type auditSource struct {
	Kind       string       `json:"kind"`
	URL        string       `json:"url"`
	PullNumber int          `json:"pull_number,omitempty"`
	Commit     string       `json:"commit,omitempty"`
	Title      string       `json:"title"`
	Author     string       `json:"author"`
	Note       *releaseNote `json:"release_note,omitempty"`
	Issue      string       `json:"issue,omitempty"`
}

type newContributor struct {
	Login      string `json:"login"`
	ProfileURL string `json:"profile_url"`
	PullNumber int    `json:"pull_number"`
	PullURL    string `json:"pull_url"`
}

type auditReport struct {
	Repository             string           `json:"repository"`
	From                   string           `json:"from,omitempty"`
	To                     string           `json:"to"`
	RecommendedVersionBump string           `json:"recommended_version_bump"`
	Sources                []auditSource    `json:"sources"`
	NewContributors        []newContributor `json:"new_contributors"`
}

// preparePlan 是审核后、可纳入 release-prep PR 的编辑输入。工具刻意不从 PR
// 标题自动生成面向用户的文案：维护者需要在此处合并相关变更并提供双语摘要。
type preparePlan struct {
	Version         string           `json:"version"`
	PreviousTag     *string          `json:"previous_tag"`
	CompareURL      *string          `json:"compare_url"`
	Entries         []preparedEntry  `json:"entries"`
	NewContributors []newContributor `json:"new_contributors,omitempty"`
}

type preparedEntry struct {
	Category string   `json:"category"`
	Breaking bool     `json:"breaking,omitempty"`
	English  string   `json:"english"`
	Chinese  string   `json:"zh_cn"`
	Sources  []string `json:"sources"`
}

type prepareConfig struct {
	Version       string
	Previous      string
	Date          string
	ChangelogRoot string
	PlanPath      string
	AuditPath     string
	Apply         bool
	Replace       bool
}

type syncHistoryConfig struct {
	Repository string
	Version    string
	Directory  string
	Client     githubClient
	Apply      bool
}

type githubRelease struct {
	ID      int                  `json:"id"`
	TagName string               `json:"tag_name"`
	Name    string               `json:"name"`
	Body    string               `json:"body"`
	Assets  []githubReleaseAsset `json:"assets"`
}

type githubReleaseAsset struct {
	Name string `json:"name"`
}

type githubReleaseWrite struct {
	TagName string `json:"tag_name,omitempty"`
	Name    string `json:"name,omitempty"`
	Body    string `json:"body"`
	Draft   bool   `json:"draft"`
}

type releaseDocument struct {
	sections []releaseSection
	sources  []string
	compare  string
}

type releaseSection struct {
	name    string
	entries []string
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(arguments []string) error {
	if len(arguments) == 0 {
		return errors.New("a subcommand is required: validate, audit, prepare, render, pr-validate, or sync-history")
	}
	switch arguments[0] {
	case "validate":
		return runValidate(arguments[1:])
	case "audit":
		return runAudit(arguments[1:])
	case "prepare":
		return runPrepare(arguments[1:])
	case "render":
		return runRender(arguments[1:])
	case "pr-validate":
		return runPullRequestValidate(arguments[1:])
	case "sync-history":
		return runSyncHistory(arguments[1:])
	case "-h", "--help", "help":
		return errors.New("usage: releasenotes validate|audit|prepare|render|pr-validate|sync-history")
	default:
		return fmt.Errorf("unknown subcommand %q", arguments[0])
	}
}

// runRender 输出 GitHub Release 所用的稳定双语正文，使发布 workflow 与历史同步共享完全相同的渲染。
