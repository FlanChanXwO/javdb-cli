// Package detail 提供影片详情与可选磁力输出命令。
package detail

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

// New builds the movie detail command (graph ids for agent navigation).
func New(options *invocation.RootOptions, streams *invocation.Streams) *cobra.Command {
	var isID, withMagnets, asJSON, asNDJSON bool
	runOne := func(c *javdb.Client, ctx context.Context, input pipeline.Envelope) (pipeline.Envelope, error) {
		mid := input.ID
		if mid == "" {
			mid = input.Ref
		}
		var err error
		if !isID && input.ID == "" {
			mid, err = c.ResolveMovieID(ctx, mid)
			if err != nil {
				return pipeline.Envelope{}, err
			}
		}
		movie, err := c.MovieDetail(ctx, mid)
		if err != nil {
			return pipeline.Envelope{}, fmt.Errorf("detail failed: %w", err)
		}
		envelope := pipeline.New(pipeline.KindMovie, input.Ref, mid).WithData(map[string]any{"movie": movie})
		if withMagnets {
			mags, err := c.MovieMagnets(ctx, mid)
			if err != nil {
				return pipeline.Envelope{}, fmt.Errorf("magnets failed: %w", err)
			}
			envelope.Data["magnets"] = mags
		}
		return envelope, nil
	}
	runner := &pipeline.BatchRunner{
		Name:       "detail",
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
			return client.WithOptionalAuth(options, streams.Err, func(c *javdb.Client) error {
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
					return fmt.Errorf("detail failed: %w", err)
				}
				var mags []map[string]any
				if withMagnets {
					mags, err = c.MovieMagnets(ctx, mid)
					if err != nil {
						return fmt.Errorf("magnets failed: %w", err)
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
					renderMagnets(streams.Out, streams.Err, mags)
				}
				return nil
			})
		},
	}
	cmd := &cobra.Command{
		Use:   "detail NUMBER",
		Short: "Show movie detail (graph ids for agent navigation)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			runner.Context = cmd.Context()
			return runner.Execute(streams, args, asNDJSON, asJSON)
		},
	}
	cmd.Flags().BoolVarP(&isID, "id", "i", false, "Treat argument as internal movie id")
	cmd.Flags().BoolVar(&withMagnets, "magnets", false, "Also list magnet links")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Machine-readable JSON")
	cmd.Flags().BoolVar(&asNDJSON, "ndjson", false, "Pipeline NDJSON envelopes")
	return cmd
}
