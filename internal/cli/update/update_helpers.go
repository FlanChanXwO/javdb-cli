package update

import (
	"fmt"
	"io"

	"github.com/FlanChanXwO/javdb-cli/internal/buildinfo"
	"github.com/FlanChanXwO/javdb-cli/internal/config/paths"
	"github.com/FlanChanXwO/javdb-cli/internal/config/settings"
	"github.com/FlanChanXwO/javdb-cli/internal/i18n"
	"github.com/FlanChanXwO/javdb-cli/internal/session"
	"github.com/FlanChanXwO/javdb-cli/internal/update"
	"github.com/FlanChanXwO/javdb-cli/internal/update/archive"
	"github.com/FlanChanXwO/javdb-cli/internal/update/process"
	"github.com/FlanChanXwO/javdb-cli/internal/update/release"
	"github.com/FlanChanXwO/javdb-cli/internal/update/source"
)

// resolveProxy 只解析 GitHub Release 流程使用的 proxy（与 JavDB host 无关）。
func resolveProxy(options *session.Options) (string, error) {
	path, err := paths.ConfigPath()
	if err != nil {
		return "", fmt.Errorf(i18n.T("cli.resolve_update_configuration_path_w"), err)
	}
	cfg, err := settings.LoadFile(path)
	if err != nil {
		return "", fmt.Errorf(i18n.T("cli.load_update_configuration_w"), err)
	}
	// update 访问 GitHub Releases，与 JavDB host 无关；只沿用同一 proxy 优先级。
	proxy, err := settings.ResolveProxy(cfg, options.Proxy)
	if err != nil {
		return "", fmt.Errorf(i18n.T("cli.resolve_update_configuration_w"), err)
	}
	return proxy, nil
}

// newProductionCoordinator 组装显式、可离线测试的 update 依赖。
func newProductionCoordinator(proxy string, stdout, stderr io.Writer) (*update.Coordinator, error) {
	httpClient, err := release.NewReleaseHTTPClient(proxy)
	if err != nil {
		return nil, fmt.Errorf(i18n.T("cli.create_update_http_client_w"), err)
	}
	releaseClient, err := release.NewGitHubReleaseClient(release.ReleaseClientOptions{HTTPClient: httpClient})
	if err != nil {
		return nil, fmt.Errorf(i18n.T("cli.create_github_release_client_w"), err)
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
