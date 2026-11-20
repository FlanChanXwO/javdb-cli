// Package cli assembles the Cobra root command and implements the CLI entrypoint.
package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/FlanChanXwO/javdb-cli/internal/buildinfo"
	"github.com/FlanChanXwO/javdb-cli/internal/environment"
	"github.com/FlanChanXwO/javdb-cli/internal/i18n"
	"github.com/FlanChanXwO/javdb-cli/internal/logging"
	"github.com/FlanChanXwO/javdb-cli/internal/session"
	"github.com/FlanChanXwO/javdb-cli/internal/update/model"
)

// New builds the root command with the original persistent flags and command order.
func New(stdin io.Reader, stdout, stderr io.Writer) *cobra.Command {
	command, _ := newRoot(stdin, stdout, stderr)
	return command
}

func newRoot(stdin io.Reader, stdout, stderr io.Writer) (*cobra.Command, *environment.Streams) {
	options := &session.Options{}
	var logLevel, logFormat string
	streams := environment.NewStreams(stdin, stdout, stderr)
	streams.Logger, _ = logging.New(stderr, "info", "text")
	// 非 TTY stdin 是管道输入信号；探测只在 stdin 是真实终端时生效。
	if file, ok := stdin.(*os.File); ok && term.IsTerminal(int(file.Fd())) {
		streams.InIsTerminal = true
	}
	// 非 TTY stdout 是管道/记录输出信号；探测只在 stdout 是真实终端时生效。
	if file, ok := stdout.(*os.File); ok && term.IsTerminal(int(file.Fd())) {
		streams.OutIsTerminal = true
	}
	command := &cobra.Command{
		Use:           "javdb",
		Short:         i18n.T("command.root"),
		SilenceErrors: true,
		SilenceUsage:  true,
		Version:       rootVersionLine(),
	}
	command.PersistentFlags().StringVar(&logLevel, "log-level", "", i18n.T("log.level_help"))
	command.PersistentFlags().StringVar(&logFormat, "log-format", "", i18n.T("log.format_help"))
	command.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		err := configureLogging(cmd, streams)
		options.Logger = streams.Logger
		return err
	}
	command.SetHelpTemplate("{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}\n\n{{end}}{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}")
	command.SetUsageTemplate(usageTemplate)
	command.SetVersionTemplate("{{.Version}}\n")
	command.PersistentFlags().StringVar(&options.Proxy, "proxy", "", i18n.T("flag.proxy"))
	command.PersistentFlags().StringVar(&options.Host, "host", "", i18n.T("flag.host"))

	addCommands(command, options, streams)
	command.InitDefaultHelpFlag()
	command.InitDefaultVersionFlag()
	command.InitDefaultHelpCmd()
	command.InitDefaultCompletionCmd()
	localizeBuiltinHelp(command)
	return command, streams
}

// rootVersionLine 构造根 `--version` 的展示行：
//   - 正式版（带 ReleaseDate）：`javdb version 0.7.0 (2026-08-12)` 两行，
//     第二行为 Release URL，展示版本不带 v。
//   - 开发版：单行 `javdb version dev (commit <commit>, built <timestamp>)`，
//     不显示 Release URL。
func rootVersionLine() string {
	info := buildinfo.Current()
	if info.IsDevelopment() {
		return fmt.Sprintf("javdb version dev (commit %s, built %s)", info.Commit, info.BuildDate)
	}
	displayVersion := strings.TrimPrefix(info.Version, "v")
	line := fmt.Sprintf("javdb version %s", displayVersion)
	if info.ReleaseDate != "" {
		line += fmt.Sprintf(" (%s)", info.ReleaseDate)
	}
	if info.ReleaseDate != "" && !info.IsDevelopment() {
		line += "\n" + "https://github.com/" + model.GitHubRepository + "/releases/tag/" + info.Version
	}
	return line
}
