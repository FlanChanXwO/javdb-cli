package lists

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/client"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/sdk"
)

// NewSearch builds the lists search command.
func NewSearch(options *invocation.RootOptions, streams *invocation.Streams) *cobra.Command {
	var page, limit int
	var zone string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "search KEYWORD",
		Short: "Search public 合集",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.New(options, "")
			if err != nil {
				return err
			}
			res, err := c.Search(context.Background(), args[0], javdb.SearchOptions{
				Page: page, Limit: limit, Zone: zone, Type: "list",
			})
			if err != nil {
				return fmt.Errorf("lists search failed: %w", err)
			}
			items := res.Named("lists")
			if asJSON {
				return writeJSON(streams.Out, map[string]any{"lists": items})
			}
			return writeListRows(streams.Out, streams.Err, items)
		},
	}
	cmd.Flags().IntVar(&page, "page", 1, "Page")
	cmd.Flags().IntVar(&limit, "limit", 0, "Page size")
	cmd.Flags().StringVar(&zone, "zone", "all", "censored|uncensored|western|fc2|all")
	cmd.Flags().BoolVar(&asJSON, "json", false, "JSON output")
	return cmd
}
