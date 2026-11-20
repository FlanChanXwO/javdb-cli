package auth

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/config/paths"
	"github.com/FlanChanXwO/javdb-cli/internal/environment"
	"github.com/FlanChanXwO/javdb-cli/internal/i18n"
	"github.com/FlanChanXwO/javdb-cli/internal/record"
	"github.com/FlanChanXwO/javdb-cli/internal/session"
	"github.com/FlanChanXwO/javdb-cli/internal/session/authstore"
)

// NewUse builds the auth use command.
func NewUse(streams *environment.Streams) *cobra.Command {
	var asJSON, asNDJSON bool
	runOne := func(userID int64) (string, error) {
		// 参数校验通过后再按"第一个真实命令创建配置"契约触发首次创建。
		if err := paths.EnsureDefaultConfigFile(); err != nil {
			return "", err
		}
		fileStore, store, err := authstore.Open()
		if err != nil {
			return "", err
		}
		if err := store.Use(userID); err != nil {
			return "", err
		}
		if err := fileStore.Commit(store); err != nil {
			return "", err
		}
		account, _ := store.Get(userID)
		return account.Username, nil
	}
	runner := &record.BatchRunner{
		Name:          "auth use",
		Kinds:         []record.Kind{record.KindAccount},
		ClientFactory: nil,
		RunOne: func(_ *session.Client, ctx context.Context, input record.Envelope) (record.Envelope, error) {
			userID, err := strconv.ParseInt(record.ConsumerRef(input), 10, 64)
			if err != nil {
				return record.Envelope{}, fmt.Errorf(i18n.T("auth.invalid_user_id"), err)
			}
			username, err := runOne(userID)
			if err != nil {
				return record.Envelope{}, err
			}
			return record.New(record.KindAccount, username, fmt.Sprint(userID)).WithData(map[string]any{"user_id": userID, "default": true}), nil
		},
		Legacy: func(args []string) error {
			userID, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf(i18n.T("auth.invalid_user_id"), err)
			}
			username, err := runOne(userID)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(streams.Out, i18n.T("auth.default_set"), username, userID)
			return err
		},
	}
	cmd := &cobra.Command{
		Use:   "use <user_id>",
		Short: i18n.T("command.auth_use"),
		Args:  cobra.MaximumNArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			return runner.Execute(streams, args, asNDJSON, asJSON)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, i18n.T("flag.json"))
	cmd.Flags().BoolVar(&asNDJSON, "ndjson", false, i18n.T("flag.ndjson"))
	return cmd
}
