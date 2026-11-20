package auth

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/environment"
	"github.com/FlanChanXwO/javdb-cli/internal/i18n"
	"github.com/FlanChanXwO/javdb-cli/internal/session"
	"github.com/FlanChanXwO/javdb-cli/internal/session/authstore"
	"github.com/FlanChanXwO/javdb-cli/internal/storage/auth"
)

// NewLogin builds the auth login command.
func NewLogin(options *session.Options, streams *environment.Streams) *cobra.Command {
	var username, password string
	use := true
	command := &cobra.Command{
		Use:   "login",
		Short: i18n.T("command.auth_login"),
		RunE: func(command *cobra.Command, args []string) error {
			c, err := session.New(options, "")
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
				return &session.LoginError{Key: "auth.login_failed", Cause: err}
			}
			userID, resolvedName, err := c.ResolveUserID(ctx)
			if err != nil {
				return fmt.Errorf(i18n.T("auth.missing_user_id"), err)
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
			fmt.Fprintf(streams.Out, i18n.T("auth.signed_in"), resolvedName, userID)
			return nil
		},
	}
	command.Flags().StringVarP(&username, "username", "u", "", i18n.T("flag.username"))
	command.Flags().StringVarP(&password, "password", "p", "", i18n.T("flag.password"))
	command.Flags().BoolVar(&use, "use", true, i18n.T("flag.default_account"))
	return command
}
