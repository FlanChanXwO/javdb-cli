package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/sdk"
)

func newCommentsCmd(rf *rootFlags, aio *appIO) *cobra.Command {
	var page, limit int
	var isID, asJSON bool
	cmd := &cobra.Command{
		Use:   "comments NUMBER",
		Short: "List one page of movie reviews",
		Long:  "List one requested page of movie reviews. The command never fetches later pages automatically.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if page < 1 {
				return fmt.Errorf("--page must be positive")
			}
			if limit < 1 {
				return fmt.Errorf("--limit must be positive")
			}
			return withOptionalAuthClient(rf, aio, func(c *javdb.Client) error {
				ctx := context.Background()
				movieID := args[0]
				var err error
				if !isID {
					movieID, err = c.ResolveMovieID(ctx, args[0])
					if err != nil {
						return err
					}
				}
				reviews, err := c.MovieComments(ctx, movieID, page, limit)
				if err != nil {
					return fmt.Errorf("comments failed: %w", err)
				}
				if asJSON {
					return EmitJSON(aio.out, map[string]any{
						"movie_id": movieID,
						"page":     page,
						"limit":    limit,
						"reviews":  reviews,
					})
				}
				PrintMovieComments(aio.out, aio.err, reviews)
				return nil
			})
		},
	}
	cmd.Flags().BoolVarP(&isID, "id", "i", false, "Treat NUMBER as internal movie id")
	cmd.Flags().IntVar(&page, "page", 1, "Page number")
	cmd.Flags().IntVar(&limit, "limit", 20, "Page size")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Machine-readable JSON")
	return cmd
}
