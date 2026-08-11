package auth

import (
	"fmt"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/authstore"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"strconv"

	"github.com/spf13/cobra"
)

// NewRemove builds the auth remove command.
func NewRemove(streams *invocation.Streams) *cobra.Command {
	return &cobra.Command{
		Use:   "remove <user_id>",
		Short: "Remove a saved account",
		Args:  cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			userID, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("user_id must be integer: %w", err)
			}
			fileStore, store, err := authstore.Open()
			if err != nil {
				return err
			}
			if err := store.Remove(userID); err != nil {
				return err
			}
			if err := fileStore.Commit(store); err != nil {
				return err
			}
			_, err = fmt.Fprintf(streams.Out, "removed account id=%d\n", userID)
			return err
		},
	}
}
