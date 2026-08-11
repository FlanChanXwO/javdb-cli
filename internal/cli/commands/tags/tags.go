// Package tags 提供内容标签 taxonomy 命令。
package tags

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/app"
)

// New builds the taxonomy cache command.
func New(flags *app.Flags, aio *app.IO) *cobra.Command {
	var zone string
	var refresh bool
	cmd := &cobra.Command{
		Use:   "tags",
		Short: "List content-tag taxonomy (id + EN + 中文)",
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := app.LoadRuntime(flags)
			if err != nil {
				return err
			}
			c, err := app.NewClient(rt, "")
			if err != nil {
				return err
			}
			doc, path, err := c.LoadOrRefreshTaxonomy(context.Background(), zone, refresh)
			if err != nil {
				return fmt.Errorf("tags failed: %w", err)
			}
			if refresh {
				fmt.Fprintf(aio.Err, "taxonomy 已写入 %s\n", path)
			}
			if doc == nil || len(doc.Categories) == 0 {
				fmt.Fprintln(aio.Err, "(空列表)")
				return nil
			}
			for _, cat := range doc.Categories {
				cname := cat.NameEN
				if cname == "" {
					cname = cat.NameZH
				}
				fmt.Fprintf(aio.Out, "# %s\t%s\n", cat.ID, cname)
				for _, t := range cat.Tags {
					fmt.Fprintf(aio.Out, "%s\t%s\t%s\n", t.ID, t.NameEN, t.NameZH)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&zone, "zone", "censored", "censored|uncensored|western|fc2")
	cmd.Flags().BoolVar(&refresh, "refresh", false, "Re-fetch from API and rewrite local JSON")
	return cmd
}
