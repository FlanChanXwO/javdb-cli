package auth

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/config/paths"
	"github.com/FlanChanXwO/javdb-cli/internal/environment"
	"github.com/FlanChanXwO/javdb-cli/internal/i18n"
	"github.com/FlanChanXwO/javdb-cli/internal/session/authstore"
)

// NewList builds the auth list command.
func NewList(streams *environment.Streams) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: i18n.T("cli.list_saved_accounts"),
		RunE: func(command *cobra.Command, args []string) error {
			// 本地命令也按"第一个真实命令创建配置"契约触发首次创建。
			if err := paths.EnsureDefaultConfigFile(); err != nil {
				return err
			}
			_, store, err := authstore.Open()
			if err != nil {
				return err
			}
			if len(store.Accounts) == 0 {
				fmt.Fprintln(streams.Diagnostics(slog.LevelInfo), i18n.T("cli.no_accounts"))
				return nil
			}
			for _, account := range store.Accounts {
				mark := " "
				if account.UserID == store.DefaultUserID {
					mark = "*"
				}
				fmt.Fprintf(streams.Out, i18n.T("cli.s_d_s_has_token_v"), mark, account.UserID, account.Username, account.Token != "")
			}
			return nil
		},
	}
}
