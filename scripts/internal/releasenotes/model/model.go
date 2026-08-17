// Package model 保存 release-note 流程跨领域传递的稳定数据模型。
package model

import "time"

// ReleaseNote 是 PR 模板中 release-note 注释的稳定语义模型。
// 它不包含 PR 编号，因为 audit 会将声明与 GitHub 返回的 PR 元数据关联。
type ReleaseNote struct {
	Category   string `json:"category"`
	Breaking   bool   `json:"breaking"`
	Summary    string `json:"summary"`
	NoneReason string `json:"none_reason"`
}

// GitHubUser 是 release-note 审计所需的 GitHub 用户字段。
type GitHubUser struct {
	Login   string `json:"login"`
	Type    string `json:"type"`
	HTMLURL string `json:"html_url"`
}

// GitHubPullRequest 是 release-note 审计所需的 PR 字段。
type GitHubPullRequest struct {
	Number         int        `json:"number"`
	Title          string     `json:"title"`
	Body           string     `json:"body"`
	HTMLURL        string     `json:"html_url"`
	MergedAt       *time.Time `json:"merged_at"`
	MergeCommitSHA string     `json:"merge_commit_sha"`
	User           GitHubUser `json:"user"`
}

// GitHubPullRequestSearchResult 是 GitHub search/issues 响应的最小模型。
type GitHubPullRequestSearchResult struct {
	Items []GitHubPullRequest `json:"items"`
}

// AuditSource 是 audit 报告中的一个 PR 或直接 commit 来源。
type AuditSource struct {
	Kind       string       `json:"kind"`
	URL        string       `json:"url"`
	PullNumber int          `json:"pull_number,omitempty"`
	Commit     string       `json:"commit,omitempty"`
	Title      string       `json:"title"`
	Author     string       `json:"author"`
	Note       *ReleaseNote `json:"release_note,omitempty"`
	Issue      string       `json:"issue,omitempty"`
}

// NewContributor 记录首次合并 PR 的外部贡献者。
type NewContributor struct {
	Login      string `json:"login"`
	ProfileURL string `json:"profile_url"`
	PullNumber int    `json:"pull_number"`
	PullURL    string `json:"pull_url"`
}

// AuditReport 是 audit 命令的 JSON 输出模型。
type AuditReport struct {
	Repository             string           `json:"repository"`
	From                   string           `json:"from,omitempty"`
	To                     string           `json:"to"`
	RecommendedVersionBump string           `json:"recommended_version_bump"`
	Sources                []AuditSource    `json:"sources"`
	NewContributors        []NewContributor `json:"new_contributors"`
}

// PreparePlan 是审核后、可纳入 release-prep 流程的编辑输入。
// 工具刻意不从 PR 标题自动生成面向用户的文案：维护者需要在此处提供双语摘要。
type PreparePlan struct {
	Version         string           `json:"version"`
	PreviousTag     *string          `json:"previous_tag"`
	CompareURL      *string          `json:"compare_url"`
	Entries         []PreparedEntry  `json:"entries"`
	NewContributors []NewContributor `json:"new_contributors,omitempty"`
}

// PreparedEntry 是发布计划中的一条双语变更说明。
type PreparedEntry struct {
	Category string   `json:"category"`
	Breaking bool     `json:"breaking,omitempty"`
	English  string   `json:"english"`
	Chinese  string   `json:"zh_cn"`
	Sources  []string `json:"sources"`
}

// PrepareConfig 是 prepare 命令的运行配置。
type PrepareConfig struct {
	Version       string
	Previous      string
	Date          string
	ChangelogRoot string
	PlanPath      string
	AuditPath     string
	Apply         bool
	Replace       bool
}

// GitHubRelease 是历史 Release 同步所需的 GitHub Release 字段。
type GitHubRelease struct {
	ID      int                  `json:"id"`
	TagName string               `json:"tag_name"`
	Name    string               `json:"name"`
	Body    string               `json:"body"`
	Assets  []GitHubReleaseAsset `json:"assets"`
}

// GitHubReleaseAsset 是历史 Release 中用于回归确认的资产字段。
type GitHubReleaseAsset struct {
	Name string `json:"name"`
}

// GitHubReleaseWrite 是创建或更新历史 Release 时提交的 JSON 模型。
type GitHubReleaseWrite struct {
	TagName string `json:"tag_name,omitempty"`
	Name    string `json:"name,omitempty"`
	Body    string `json:"body"`
	Draft   bool   `json:"draft"`
}
