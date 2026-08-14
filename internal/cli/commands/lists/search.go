package lists

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/client"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/pipeline"
	"github.com/FlanChanXwO/javdb-cli/sdk"
)

// NewSearch builds the lists search command.
func NewSearch(options *invocation.RootOptions, streams *invocation.Streams) *cobra.Command {
	var page, limit int
	var zone string
	var asJSON, asJSONL, asText bool
	runOne := func(c *javdb.Client, ctx context.Context, keyword string) ([]map[string]any, error) {
		res, err := c.Search(ctx, keyword, javdb.SearchOptions{
			Page: page, Limit: limit, Zone: zone, Type: "list",
		})
		if err != nil {
			return nil, fmt.Errorf("lists search failed: %w", err)
		}
		return res.Named("lists"), nil
	}
	runner := &pipeline.BatchRunner{
		Name:       "lists search",
		LegacyJSON: true,
		Kinds:      []pipeline.Kind{pipeline.KindList},
		ClientFactory: func() (*javdb.Client, error) {
			return client.New(options, "")
		},
		RunOne: func(c *javdb.Client, ctx context.Context, input pipeline.Envelope) (pipeline.Envelope, error) {
			items, err := runOne(c, ctx, pipeline.ConsumerRef(input))
			if err != nil {
				return pipeline.Envelope{}, err
			}
			return pipeline.New(pipeline.KindList, input.Ref, "").WithData(map[string]any{"lists": items}), nil
		},
		Legacy: func(args []string) error {
			c, err := client.New(options, "")
			if err != nil {
				return err
			}
			items, err := runOne(c, context.Background(), args[0])
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
		Use:   "search KEYWORD",
		Short: "Search public 合集",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runner.Execute(streams, args, asJSONL, asText, asJSON)
		},
	}
	cmd.Flags().IntVar(&page, "page", 1, "Page")
	cmd.Flags().IntVar(&limit, "limit", 0, "Page size")
	cmd.Flags().StringVar(&zone, "zone", "all", "censored|uncensored|western|fc2|all")
	cmd.Flags().BoolVar(&asJSON, "json", false, "JSON output")
	cmd.Flags().BoolVar(&asJSONL, "jsonl", false, "Pipeline JSONL envelopes")
	cmd.Flags().BoolVar(&asText, "text", false, "Plain text lines (default for TTY)")
	return cmd
}
