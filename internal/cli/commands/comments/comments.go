// Package comments 提供影片评论单页命令。
package comments

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/client"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/pipeline"
	"github.com/FlanChanXwO/javdb-cli/internal/common/jsonx"
	javdb "github.com/FlanChanXwO/javdb-cli/sdk"
)

// New builds the one-page movie review command.
func New(options *invocation.RootOptions, streams *invocation.Streams) *cobra.Command {
	var page, limit int
	var isID, asJSON, asJSONL, asText bool
	runner := &pipeline.BatchRunner{
		Name:       "comments",
		LegacyJSON: true,
		Kinds:      []pipeline.Kind{pipeline.KindMovie},
		ClientFactory: func() (*javdb.Client, error) {
			return client.NewWithDefaultToken(options)
		},
		RunOne: func(c *javdb.Client, ctx context.Context, input pipeline.Envelope) (pipeline.Envelope, error) {
			movieID := input.ID
			if movieID == "" {
				movieID = input.Ref
			}
			var err error
			if !isID && input.ID == "" {
				movieID, err = c.ResolveMovieID(ctx, movieID)
				if err != nil {
					return pipeline.Envelope{}, err
				}
			}
			reviews, err := c.MovieComments(ctx, movieID, page, limit)
			if err != nil {
				return pipeline.Envelope{}, fmt.Errorf("comments failed: %w", err)
			}
			return pipeline.New(pipeline.KindComment, input.Ref, movieID).
				WithData(map[string]any{"movie_id": movieID, "page": page, "limit": limit, "reviews": reviews}), nil
		},
		Legacy: func(args []string) error {
			if page < 1 {
				return fmt.Errorf("--page must be positive")
			}
			if limit < 1 {
				return fmt.Errorf("--limit must be positive")
			}
			return client.WithOptionalAuth(options, streams.Err, func(c *javdb.Client) error {
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
					b, err := jsonx.MarshalLine(map[string]any{
						"movie_id": movieID,
						"page":     page,
						"limit":    limit,
						"reviews":  reviews,
					})
					if err != nil {
						return err
					}
					_, err = streams.Out.Write(b)
					return err
				}
				return writeComments(streams.Out, streams.Err, reviews)
			})
		},
	}
	cmd := &cobra.Command{
		Use:   "comments NUMBER",
		Short: "List one page of movie reviews",
		Long:  "List one requested page of movie reviews. The command never fetches later pages automatically.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if page < 1 {
				return fmt.Errorf("--page must be positive")
			}
			if limit < 1 {
				return fmt.Errorf("--limit must be positive")
			}
			return runner.Execute(streams, args, asJSONL, asText, asJSON)
		},
	}
	cmd.Flags().BoolVarP(&isID, "id", "i", false, "Treat NUMBER as internal movie id")
	cmd.Flags().IntVar(&page, "page", 1, "Page number")
	cmd.Flags().IntVar(&limit, "limit", 20, "Page size")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Machine-readable JSON")
	cmd.Flags().BoolVar(&asJSONL, "jsonl", false, "Pipeline JSONL envelopes")
	cmd.Flags().BoolVar(&asText, "text", false, "Plain text lines (default for TTY)")
	return cmd
}
