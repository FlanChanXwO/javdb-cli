// Package unmark 提供影片取消标记命令。
package unmark

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/environment"
	"github.com/FlanChanXwO/javdb-cli/internal/i18n"
	"github.com/FlanChanXwO/javdb-cli/internal/record"
	"github.com/FlanChanXwO/javdb-cli/internal/session"
)

// New builds the unmark command.
func New(options *session.Options, streams *environment.Streams) *cobra.Command {
	var isID bool
	var asJSON, asNDJSON bool
	fetch := func(c *session.Client, ctx context.Context, ref string, useID bool) (string, bool, error) {
		mid := ref
		var err error
		if !useID {
			mid, err = c.ResolveMovieID(ctx, ref)
			if err != nil {
				return "", false, err
			}
		}
		ok, err := c.Unmark(ctx, mid)
		if err != nil {
			return "", false, err
		}
		return mid, ok, nil
	}
	runner := &record.BatchRunner{
		Name:  "unmark",
		Kinds: []record.Kind{record.KindMovie},
		ClientFactory: func() (*session.Client, error) {
			return session.NewRequired(options)
		},
		RunOne: func(c *session.Client, ctx context.Context, input record.Envelope) (record.Envelope, error) {
			ref := record.ConsumerRef(input)
			// 管道信封的内部 ID 是权威目标；无 ID 时再由 --id 决定 raw ref 是否直用。
			mid, ok, err := fetch(c, ctx, ref, input.ID != "" || isID)
			if err != nil {
				return record.Envelope{}, err
			}
			return record.New(record.KindMovie, input.Ref, mid).
				WithData(map[string]any{"movie_id": mid, "removed": ok}), nil
		},
		Legacy: func(args []string) error {
			return session.WithRequiredAuth(options, streams.Err, func(c *session.Client) error {
				mid, ok, err := fetch(c, context.Background(), args[0], isID)
				if err != nil {
					return err
				}
				if ok {
					fmt.Fprintf(streams.Out, i18n.T("cli.s_s_9"), args[0], mid)
				} else {
					fmt.Fprintf(streams.Out, i18n.T("cli.s_s_10"), args[0], mid)
				}
				return nil
			})
		},
	}
	cmd := &cobra.Command{
		Use:   "unmark NUMBER",
		Short: i18n.T("cli.remove_watched_want_mark_for_a_movie"),
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runner.Execute(streams, args, asNDJSON, asJSON)
		},
	}
	cmd.Flags().BoolVarP(&isID, "id", "i", false, i18n.T("cli.treat_number_as_internal_movie_id"))
	cmd.Flags().BoolVar(&asJSON, "json", false, i18n.T("cli.machine_readable_json"))
	cmd.Flags().BoolVar(&asNDJSON, "ndjson", false, i18n.T("cli.pipeline_ndjson_envelopes"))
	return cmd
}
