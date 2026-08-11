// Package model 保存更新流程跨领域共享的数据模型。
package model

import "github.com/FlanChanXwO/javdb-cli/internal/buildinfo"

const (
	// GitHubRepository 是官方 GitHub 仓库标识。
	GitHubRepository = "FlanChanXwO/javdb-cli"
	// GoInstallPackage 是 go install 更新使用的模块路径。
	GoInstallPackage = "github.com/FlanChanXwO/javdb-cli/cmd/javdb"
	// HomebrewFormula 是 Homebrew 更新使用的 formula 标识。
	HomebrewFormula = "FlanChanXwO/tap/javdb-cli"
)

// InstallSource describes the package manager or artifact that owns javdb.
type InstallSource string

const (
	InstallSourceDevelopment InstallSource = "development"
	InstallSourceHomebrew    InstallSource = "homebrew"
	InstallSourceGoInstall   InstallSource = "go-install"
	InstallSourceRelease     InstallSource = "release"
)

// ReleaseAsset is the subset of GitHub Release asset metadata used by updater.
type ReleaseAsset struct {
	Name        string `json:"name"`
	DownloadURL string `json:"browser_download_url"`
}

// Release is a verified candidate returned by the GitHub Releases API.
type Release struct {
	TagName    string         `json:"tag_name"`
	Prerelease bool           `json:"prerelease"`
	Assets     []ReleaseAsset `json:"assets"`
}

// Request 描述一次显式 javdb update 调用。
type Request struct {
	BuildInfo         buildinfo.Info
	Check             bool
	IncludePrerelease bool
}

// Result 是人类输出和 JSON 检查结果共用的稳定状态模型。
type Result struct {
	Source           InstallSource `json:"source"`
	CurrentVersion   string        `json:"current_version"`
	LatestVersion    *string       `json:"latest_version"`
	LatestPrerelease bool          `json:"latest_prerelease"`
	UpdateAvailable  bool          `json:"update_available"`
}
