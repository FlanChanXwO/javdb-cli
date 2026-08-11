package auth

import (
	"fmt"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/authstore"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"

	"github.com/spf13/cobra"
)

// NewList builds the auth list command.
func NewList(streams *invocation.Streams) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List saved accounts",
		RunE: func(command *cobra.Command, args []string) error {
			_, store, err := authstore.Open()
			if err != nil {
				return err
			}
			if len(store.Accounts) == 0 {
				fmt.Fprintln(streams.Err, "(no accounts)")
				return nil
			}
			for _, account := range store.Accounts {
				mark := " "
				if account.UserID == store.DefaultUserID {
					mark = "*"
				}
				fmt.Fprintf(streams.Out, "%s\t%d\t%s\thas_token=%v\n", mark, account.UserID, account.Username, account.Token != "")
			}
			return nil
		},
	}
}
