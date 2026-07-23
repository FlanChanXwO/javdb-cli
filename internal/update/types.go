// Package update implements the explicit javdb self-update workflow.
package update

const (
	githubRepository = "FlanChanXwO/javdb-cli"
	goInstallPackage = "github.com/FlanChanXwO/javdb-cli/cmd/javdb"
	homebrewFormula  = "FlanChanXwO/tap/javdb-cli"
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
