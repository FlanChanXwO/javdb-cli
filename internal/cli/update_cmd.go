package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/buildinfo"
	"github.com/FlanChanXwO/javdb-cli/internal/config"
	"github.com/FlanChanXwO/javdb-cli/internal/update"
)

type updateOptions struct {
	check             bool
	includePrerelease bool
	jsonOut           bool
}

func newUpdateCmd(rf *rootFlags, aio *appIO) *cobra.Command {
	var options updateOptions
	command := &cobra.Command{
		Use:     "update",
		Short:   "Check for or install updates",
		Example: "javdb update --check",
		Args:    cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			// 安装过程可输出 Homebrew/Go 的原始进度，故 JSON 只承诺只读检查的单一对象。
			if options.jsonOut && !options.check {
				return fmt.Errorf("--json is only supported with --check")
			}
			proxy, err := resolveUpdateProxy(rf)
			if err != nil {
				return err
			}
			coordinator, err := newProductionUpdateCoordinator(proxy, aio.out, aio.err)
			if err != nil {
				return err
			}
			result, err := coordinator.Execute(command.Context(), update.Request{
				BuildInfo:         buildinfo.Current(),
				Check:             options.check,
				IncludePrerelease: options.includePrerelease,
			})
			if err != nil {
				return err
			}
			if options.jsonOut {
				return EmitJSON(command.OutOrStdout(), result)
			}
			return printUpdateResult(command.OutOrStdout(), result)
		},
	}
	flags := command.Flags()
	flags.BoolVar(&options.check, "check", false, "Check for an update without installing it")
	flags.BoolVar(&options.includePrerelease, "prerelease", false, "Include prerelease updates")
	flags.BoolVar(&options.jsonOut, "json", false, "Print update check status as JSON (requires --check)")
	return command
}

func resolveUpdateProxy(rf *rootFlags) (string, error) {
	path, err := config.ConfigPath()
	if err != nil {
		return "", fmt.Errorf("resolve update configuration path: %w", err)
	}
	settings, err := config.LoadFile(path)
	if err != nil {
		return "", fmt.Errorf("load update configuration: %w", err)
	}
	// update 访问 GitHub Releases，与 JavDB host 无关；但沿用同一 proxy 优先级。
	return config.Resolve(settings, "", rf.proxy, nil).Proxy, nil
}

func newProductionUpdateCoordinator(proxy string, stdout, stderr io.Writer) (*update.Coordinator, error) {
	httpClient, err := update.NewReleaseHTTPClient(proxy)
	if err != nil {
		return nil, fmt.Errorf("create update HTTP client: %w", err)
	}
	releaseClient, err := update.NewGitHubReleaseClient(update.ReleaseClientOptions{HTTPClient: httpClient})
	if err != nil {
		return nil, fmt.Errorf("create GitHub Release client: %w", err)
	}
	return update.NewCoordinator(update.CoordinatorOptions{
		SourceDetector:   update.SourceDetectorFunc(update.DetectInstallSource),
		ReleaseChecker:   releaseClient,
		CommandRunner:    update.NewCommandRunner(stdout, stderr),
		ReleaseInstaller: update.NewReleaseInstaller(update.ReleaseInstallerOptions{HTTPClient: httpClient}),
	})
}

func printUpdateResult(out io.Writer, result update.Result) error {
	latestVersion := "none"
	if result.LatestVersion != nil {
		latestVersion = *result.LatestVersion
	}
	_, err := fmt.Fprintf(out, "source: %s\ncurrent version: %s\nlatest version: %s\nupdate available: %t\n", result.Source, result.CurrentVersion, latestVersion, result.UpdateAvailable)
	return err
}
