package auth

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/app"
)

// NewUse builds the auth use command.
func NewUse(aio *app.IO) *cobra.Command {
	return &cobra.Command{
		Use:   "use <user_id>",
		Short: "Set the default account",
		Args:  cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			userID, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("user_id must be integer: %w", err)
			}
			fileStore, store, err := app.OpenAuth()
			if err != nil {
				return err
			}
			if err := store.Use(userID); err != nil {
				return err
			}
			if err := fileStore.Commit(store); err != nil {
				return err
			}
			account, _ := store.Get(userID)
			_, err = fmt.Fprintf(aio.Out, "default account → %s (id=%d)\n", account.Username, userID)
			return err
		},
	}
}
