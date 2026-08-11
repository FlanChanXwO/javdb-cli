// Package mark 提供影片标记命令。
package mark

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/client"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/common/scalar"
	"github.com/FlanChanXwO/javdb-cli/sdk"
)

// New builds the mark command (watched or want).
func New(options *invocation.RootOptions, streams *invocation.Streams) *cobra.Command {
	var watched, want, isID bool
	var score int
	var content string
	cmd := &cobra.Command{
		Use:   "mark NUMBER",
		Short: "Mark a movie as 看過 (--watched) or 想看 (--want)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if watched == want {
				return fmt.Errorf("specify exactly one of --watched or --want")
			}
			status := "want_watch"
			label := "想看"
			if watched {
				status = "watched"
				label = "看過"
			}
			return client.WithRequiredAuth(options, streams.Err, func(c *javdb.Client) error {
				ctx := context.Background()
				mid := args[0]
				var err error
				if !isID {
					mid, err = c.ResolveMovieID(ctx, args[0])
					if err != nil {
						return err
					}
				}
				rev, err := c.Mark(ctx, mid, status, score, content)
				if err != nil {
					return fmt.Errorf("mark failed: %w", err)
				}
				_, err = fmt.Fprintf(streams.Out, "已标记 %s (%s) → %s\treview_id=%s\n",
					args[0], mid, label, display(rev["id"]))
				return err
			})
		},
	}
	cmd.Flags().BoolVar(&watched, "watched", false, "Mark as 看過")
	cmd.Flags().BoolVar(&want, "want", false, "Mark as 想看")
	cmd.Flags().IntVar(&score, "score", 0, "Optional score")
	cmd.Flags().StringVar(&content, "content", "", "Optional review text")
	cmd.Flags().BoolVarP(&isID, "id", "i", false, "Treat NUMBER as internal movie id")
	return cmd
}

func display(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatInt(int64(t), 10)
	default:
		return scalar.String(t)
	}
}
