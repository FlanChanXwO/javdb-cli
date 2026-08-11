package lists

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/app"
)

// NewRelated builds the lists related command.
func NewRelated(flags *app.Flags, aio *app.IO) *cobra.Command {
	var isID bool
	var page, limit int
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "related NUMBER",
		Short: "Public 合集 related to a movie",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := app.LoadRuntime(flags)
			if err != nil {
				return err
			}
			c, err := app.NewClient(rt, "")
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
				return writeJSON(aio.Out, map[string]any{"lists": items})
			}
			return writeListRows(aio.Out, aio.Err, items)
		},
	}
	cmd.Flags().BoolVarP(&isID, "id", "i", false, "Treat NUMBER as internal movie id")
	cmd.Flags().IntVar(&page, "page", 1, "Page")
	cmd.Flags().IntVar(&limit, "limit", 20, "Page size")
	cmd.Flags().BoolVar(&asJSON, "json", false, "JSON output")
	return cmd
}
