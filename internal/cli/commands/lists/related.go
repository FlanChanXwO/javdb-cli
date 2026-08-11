package lists

import (
	"context"
	"fmt"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/client"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"

	"github.com/spf13/cobra"
)

// NewRelated builds the lists related command.
func NewRelated(options *invocation.RootOptions, streams *invocation.Streams) *cobra.Command {
	var isID bool
	var page, limit int
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "related NUMBER",
		Short: "Public 合集 related to a movie",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.New(options, "")
			if err != nil {
				return err
			}
			ctx := context.Background()
			mid := args[0]
			if !isID {
				mid, err = c.ResolveMovieID(ctx, args[0])
				if err != nil {
					return err
				}
			}
			res, err := c.RelatedLists(ctx, mid, page, limit)
			if err != nil {
				return fmt.Errorf("lists related failed: %w", err)
			}
			items := res.Named("lists")
			if asJSON {
				return writeJSON(streams.Out, map[string]any{"lists": items})
			}
			return writeListRows(streams.Out, streams.Err, items)
		},
	}
	cmd.Flags().BoolVarP(&isID, "id", "i", false, "Treat NUMBER as internal movie id")
	cmd.Flags().IntVar(&page, "page", 1, "Page")
	cmd.Flags().IntVar(&limit, "limit", 20, "Page size")
	cmd.Flags().BoolVar(&asJSON, "json", false, "JSON output")
	return cmd
}
