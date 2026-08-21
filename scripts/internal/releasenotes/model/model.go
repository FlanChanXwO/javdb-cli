// Package model 保存 release-note 流程跨领域传递的稳定数据模型。
package model

import "time"

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
	HTMLURL        string     `json:"html_url"`
	MergedAt       *time.Time `json:"merged_at"`
	MergeCommitSHA string     `json:"merge_commit_sha"`
	User           GitHubUser `json:"user"`
}

// AuditSource 是 audit 报告中的一个 PR 或直接 commit 来源。
type AuditSource struct {
	Kind       string `json:"kind"`
	URL        string `json:"url"`
	PullNumber int    `json:"pull_number,omitempty"`
	Commit     string `json:"commit,omitempty"`
	Title      string `json:"title"`
	Author     string `json:"author"`
	Issue      string `json:"issue,omitempty"`
}

// AuditReport 是 audit 命令的 JSON 输出模型。
type AuditReport struct {
	Repository string        `json:"repository"`
	From       string        `json:"from,omitempty"`
	To         string        `json:"to"`
	Sources    []AuditSource `json:"sources"`
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
