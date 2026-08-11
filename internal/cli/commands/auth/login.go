package auth

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/app"
	"github.com/FlanChanXwO/javdb-cli/internal/storage/auth"
)

// NewLogin builds the auth login command.
func NewLogin(flags *app.Flags, aio *app.IO) *cobra.Command {
	var username, password string
	use := true
	command := &cobra.Command{
		Use:   "login",
		Short: "Log in with username/password (interactive if flags omitted)",
		RunE: func(command *cobra.Command, args []string) error {
			runtimeConfig, err := app.LoadRuntime(flags)
			if err != nil {
				return err
			}
			if username == "" {
				username, err = PromptUsername(aio.In, aio.Out)
				if err != nil {
					return err
				}
			}
			if password == "" {
				password, err = PromptPassword(aio.Out)
				if err != nil {
					return err
				}
			}
			client, err := app.NewClient(runtimeConfig, "")
			if err != nil {
				return err
			}
			ctx := context.Background()
			token, err := client.Login(ctx, username, password)
			if err != nil {
				return fmt.Errorf("login failed: %w", err)
			}
			userID, resolvedName, err := client.ResolveUserID(ctx)
			if err != nil {
				return fmt.Errorf("login ok but user id required: %w", err)
			}
			if resolvedName == "" {
				resolvedName = username
			}
			fileStore, store, err := app.OpenAuth()
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
			fmt.Fprintf(aio.Out, "logged in as %s (id=%d)\n", resolvedName, userID)
			return nil
		},
	}
	command.Flags().StringVarP(&username, "username", "u", "", "Username / email")
	command.Flags().StringVarP(&password, "password", "p", "", "Password")
	command.Flags().BoolVar(&use, "use", true, "Set as default account after login")
	return command
}
