// Package detail 提供影片详情与可选磁力输出命令。
package detail

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

// New builds the movie detail command (graph ids for agent navigation).
func New(options *session.Options, streams *environment.Streams) *cobra.Command {
	var isID, withMagnets, asJSON, asNDJSON bool
	runOne := func(c *session.Client, ctx context.Context, input record.Envelope) (record.Envelope, error) {
		mid := input.ID
		if mid == "" {
			mid = input.Ref
		}
		var err error
		if !isID && input.ID == "" {
			mid, err = c.ResolveMovieID(ctx, mid)
			if err != nil {
				return record.Envelope{}, err
			}
		}
		movie, err := c.MovieDetail(ctx, mid)
		if err != nil {
			return record.Envelope{}, err
		}
		envelope := record.New(record.KindMovie, input.Ref, mid).WithData(map[string]any{"movie": movie})
		if withMagnets {
			mags, err := c.MovieMagnets(ctx, mid)
			if err != nil {
				return record.Envelope{}, err
			}
			envelope.Data["magnets"] = mags
		}
		return envelope, nil
	}
	runner := &record.BatchRunner{
		Name:       "detail",
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
			return session.WithOptionalAuth(options, streams.Err, func(c *session.Client) error {
				var err error
				ctx := context.Background()
				mid := args[0]
				if !isID {
					mid, err = c.ResolveMovieID(ctx, args[0])
					if err != nil {
						return err
					}
				}
				movie, err := c.MovieDetail(ctx, mid)
				if err != nil {
					return err
				}
				var mags []map[string]any
				if withMagnets {
					mags, err = c.MovieMagnets(ctx, mid)
					if err != nil {
						return err
					}
				}
				if asJSON {
					payload := map[string]any{}
					for k, v := range movie {
						payload[k] = v
					}
					if withMagnets {
						payload["magnets"] = mags
					}
					b, err := jsonx.MarshalLine(payload)
					if err != nil {
						return err
					}
					_, err = streams.Out.Write(b)
					return err
				}
				renderDetail(streams.Out, movie)
				if withMagnets {
					renderMagnets(streams.Out, streams.Diagnostics(slog.LevelInfo), mags)
				}
				return nil
			})
		},
	}
	cmd := &cobra.Command{
		Use:   "detail NUMBER",
		Short: i18n.T("cli.show_movie_detail_graph_ids_for_agent_navigation"),
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			runner.Context = cmd.Context()
			return runner.Execute(streams, args, asNDJSON, asJSON)
		},
	}
	cmd.Flags().BoolVarP(&isID, "id", "i", false, i18n.T("cli.treat_argument_as_internal_movie_id"))
	cmd.Flags().BoolVar(&withMagnets, "magnets", false, i18n.T("cli.also_list_magnet_links"))
	cmd.Flags().BoolVar(&asJSON, "json", false, i18n.T("cli.machine_readable_json"))
	cmd.Flags().BoolVar(&asNDJSON, "ndjson", false, i18n.T("cli.pipeline_ndjson_envelopes"))
	return cmd
}
