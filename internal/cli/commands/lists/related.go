package lists

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/client"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/pipeline"
	javdb "github.com/FlanChanXwO/javdb-cli/sdk"
)

// NewRelated builds the lists related command.
func NewRelated(options *invocation.RootOptions, streams *invocation.Streams) *cobra.Command {
	var isID bool
	var page, limit int
	var asJSON, asJSONL, asText bool
	runOne := func(c *javdb.Client, ctx context.Context, ref string, useID bool) (string, []map[string]any, error) {
		mid := ref
		var err error
		if !useID {
			mid, err = c.ResolveMovieID(ctx, ref)
			if err != nil {
				return "", nil, err
			}
		}
		res, err := c.RelatedLists(ctx, mid, page, limit)
		if err != nil {
			return "", nil, fmt.Errorf("lists related failed: %w", err)
		}
		return mid, res.Named("lists"), nil
	}
	runner := &pipeline.BatchRunner{
		Name:       "lists related",
		LegacyJSON: true,
		Kinds:      []pipeline.Kind{pipeline.KindMovie},
		ClientFactory: func() (*javdb.Client, error) {
			return client.New(options, "")
		},
		RunOne: func(c *javdb.Client, ctx context.Context, input pipeline.Envelope) (pipeline.Envelope, error) {
			ref := pipeline.ConsumerRef(input)
			mid, items, err := runOne(c, ctx, ref, isID && input.ID == "")
			if err != nil {
				return pipeline.Envelope{}, err
			}
			return pipeline.New(pipeline.KindList, input.Ref, mid).WithData(map[string]any{"lists": items}), nil
		},
		Legacy: func(args []string) error {
			c, err := client.New(options, "")
			if err != nil {
				return err
			}
			_, items, err := runOne(c, context.Background(), args[0], isID)
			if err != nil {
				return err
			}
			if asJSON {
				return writeJSON(streams.Out, map[string]any{"lists": items})
			}
			return writeListRows(streams.Out, streams.Err, items)
		},
	}
	cmd := &cobra.Command{
		Use:   "related NUMBER",
		Short: "Public 合集 related to a movie",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runner.Execute(streams, args, asJSONL, asText, asJSON)
		},
	}
	cmd.Flags().BoolVarP(&isID, "id", "i", false, "Treat NUMBER as internal movie id")
	cmd.Flags().IntVar(&page, "page", 1, "Page")
	cmd.Flags().IntVar(&limit, "limit", 20, "Page size")
	cmd.Flags().BoolVar(&asJSON, "json", false, "JSON output")
	cmd.Flags().BoolVar(&asJSONL, "jsonl", false, "Pipeline JSONL envelopes")
	cmd.Flags().BoolVar(&asText, "text", false, "Plain text lines (default for TTY)")
	return cmd
}
