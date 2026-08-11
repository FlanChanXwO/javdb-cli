// Package auth 提供真实的 auth 命令组与交互式凭据输入。
package auth

import (
	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/app"
)

// New builds the auth login/list/use/remove/check command tree.
func New(flags *app.Flags, aio *app.IO) *cobra.Command {
	command := &cobra.Command{
		Use:   "auth",
		Short: "Account login and multi-account management",
	}
	command.AddCommand(NewLogin(flags, aio))
	command.AddCommand(NewList(aio))
	command.AddCommand(NewUse(aio))
	command.AddCommand(NewRemove(aio))
	command.AddCommand(NewCheck(flags, aio))
	return command
}
