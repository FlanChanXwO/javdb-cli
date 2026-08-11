// Package unmark 提供影片取消标记命令。
package unmark

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/app"
	"github.com/FlanChanXwO/javdb-cli/sdk"
)

// New builds the unmark command.
func New(flags *app.Flags, aio *app.IO) *cobra.Command {
	var isID bool
	cmd := &cobra.Command{
		Use:   "unmark NUMBER",
		Short: "Remove watched/want mark for a movie",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.WithAuthedClient(flags, aio, func(c *javdb.Client) error {
				ctx := context.Background()
				mid := args[0]
				var err error
				if !isID {
					mid, err = c.ResolveMovieID(ctx, args[0])
					if err != nil {
						return err
					}
				}
				ok, err := c.Unmark(ctx, mid)
				if err != nil {
					return fmt.Errorf("unmark failed: %w", err)
				}
				if ok {
					fmt.Fprintf(aio.Out, "已取消标记 %s (%s)\n", args[0], mid)
				} else {
					fmt.Fprintf(aio.Out, "无标记可取消 %s (%s)\n", args[0], mid)
				}
				return nil
			})
		},
	}
	cmd.Flags().BoolVarP(&isID, "id", "i", false, "Treat NUMBER as internal movie id")
	return cmd
}
