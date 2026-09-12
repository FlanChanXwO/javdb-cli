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
	var asJSON, asNDJSON bool
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
		RunMany: func(c *javdb.Client, ctx context.Context, input pipeline.Envelope) ([]pipeline.Envelope, error) {
			ref := pipeline.ConsumerRef(input)
			// 信封自带的 ID 是权威 movie ID；--id 只影响没有 ID 的输入。
			_, items, err := runOne(c, ctx, ref, input.ID != "" || isID)
			if err != nil {
				return nil, err
			}
			envelopes := make([]pipeline.Envelope, 0, len(items))
			for _, item := range items {
				id := display(item["id"])
				listRef := display(item["name"])
				if listRef == "" {
					listRef = id
				}
				envelopes = append(envelopes, pipeline.New(pipeline.KindList, listRef, id).WithData(map[string]any{"list": item}))
			}
			return envelopes, nil
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
			return runner.Execute(streams, args, asNDJSON, asJSON)
		},
	}
	cmd.Flags().BoolVarP(&isID, "id", "i", false, "Treat NUMBER as internal movie id")
	cmd.Flags().IntVar(&page, "page", 1, "Page")
	cmd.Flags().IntVar(&limit, "limit", 20, "Page size")
	cmd.Flags().BoolVar(&asJSON, "json", false, "JSON output")
	cmd.Flags().BoolVar(&asNDJSON, "ndjson", false, "Pipeline NDJSON envelopes")
	return cmd
}
