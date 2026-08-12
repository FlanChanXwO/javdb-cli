// Package cli assembles the Cobra root command and implements the CLI entrypoint.
package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	actorcmd "github.com/FlanChanXwO/javdb-cli/internal/cli/commands/actor"
	authcmd "github.com/FlanChanXwO/javdb-cli/internal/cli/commands/auth"
	browsecmd "github.com/FlanChanXwO/javdb-cli/internal/cli/commands/browse"
	cachecmd "github.com/FlanChanXwO/javdb-cli/internal/cli/commands/cache"
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
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/update/process"
)

// New builds the root command with the original persistent flags and command order.
func New(stdin io.Reader, stdout, stderr io.Writer) *cobra.Command {
	options := &invocation.RootOptions{}
	streams := invocation.NewStreams(stdin, stdout, stderr)
	// 非 TTY stdin 是管道输入信号；探测只在 stdin 是真实终端时生效。
	if file, ok := stdin.(*os.File); ok && term.IsTerminal(int(file.Fd())) {
		streams.InIsTerminal = true
	}
	command := &cobra.Command{
		Use:           "javdb",
		Short:         "JavDB app API command-line client",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	command.PersistentFlags().StringVar(&options.Proxy, "proxy", "", "Proxy URL (else HTTPS_PROXY/ALL_PROXY/config)")
	command.PersistentFlags().StringVar(&options.Host, "host", "", "auto|mirror|main|URL (default: config or auto)")

	// 保持原 root.go 的 AddCommand 顺序，避免 help 与 completion 输出漂移。
	command.AddCommand(authcmd.New(options, streams))
	command.AddCommand(configcmd.New(streams))
	command.AddCommand(cachecmd.New(streams))
	command.AddCommand(searchcmd.New(options, streams))
	command.AddCommand(detailcmd.New(options, streams))
	command.AddCommand(commentscmd.New(options, streams))
	command.AddCommand(magnetscmd.New(options, streams))
	command.AddCommand(downloadcmd.New(options, streams))
	command.AddCommand(tagscmd.New(options, streams))
	command.AddCommand(browsecmd.New(options, streams))
	command.AddCommand(actorcmd.New(options, streams))
	command.AddCommand(seriescmd.New(options, streams))
	command.AddCommand(makercmd.New(options, streams))
	command.AddCommand(directorcmd.New(options, streams))
	command.AddCommand(codecmd.New(options, streams))
	command.AddCommand(listcmd.New(options, streams))
	command.AddCommand(watchedcmd.New(options, streams))
	command.AddCommand(wantcmd.New(options, streams))
	command.AddCommand(recentcmd.New(options, streams))
	command.AddCommand(collectionscmd.New(options, streams))
	command.AddCommand(markcmd.New(options, streams))
	command.AddCommand(unmarkcmd.New(options, streams))
	command.AddCommand(rankingscmd.New(options, streams))
	command.AddCommand(top250cmd.New(options, streams))
	command.AddCommand(listscmd.New(options, streams))
	command.AddCommand(updatecmd.New(options, streams))
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
