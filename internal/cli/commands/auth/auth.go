// Package auth 提供真实的 auth 命令组与交互式凭据输入。
package auth

import (
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/spf13/cobra"
)

// New builds the auth login/list/use/remove/check command tree.
func New(options *invocation.RootOptions, streams *invocation.Streams) *cobra.Command {
	command := &cobra.Command{
		Use:   "auth",
		Short: "Account login and multi-account management",
	}
	command.AddCommand(NewLogin(options, streams))
	command.AddCommand(NewList(streams))
	command.AddCommand(NewUse(streams))
	command.AddCommand(NewRemove(streams))
	command.AddCommand(NewCheck(options, streams))
	return command
}
