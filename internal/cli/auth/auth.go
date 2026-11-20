// Package auth 提供真实的 auth 命令组与交互式凭据输入。
package auth

import (
	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/environment"
	"github.com/FlanChanXwO/javdb-cli/internal/i18n"
	"github.com/FlanChanXwO/javdb-cli/internal/session"
)

// New builds the auth login/list/use/remove/check command tree.
func New(options *session.Options, streams *environment.Streams) *cobra.Command {
	command := &cobra.Command{
		Use:   "auth",
		Short: i18n.T("cli.account_login_and_multi_account_management"),
	}
	command.AddCommand(NewLogin(options, streams))
	command.AddCommand(NewList(streams))
	command.AddCommand(NewUse(streams))
	command.AddCommand(NewRemove(streams))
	command.AddCommand(NewCheck(options, streams))
	return command
}
