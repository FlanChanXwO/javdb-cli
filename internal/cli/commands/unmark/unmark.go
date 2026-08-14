// Package unmark 提供影片取消标记命令。
package unmark

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/client"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/pipeline"
	"github.com/FlanChanXwO/javdb-cli/sdk"
)

// New builds the unmark command.
func New(options *invocation.RootOptions, streams *invocation.Streams) *cobra.Command {
	var isID bool
	var asJSON, asJSONL, asText bool
	fetch := func(c *javdb.Client, ctx context.Context, ref string, useID bool) (string, bool, error) {
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
			return "", false, fmt.Errorf("unmark failed: %w", err)
		}
		return mid, ok, nil
	}
	runner := &pipeline.BatchRunner{
		Name:  "unmark",
		Kinds: []pipeline.Kind{pipeline.KindMovie},
		ClientFactory: func() (*javdb.Client, error) {
			return client.NewWithDefaultToken(options)
		},
		RunOne: func(c *javdb.Client, ctx context.Context, input pipeline.Envelope) (pipeline.Envelope, error) {
			ref := pipeline.ConsumerRef(input)
			mid, ok, err := fetch(c, ctx, ref, isID && input.ID == "")
			if err != nil {
				return pipeline.Envelope{}, err
			}
			return pipeline.New(pipeline.KindMovie, input.Ref, mid).
				WithData(map[string]any{"movie_id": mid, "removed": ok}), nil
		},
		Legacy: func(args []string) error {
			return client.WithRequiredAuth(options, streams.Err, func(c *javdb.Client) error {
				mid, ok, err := fetch(c, context.Background(), args[0], isID)
				if err != nil {
					return err
				}
				if ok {
					fmt.Fprintf(streams.Out, "已取消标记 %s (%s)\n", args[0], mid)
				} else {
					fmt.Fprintf(streams.Out, "无标记可取消 %s (%s)\n", args[0], mid)
				}
				return nil
			})
		},
	}
	cmd := &cobra.Command{
		Use:   "unmark NUMBER",
		Short: "Remove watched/want mark for a movie",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runner.Execute(streams, args, asJSONL, asText, asJSON)
		},
	}
	cmd.Flags().BoolVarP(&isID, "id", "i", false, "Treat NUMBER as internal movie id")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Machine-readable JSON")
	cmd.Flags().BoolVar(&asJSONL, "jsonl", false, "Pipeline JSONL envelopes")
	cmd.Flags().BoolVar(&asText, "text", false, "Plain text lines (default)")
	return cmd
}
