package auth

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/common/jsonx"
	"github.com/FlanChanXwO/javdb-cli/internal/environment"
	"github.com/FlanChanXwO/javdb-cli/internal/i18n"
	"github.com/FlanChanXwO/javdb-cli/internal/session"
	"github.com/FlanChanXwO/javdb-cli/internal/session/authstore"
	javdb "github.com/FlanChanXwO/javdb-cli/sdk"
)

// NewCheck verifies the saved token with the server without changing account state.
func NewCheck(options *session.Options, streams *environment.Streams) *cobra.Command {
	var asJSON bool
	command := &cobra.Command{Use: "check", Short: i18n.T("cli.check_default_account_token_does_not_print_token"), RunE: func(command *cobra.Command, _ []string) error {
		_, store, err := authstore.Open()
		if err != nil {
			return err
		}
		account, err := store.Default()
		if err != nil {
			return &session.LoginError{Key: "auth.required", Cause: err}
		}
		status := struct {
			UserID   int64  `json:"user_id"`
			Username string `json:"username"`
			HasToken bool   `json:"has_token"`
			OK       bool   `json:"ok"`
			Error    string `json:"error,omitempty"`
		}{UserID: account.UserID, Username: account.Username, HasToken: account.Token != ""}
		var failure error
		if account.Token == "" {
			failure = &session.LoginError{Key: "auth.required"}
		} else {
			c, err := session.New(options, account.Token)
			if err != nil {
				return err
			}
			failure = c.CheckAuth(command.Context())
			var rejected *javdb.AuthRequired
			if errors.As(failure, &rejected) {
				failure = &session.LoginError{Key: "auth.invalid", Cause: failure}
			}
		}
		status.OK = failure == nil
		if failure != nil {
			status.Error = failure.Error()
		}
		if asJSON {
			data, err := jsonx.MarshalLine(status)
			if err != nil {
				return err
			}
			if _, err = streams.Out.Write(data); err != nil {
				return err
			}
		} else if status.OK {
			if _, err := fmt.Fprintf(streams.Out, i18n.T("cli.ok_d_s"), status.UserID, status.Username); err != nil {
				return err
			}
		} else if _, err := fmt.Fprintf(streams.Out, i18n.T("cli.fail_d_s_s"), status.UserID, status.Username, status.Error); err != nil {
			return err
		}
		return failure
	}}
	command.Flags().BoolVar(&asJSON, "json", false, i18n.T("cli.machine_readable_json"))
	return command
}
