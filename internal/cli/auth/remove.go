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

// NewRemove builds the auth remove command.
func NewRemove(streams *environment.Streams) *cobra.Command {
	var asJSON, asNDJSON bool
	runOne := func(userID int64) error {
		// 参数校验通过后再按"第一个真实命令创建配置"契约触发首次创建。
		if err := paths.EnsureDefaultConfigFile(); err != nil {
			return err
		}
		fileStore, store, err := authstore.Open()
		if err != nil {
			return err
		}
		if err := store.Remove(userID); err != nil {
			return err
		}
		return fileStore.Commit(store)
	}
	runner := &record.BatchRunner{
		Name:          "auth remove",
		Kinds:         []record.Kind{record.KindAccount},
		ClientFactory: nil,
		RunOne: func(_ *session.Client, ctx context.Context, input record.Envelope) (record.Envelope, error) {
			userID, err := strconv.ParseInt(record.ConsumerRef(input), 10, 64)
			if err != nil {
				return record.Envelope{}, fmt.Errorf(i18n.T("cli.user_id_must_be_integer_w"), err)
			}
			if err := runOne(userID); err != nil {
				return record.Envelope{}, err
			}
			return record.New(record.KindAccount, "", fmt.Sprint(userID)).WithData(map[string]any{"user_id": userID, "removed": true}), nil
		},
		Legacy: func(args []string) error {
			userID, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf(i18n.T("cli.user_id_must_be_integer_w"), err)
			}
			if err := runOne(userID); err != nil {
				return err
			}
			_, err = fmt.Fprintf(streams.Out, i18n.T("cli.removed_account_id_d"), userID)
			return err
		},
	}
	cmd := &cobra.Command{
		Use:   "remove <user_id>",
		Short: i18n.T("cli.remove_a_saved_account"),
		Args:  cobra.MaximumNArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			return runner.Execute(streams, args, asNDJSON, asJSON)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, i18n.T("cli.machine_readable_json"))
	cmd.Flags().BoolVar(&asNDJSON, "ndjson", false, i18n.T("cli.pipeline_ndjson_envelopes"))
	return cmd
}
