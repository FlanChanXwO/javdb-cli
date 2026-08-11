// Package collections 提供用户收藏列表命令。
package collections

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/client"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/result"
	"github.com/FlanChanXwO/javdb-cli/sdk"
)

// New builds the collection listing command.
func New(options *invocation.RootOptions, streams *invocation.Streams) *cobra.Command {
	return &cobra.Command{
		Use:   "collections KIND",
		Short: "List a collection: actors|series|codes|makers|directors",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			kind := args[0]
			return client.WithRequiredAuth(options, streams.Err, func(c *javdb.Client) error {
				items, err := c.Collected(context.Background(), kind)
				if err != nil {
					return err
				}
				return writeNamed(streams.Out, streams.Err, items)
			})
		},
	}
}

// writeNamed 用 entity 投影写出命名实体列表文本；空列表输出 (空列表)。
func writeNamed(w, errW io.Writer, items []map[string]any) error {
	if len(items) == 0 {
		_, err := errW.Write([]byte("(空列表)\n"))
		return err
	}
	for _, row := range result.ProjectNamedAll(items) {
		if _, err := fmt.Fprintln(w, row.Line()); err != nil {
			return err
		}
	}
	return nil
}
