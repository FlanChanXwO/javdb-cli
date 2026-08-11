package auth

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/app"
)

// NewList builds the auth list command.
func NewList(aio *app.IO) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List saved accounts",
		RunE: func(command *cobra.Command, args []string) error {
			_, store, err := app.OpenAuth()
			if err != nil {
				return err
			}
			if len(store.Accounts) == 0 {
				fmt.Fprintln(aio.Err, "(no accounts)")
				return nil
			}
			for _, account := range store.Accounts {
				mark := " "
				if account.UserID == store.DefaultUserID {
					mark = "*"
				}
				fmt.Fprintf(aio.Out, "%s\t%d\t%s\thas_token=%v\n", mark, account.UserID, account.Username, account.Token != "")
			}
			return nil
		},
	}
}
