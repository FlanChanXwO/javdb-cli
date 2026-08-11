package app

import (
	"fmt"
	"io"

	"github.com/FlanChanXwO/javdb-cli/internal/buildinfo"
	"github.com/FlanChanXwO/javdb-cli/internal/config/paths"
	"github.com/FlanChanXwO/javdb-cli/internal/config/settings"
	"github.com/FlanChanXwO/javdb-cli/internal/update"
	"github.com/FlanChanXwO/javdb-cli/internal/update/archive"
	"github.com/FlanChanXwO/javdb-cli/internal/update/process"
	"github.com/FlanChanXwO/javdb-cli/internal/update/release"
	"github.com/FlanChanXwO/javdb-cli/internal/update/source"
)

// ResolveUpdateProxy resolves only the proxy used by the GitHub release flow.
func ResolveUpdateProxy(flags *Flags) (string, error) {
	path, err := paths.ConfigPath()
	if err != nil {
		return "", fmt.Errorf("resolve update configuration path: %w", err)
	}
	cfg, err := settings.LoadFile(path)
	if err != nil {
		return "", fmt.Errorf("load update configuration: %w", err)
	}
	// update 访问 GitHub Releases，与 JavDB host 无关；沿用同一 proxy 优先级。
	return settings.Resolve(cfg, "", flags.Proxy, nil).Proxy, nil
}

// NewProductionUpdateCoordinator assembles the explicit offline-testable update dependencies.
func NewProductionUpdateCoordinator(proxy string, stdout, stderr io.Writer) (*update.Coordinator, error) {
	httpClient, err := release.NewReleaseHTTPClient(proxy)
	if err != nil {
		return nil, fmt.Errorf("create update HTTP client: %w", err)
	}
	releaseClient, err := release.NewGitHubReleaseClient(release.ReleaseClientOptions{HTTPClient: httpClient})
	if err != nil {
		return nil, fmt.Errorf("create GitHub Release client: %w", err)
	}
	return update.NewCoordinator(update.CoordinatorOptions{
		SourceDetector:   update.SourceDetectorFunc(source.DetectInstallSource),
		ReleaseChecker:   releaseClient,
		CommandRunner:    process.NewCommandRunner(stdout, stderr),
		ReleaseInstaller: archive.NewReleaseInstaller(archive.ReleaseInstallerOptions{HTTPClient: httpClient}),
	})
}

// BuildInfo returns the current build metadata for update commands.
func BuildInfo() buildinfo.Info { return buildinfo.Current() }
