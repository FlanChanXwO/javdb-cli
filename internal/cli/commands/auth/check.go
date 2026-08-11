package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/app"
	"github.com/FlanChanXwO/javdb-cli/internal/common/jsonx"
	"github.com/FlanChanXwO/javdb-cli/sdk"
)

// NewCheck builds the auth check command (does not print the token).
func NewCheck(flags *app.Flags, aio *app.IO) *cobra.Command {
	var asJSON bool
	command := &cobra.Command{
		Use:   "check",
		Short: "Check default account token (does not print token)",
		RunE: func(command *cobra.Command, args []string) error {
			runtimeConfig, err := app.LoadRuntime(flags)
			if err != nil {
				return err
			}
			_, store, err := app.OpenAuth()
			if err != nil {
				return err
			}
			account, err := store.Default()
			if err != nil {
				return errors.New("no default account; run: javdb auth login")
			}
			type result struct {
				UserID   int64  `json:"user_id"`
				Username string `json:"username"`
				HasToken bool   `json:"has_token"`
				OK       bool   `json:"ok"`
				Error    string `json:"error,omitempty"`
			}
			status := result{UserID: account.UserID, Username: account.Username, HasToken: account.Token != ""}
			if account.Token == "" {
				status.Error = "no token"
			} else {
				client, err := app.NewClient(runtimeConfig, account.Token)
				if err != nil {
					return err
				}
				if _, _, err := client.ResolveUserID(context.Background()); err != nil {
					status.Error = err.Error()
					var authRequired *javdb.AuthRequired
					if errors.As(err, &authRequired) {
						status.Error = "token expired or invalid; run: javdb auth login"
					}
				} else {
					status.OK = true
				}
			}
			if asJSON {
				b, err := jsonx.MarshalLine(status)
				if err != nil {
					return err
				}
				_, err = aio.Out.Write(b)
				return err
			}
			if status.OK {
				fmt.Fprintf(aio.Out, "ok\t%d\t%s\n", status.UserID, status.Username)
			} else {
				fmt.Fprintf(aio.Out, "fail\t%d\t%s\t%s\n", status.UserID, status.Username, status.Error)
				return errors.New(status.Error)
			}
			return nil
		},
	}
	command.Flags().BoolVar(&asJSON, "json", false, "Machine-readable JSON")
	return command
}
