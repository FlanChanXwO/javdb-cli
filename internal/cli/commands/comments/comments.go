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
	var isID, asJSON, asNDJSON bool
	runOne := func(c *javdb.Client, ctx context.Context, input pipeline.Envelope) (pipeline.Envelope, error) {
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
	}
	runner := &pipeline.BatchRunner{
		Name:       "comments",
		LegacyJSON: true,
		// 非 TTY 单项与 stdin 批量统一输出稳定影片 ref；TTY 仍使用 Legacy 人类渲染。
		RouteTextThroughPipeline: true,
		Kinds:                    []pipeline.Kind{pipeline.KindMovie},
		ClientFactory: func() (*javdb.Client, error) {
			return client.NewWithDefaultToken(options)
		},
		RunOne: func(c *javdb.Client, ctx context.Context, input pipeline.Envelope) (pipeline.Envelope, error) {
			var output pipeline.Envelope
			err := client.RetryAnonymousAuth(c, streams.Err, func() error {
				var runErr error
				output, runErr = runOne(c, ctx, input)
				return runErr
			})
			return output, err
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
			runner.Context = cmd.Context()
			return runner.Execute(streams, args, asNDJSON, asJSON)
		},
	}
	cmd.Flags().BoolVarP(&isID, "id", "i", false, "Treat NUMBER as internal movie id")
	cmd.Flags().IntVar(&page, "page", 1, "Page number")
	cmd.Flags().IntVar(&limit, "limit", 20, "Page size")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Machine-readable JSON")
	cmd.Flags().BoolVar(&asNDJSON, "ndjson", false, "Pipeline NDJSON envelopes")
	return cmd
}
