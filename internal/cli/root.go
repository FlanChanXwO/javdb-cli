// Package cli assembles the Cobra root command and implements the CLI entrypoint.
package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/app"
	actorcmd "github.com/FlanChanXwO/javdb-cli/internal/cli/commands/actor"
	authcmd "github.com/FlanChanXwO/javdb-cli/internal/cli/commands/auth"
	browsecmd "github.com/FlanChanXwO/javdb-cli/internal/cli/commands/browse"
	codecmd "github.com/FlanChanXwO/javdb-cli/internal/cli/commands/code"
	collectionscmd "github.com/FlanChanXwO/javdb-cli/internal/cli/commands/collections"
	commentscmd "github.com/FlanChanXwO/javdb-cli/internal/cli/commands/comments"
	configcmd "github.com/FlanChanXwO/javdb-cli/internal/cli/commands/config"
	detailcmd "github.com/FlanChanXwO/javdb-cli/internal/cli/commands/detail"
	directorcmd "github.com/FlanChanXwO/javdb-cli/internal/cli/commands/director"
	downloadcmd "github.com/FlanChanXwO/javdb-cli/internal/cli/commands/download"
	listcmd "github.com/FlanChanXwO/javdb-cli/internal/cli/commands/list"
	listscmd "github.com/FlanChanXwO/javdb-cli/internal/cli/commands/lists"
	magnetscmd "github.com/FlanChanXwO/javdb-cli/internal/cli/commands/magnets"
	makercmd "github.com/FlanChanXwO/javdb-cli/internal/cli/commands/maker"
	markcmd "github.com/FlanChanXwO/javdb-cli/internal/cli/commands/mark"
	rankingscmd "github.com/FlanChanXwO/javdb-cli/internal/cli/commands/rankings"
	recentcmd "github.com/FlanChanXwO/javdb-cli/internal/cli/commands/recent"
	searchcmd "github.com/FlanChanXwO/javdb-cli/internal/cli/commands/search"
	seriescmd "github.com/FlanChanXwO/javdb-cli/internal/cli/commands/series"
	tagscmd "github.com/FlanChanXwO/javdb-cli/internal/cli/commands/tags"
	top250cmd "github.com/FlanChanXwO/javdb-cli/internal/cli/commands/top250"
	unmarkcmd "github.com/FlanChanXwO/javdb-cli/internal/cli/commands/unmark"
	updatecmd "github.com/FlanChanXwO/javdb-cli/internal/cli/commands/update"
	versioncmd "github.com/FlanChanXwO/javdb-cli/internal/cli/commands/version"
	wantcmd "github.com/FlanChanXwO/javdb-cli/internal/cli/commands/want"
	watchedcmd "github.com/FlanChanXwO/javdb-cli/internal/cli/commands/watched"
	"github.com/FlanChanXwO/javdb-cli/internal/update/process"
)

// New builds the root command with the original persistent flags and command order.
func New(stdin io.Reader, stdout, stderr io.Writer) *cobra.Command {
	flags := &app.Flags{}
	aio := app.NewIO(stdin, stdout, stderr)
	command := &cobra.Command{
		Use:           "javdb",
		Short:         "JavDB app API command-line client",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	command.PersistentFlags().StringVar(&flags.Proxy, "proxy", "", "Proxy URL (else HTTPS_PROXY/ALL_PROXY/config)")
	command.PersistentFlags().StringVar(&flags.Host, "host", "", "mirror|main (default: config or mirror)")

	// 保持原 root.go 的 AddCommand 顺序，避免 help 与 completion 输出漂移。
	command.AddCommand(authcmd.New(flags, aio))
	command.AddCommand(configcmd.New(aio))
	command.AddCommand(searchcmd.New(flags, aio))
	command.AddCommand(detailcmd.New(flags, aio))
	command.AddCommand(commentscmd.New(flags, aio))
	command.AddCommand(magnetscmd.New(flags, aio))
	command.AddCommand(downloadcmd.New(flags, aio))
	command.AddCommand(tagscmd.New(flags, aio))
	command.AddCommand(browsecmd.New(flags, aio))
	command.AddCommand(actorcmd.New(flags, aio))
	command.AddCommand(seriescmd.New(flags, aio))
	command.AddCommand(makercmd.New(flags, aio))
	command.AddCommand(directorcmd.New(flags, aio))
	command.AddCommand(codecmd.New(flags, aio))
	command.AddCommand(listcmd.New(flags, aio))
	command.AddCommand(watchedcmd.New(flags, aio))
	command.AddCommand(wantcmd.New(flags, aio))
	command.AddCommand(recentcmd.New(flags, aio))
	command.AddCommand(collectionscmd.New(flags, aio))
	command.AddCommand(markcmd.New(flags, aio))
	command.AddCommand(unmarkcmd.New(flags, aio))
	command.AddCommand(rankingscmd.New(flags, aio))
	command.AddCommand(top250cmd.New(flags, aio))
	command.AddCommand(listscmd.New(flags, aio))
	command.AddCommand(updatecmd.New(flags, aio))
	command.AddCommand(versioncmd.New())
	return command
}

// Run executes the command and preserves the original stderr/exit-code contract.
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if err := process.CleanupPendingWindowsUpdate(); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	command := New(stdin, stdout, stderr)
	command.SetArgs(args)
	command.SetIn(stdin)
	command.SetOut(stdout)
	command.SetErr(stderr)
	if err := command.Execute(); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
