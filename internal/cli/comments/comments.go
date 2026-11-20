// Package comments 提供影片评论单页命令。
package comments

import (
	"context"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/common/jsonx"
	"github.com/FlanChanXwO/javdb-cli/internal/environment"
	"github.com/FlanChanXwO/javdb-cli/internal/i18n"
	"github.com/FlanChanXwO/javdb-cli/internal/record"
	"github.com/FlanChanXwO/javdb-cli/internal/session"
)

// New builds the one-page movie review command.
func New(options *session.Options, streams *environment.Streams) *cobra.Command {
	var page, limit int
	var isID, asJSON, asNDJSON bool
	runOne := func(c *session.Client, ctx context.Context, input record.Envelope) (record.Envelope, error) {
		movieID := input.ID
		if movieID == "" {
			movieID = input.Ref
		}
		var err error
		if !isID && input.ID == "" {
			movieID, err = c.ResolveMovieID(ctx, movieID)
			if err != nil {
				return record.Envelope{}, err
			}
		}
		reviews, err := c.MovieComments(ctx, movieID, page, limit)
		if err != nil {
			return record.Envelope{}, err
		}
		return record.New(record.KindComment, input.Ref, movieID).
			WithData(map[string]any{"movie_id": movieID, "page": page, "limit": limit, "reviews": reviews}), nil
	}
	runner := &record.BatchRunner{
		Name:       "comments",
		LegacyJSON: true,
		// 非 TTY 单项与 stdin 批量统一输出稳定影片 ref；TTY 仍使用 Legacy 人类渲染。
		RouteTextThroughPipeline: true,
		Kinds:                    []record.Kind{record.KindMovie},
		ClientFactory: func() (*session.Client, error) {
			return session.NewWithDefaultToken(options)
		},
		RunOne: func(c *session.Client, ctx context.Context, input record.Envelope) (record.Envelope, error) {
			return runOne(c, ctx, input)
		},
		Legacy: func(args []string) error {
			if page < 1 {
				return i18n.Errorf("cli.page_must_be_positive")
			}
			if limit < 1 {
				return i18n.Errorf("cli.limit_must_be_positive")
			}
			return session.WithOptionalAuth(options, streams.Err, func(c *session.Client) error {
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
					return err
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
				return writeComments(streams.Out, streams.Diagnostics(slog.LevelInfo), reviews)
			})
		},
	}
	cmd := &cobra.Command{
		Use:   "comments NUMBER",
		Short: i18n.T("cli.list_one_page_of_movie_reviews"),
		Long:  i18n.T("cli.list_one_requested_page_of_movie_reviews_the_command_never_fetches_lat"),
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if page < 1 {
				return i18n.Errorf("cli.page_must_be_positive")
			}
			if limit < 1 {
				return i18n.Errorf("cli.limit_must_be_positive")
			}
			runner.Context = cmd.Context()
			return runner.Execute(streams, args, asNDJSON, asJSON)
		},
	}
	cmd.Flags().BoolVarP(&isID, "id", "i", false, i18n.T("cli.treat_number_as_internal_movie_id"))
	cmd.Flags().IntVar(&page, "page", 1, i18n.T("cli.page_number"))
	cmd.Flags().IntVar(&limit, "limit", 20, i18n.T("cli.page_size"))
	cmd.Flags().BoolVar(&asJSON, "json", false, i18n.T("cli.machine_readable_json"))
	cmd.Flags().BoolVar(&asNDJSON, "ndjson", false, i18n.T("cli.pipeline_ndjson_envelopes"))
	return cmd
}
