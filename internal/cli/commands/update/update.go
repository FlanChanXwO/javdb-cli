package update

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/common/jsonx"
	"github.com/FlanChanXwO/javdb-cli/internal/update/model"
)

type updateOptions struct {
	check             bool
	includePrerelease bool
	jsonOut           bool
}

// New builds the explicit update command.
func New(options *invocation.RootOptions, streams *invocation.Streams) *cobra.Command {
	var opts updateOptions
	command := &cobra.Command{
		Use:     "update",
		Short:   "Check for or install updates",
		Example: "javdb update --check",
		Args:    cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			// 安装过程可输出 Homebrew/Go 的原始进度，故 JSON 只承诺只读检查的单一对象。
			if opts.jsonOut && !opts.check {
				return fmt.Errorf("--json is only supported with --check")
			}
			proxy, err := resolveProxy(options)
			if err != nil {
				return err
			}
			coordinator, err := newProductionCoordinator(proxy, streams.Out, streams.Err)
			if err != nil {
				return err
			}
			result, err := coordinator.Execute(command.Context(), model.Request{
				BuildInfo:         buildInfo(),
				Check:             opts.check,
				IncludePrerelease: opts.includePrerelease,
			})
			if err != nil {
				return err
			}
			if opts.jsonOut {
				b, err := jsonx.MarshalLine(result)
				if err != nil {
					return err
				}
				_, err = command.OutOrStdout().Write(b)
				return err
			}
			return printUpdateResult(command.OutOrStdout(), result)
		},
	}
	commandFlags := command.Flags()
	commandFlags.BoolVar(&opts.check, "check", false, "Check for an update without installing it")
	commandFlags.BoolVar(&opts.includePrerelease, "prerelease", false, "Include prerelease updates")
	commandFlags.BoolVar(&opts.jsonOut, "json", false, "Print update check status as JSON (requires --check)")
	return command
}

// printUpdateResult 写出稳定的可读更新状态。
func printUpdateResult(out io.Writer, result model.Result) error {
	latestVersion := "none"
	if result.LatestVersion != nil {
		latestVersion = *result.LatestVersion
	}
	_, err := fmt.Fprintf(out, "source: %s\ncurrent version: %s\nlatest version: %s\nupdate available: %t\n",
		result.Source, result.CurrentVersion, latestVersion, result.UpdateAvailable)
	return err
}
