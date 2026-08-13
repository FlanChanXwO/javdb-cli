package auth

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/authstore"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/pipeline"
	"github.com/FlanChanXwO/javdb-cli/internal/config/paths"
	javdb "github.com/FlanChanXwO/javdb-cli/sdk"
)

// NewRemove builds the auth remove command.
func NewRemove(streams *invocation.Streams) *cobra.Command {
	var asJSON, asJSONL, asText bool
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
	runner := &pipeline.BatchRunner{
		Name:          "auth remove",
		Kinds:         []pipeline.Kind{pipeline.KindAccount},
		ClientFactory: nil,
		RunOne: func(_ *javdb.Client, ctx context.Context, input pipeline.Envelope) (pipeline.Envelope, error) {
			userID, err := strconv.ParseInt(pipeline.ConsumerRef(input), 10, 64)
			if err != nil {
				return pipeline.Envelope{}, fmt.Errorf("user_id must be integer: %w", err)
			}
			if err := runOne(userID); err != nil {
				return pipeline.Envelope{}, err
			}
			return pipeline.New(pipeline.KindAccount, "", fmt.Sprint(userID)).WithData(map[string]any{"user_id": userID, "removed": true}), nil
		},
		Legacy: func(args []string) error {
			userID, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("user_id must be integer: %w", err)
			}
			if err := runOne(userID); err != nil {
				return err
			}
			_, err = fmt.Fprintf(streams.Out, "removed account id=%d\n", userID)
			return err
		},
	}
	cmd := &cobra.Command{
		Use:   "remove <user_id>",
		Short: "Remove a saved account",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			return runner.Execute(streams, args, asJSONL, asText, asJSON)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "Machine-readable JSON")
	cmd.Flags().BoolVar(&asJSONL, "jsonl", false, "Pipeline JSONL envelopes")
	cmd.Flags().BoolVar(&asText, "text", false, "Plain text lines (default for TTY)")
	return cmd
}
