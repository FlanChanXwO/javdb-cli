// Package cli assembles the Cobra root command and implements the CLI entrypoint.
package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/config/paths"
	"github.com/FlanChanXwO/javdb-cli/internal/config/settings"

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
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/update/process"
)

// New builds the root command with the original persistent flags and command order.
func New(stdin io.Reader, stdout, stderr io.Writer) *cobra.Command {
	options := &invocation.RootOptions{}
	streams := invocation.NewStreams(stdin, stdout, stderr)
	command := &cobra.Command{
		Use:           "javdb",
		Short:         "JavDB app API command-line client",
		SilenceErrors: true,
		SilenceUsage:  true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if !shouldInitializeConfigForCommand(cmd) {
				return nil
			}
			// 有效 host 的语义校验失败会让命令在 RunE 中报错，不应先落盘基线配置。
			if !effectiveHostValid(options) {
				return nil
			}
			return paths.EnsureDefaultConfigFile()
		},
	}
	command.PersistentFlags().StringVar(&options.Proxy, "proxy", "", "Proxy URL (else HTTPS_PROXY/ALL_PROXY/config)")
	command.PersistentFlags().StringVar(&options.Host, "host", "", "auto|mirror|main|URL (default: config or auto)")

	// 保持原 root.go 的 AddCommand 顺序，避免 help 与 completion 输出漂移。
	command.AddCommand(authcmd.New(options, streams))
	command.AddCommand(configcmd.New(streams))
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

// shouldInitializeConfigForCommand 只让真正执行的远程命令触发首次配置创建。裸命令、help、
// version、completion 与参数校验失败在 Cobra 层已先于 PersistentPreRunE 返回；config 命令
// 与 update 不在这里创建（config 在各自 RunE 内于 key 校验后创建，update 不依赖 JavDB API）。
func shouldInitializeConfigForCommand(cmd *cobra.Command) bool {
	path := cmd.CommandPath()
	switch {
	case path == "javdb":
		// 裸命令不可运行，Cobra 直接返回 help；保留该分支防止未来为根命令加 RunE。
		return false
	case path == "javdb version":
		return false
	case strings.HasPrefix(path, "javdb config"):
		// config path/get/set 在 RunE 内于 key 校验后创建；unset 缺失文件是 no-op。
		return false
	case strings.HasPrefix(path, "javdb update"):
		// update 只做 GitHub Release 检查，不访问 JavDB App API，也不触发配置创建。
		return false
	case strings.HasPrefix(path, "javdb help"):
		return false
	case strings.HasPrefix(path, "javdb completion"), strings.HasPrefix(path, "javdb __complete"):
		return false
	default:
		return true
	}
}

// effectiveHostValid 解析 flag > env > config > default 合并后的有效 host。非法 host 的
// 语义校验失败会让命令在 RunE 中报错，首次创建 hook 应先跳过落盘，避免污染失败命令。
func effectiveHostValid(options *invocation.RootOptions) bool {
	path, err := paths.ConfigPath()
	if err != nil {
		return false
	}
	file, err := settings.LoadFile(path)
	if err != nil {
		return false
	}
	_, err = settings.Resolve(file, options.Host, options.Proxy, nil)
	return err == nil
}
