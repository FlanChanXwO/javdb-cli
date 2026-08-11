// Package detail 提供影片详情与可选磁力输出命令。
package detail

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/app"
	"github.com/FlanChanXwO/javdb-cli/internal/common/jsonx"
	"github.com/FlanChanXwO/javdb-cli/sdk"
)

// New builds the movie detail command (graph ids for agent navigation).
func New(flags *app.Flags, aio *app.IO) *cobra.Command {
	var isID, withMagnets, asJSON bool
	cmd := &cobra.Command{
		Use:   "detail NUMBER",
		Short: "Show movie detail (graph ids for agent navigation)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.WithOptionalAuthClient(flags, aio, func(c *javdb.Client) error {
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
					_, err = aio.Out.Write(b)
					return err
				}
				renderDetail(aio.Out, movie)
				if withMagnets {
					renderMagnets(aio.Out, aio.Err, mags)
				}
				return nil
			})
		},
	}
	cmd.Flags().BoolVarP(&isID, "id", "i", false, "Treat argument as internal movie id")
	cmd.Flags().BoolVar(&withMagnets, "magnets", false, "Also list magnet links")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Machine-readable JSON")
	return cmd
}
