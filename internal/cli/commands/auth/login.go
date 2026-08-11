package auth

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/authstore"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/client"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/storage/auth"
)

// NewLogin builds the auth login command.
func NewLogin(options *invocation.RootOptions, streams *invocation.Streams) *cobra.Command {
	var username, password string
	use := true
	command := &cobra.Command{
		Use:   "login",
		Short: "Log in with username/password (interactive if flags omitted)",
		RunE: func(command *cobra.Command, args []string) error {
			c, err := client.New(options, "")
			if err != nil {
				return err
			}
			if username == "" {
				username, err = PromptUsername(streams.In, streams.Out)
				if err != nil {
					return err
				}
			}
			if password == "" {
				password, err = PromptPassword(streams.Out)
				if err != nil {
					return err
				}
			}
			ctx := context.Background()
			token, err := c.Login(ctx, username, password)
			if err != nil {
				return fmt.Errorf("login failed: %w", err)
			}
			userID, resolvedName, err := c.ResolveUserID(ctx)
			if err != nil {
				return fmt.Errorf("login ok but user id required: %w", err)
			}
			if resolvedName == "" {
				resolvedName = username
			}
			fileStore, store, err := authstore.Open()
			if err != nil {
				return err
			}
			store.Upsert(auth.Account{
				UserID:   userID,
				Username: resolvedName,
				Password: password,
				Token:    token,
			}, use)
			if err := fileStore.Commit(store); err != nil {
				return err
			}
			fmt.Fprintf(streams.Out, "logged in as %s (id=%d)\n", resolvedName, userID)
			return nil
		},
	}
	command.Flags().StringVarP(&username, "username", "u", "", "Username / email")
	command.Flags().StringVarP(&password, "password", "p", "", "Password")
	command.Flags().BoolVar(&use, "use", true, "Set as default account after login")
	return command
}
