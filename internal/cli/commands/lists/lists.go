// Package lists 提供我的合集与公开合集命令组。
package lists

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/app"
	"github.com/FlanChanXwO/javdb-cli/internal/common/jsonx"
	"github.com/FlanChanXwO/javdb-cli/sdk"
)

// New builds the personal and public list command tree.
func New(flags *app.Flags, aio *app.IO) *cobra.Command {
	var page, limit int
	var sortBy string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "lists",
		Short: "My 合集; subcommands: show/search/related",
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.WithAuthedClient(flags, aio, func(c *javdb.Client) error {
				res, err := c.MyLists(context.Background(), page, limit, sortBy)
				if err != nil {
					return fmt.Errorf("lists failed: %w", err)
				}
				items := res.Named("lists")
				if asJSON {
					return writeJSON(aio.Out, map[string]any{
						"lists":        items,
						"current_page": jsonx.RawString(res["current_page"]),
					})
				}
				return writeListRows(aio.Out, aio.Err, items)
			})
		},
	}
	cmd.Flags().IntVar(&page, "page", 1, "Page")
	cmd.Flags().IntVar(&limit, "limit", 20, "Page size")
	cmd.Flags().StringVar(&sortBy, "sort-by", "created", "created|name|movies_count|views_count|updated|default")
	cmd.Flags().BoolVar(&asJSON, "json", false, "JSON output")

	cmd.AddCommand(NewShow(flags, aio))
	cmd.AddCommand(NewSearch(flags, aio))
	cmd.AddCommand(NewRelated(flags, aio))
	return cmd
}

// writeListRows 写出 id\tname\tmovies\tprivacy\tviews 行；空列表输出 (空列表)。
func writeListRows(w, errW io.Writer, items []map[string]any) error {
	if len(items) == 0 {
		_, err := errW.Write([]byte("(空列表)\n"))
		return err
	}
	for _, item := range items {
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			display(item["id"]), display(item["name"]), display(item["movies_count"]),
			display(item["privacy"]), display(item["views_count"])); err != nil {
			return err
		}
	}
	return nil
}

// writeJSON 以 jsonx.MarshalLine 写出紧凑 JSON 并传播编码与写入错误。
func writeJSON(w io.Writer, value any) error {
	b, err := jsonx.MarshalLine(value)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}
