package update

import (
	"fmt"
	"io"

	"github.com/FlanChanXwO/javdb-cli/internal/buildinfo"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/config/paths"
	"github.com/FlanChanXwO/javdb-cli/internal/config/settings"
	"github.com/FlanChanXwO/javdb-cli/internal/update"
	"github.com/FlanChanXwO/javdb-cli/internal/update/archive"
	"github.com/FlanChanXwO/javdb-cli/internal/update/process"
	"github.com/FlanChanXwO/javdb-cli/internal/update/release"
	"github.com/FlanChanXwO/javdb-cli/internal/update/source"
)

// resolveProxy 只解析 GitHub Release 流程使用的 proxy（与 JavDB host 无关）。
func resolveProxy(options *invocation.RootOptions) (string, error) {
	path, err := paths.ConfigPath()
	if err != nil {
		return "", fmt.Errorf("resolve update configuration path: %w", err)
	}
	cfg, err := settings.LoadFile(path)
	if err != nil {
		return "", fmt.Errorf("load update configuration: %w", err)
	}
	// update 访问 GitHub Releases，与 JavDB host 无关；只沿用同一 proxy 优先级。
	proxy, err := settings.ResolveProxy(cfg, options.Proxy)
	if err != nil {
		return "", fmt.Errorf("resolve update configuration: %w", err)
	}
	return proxy, nil
}

// newProductionCoordinator 组装显式、可离线测试的 update 依赖。
func newProductionCoordinator(proxy string, stdout, stderr io.Writer) (*update.Coordinator, error) {
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

// buildInfo 返回 update 命令使用的当前构建元数据。
func buildInfo() buildinfo.Info { return buildinfo.Current() }
