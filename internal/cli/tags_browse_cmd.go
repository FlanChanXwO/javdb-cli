package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/appapi"
	"github.com/FlanChanXwO/javdb-cli/internal/storage/tags"
)

func newTagsCmd(rf *rootFlags, aio *appIO) *cobra.Command {
	var zone string
	var refresh bool
	cmd := &cobra.Command{
		Use:   "tags",
		Short: "List content-tag taxonomy (id + EN + 中文)",
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime(rf)
			if err != nil {
				return err
			}
			c, err := newClient(rt, "")
			if err != nil {
				return err
			}
			doc, path, err := c.LoadOrRefreshTaxonomy(context.Background(), zone, refresh)
			if err != nil {
				return fmt.Errorf("tags failed: %w", err)
			}
			if refresh {
				fmt.Fprintf(aio.err, "taxonomy 已写入 %s\n", path)
			}
			if doc == nil || len(doc.Categories) == 0 {
				fmt.Fprintln(aio.err, "(空列表)")
				return nil
			}
			for _, cat := range doc.Categories {
				cname := cat.NameEN
				if cname == "" {
					cname = cat.NameZH
				}
				fmt.Fprintf(aio.out, "# %s\t%s\n", cat.ID, cname)
				for _, t := range cat.Tags {
					fmt.Fprintf(aio.out, "%s\t%s\t%s\n", t.ID, t.NameEN, t.NameZH)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&zone, "zone", "censored", "censored|uncensored|western|fc2")
	cmd.Flags().BoolVar(&refresh, "refresh", false, "Re-fetch from API and rewrite local JSON")
	return cmd
}

func newBrowseCmd(rf *rootFlags, aio *appIO) *cobra.Command {
	var (
		zone, year, month, sort, order string
		page, limit                    int
		tagRefs, mainFlags             []string
		hasMagnets, asJSON             bool
	)
	cmd := &cobra.Command{
		Use:   "browse",
		Short: "Browse movies by content tags / year / month",
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime(rf)
			if err != nil {
				return err
			}
			c, err := newClient(rt, "")
			if err != nil {
				return err
			}
			ctx := context.Background()
			var tagIDs []string
			if len(tagRefs) > 0 {
				tagIDs, err = c.ResolveTags(ctx, tagRefs, zone)
				if err != nil {
					return fmt.Errorf("browse failed: %w", err)
				}
			}
			res, err := c.Browse(ctx, appapi.BrowseOptions{
				Zone: zone, Main: mainFlags, TagIDs: tagIDs,
				Year: year, Month: month, Sort: sort, Order: order,
				Page: page, Limit: limit,
			})
			if err != nil {
				return fmt.Errorf("browse failed: %w", err)
			}
			movies := res.Movies()
			if hasMagnets {
				movies = FilterHasMagnets(movies)
			}
			if asJSON {
				return EmitJSON(aio.out, map[string]any{"movies": movies})
			}
			PrintMovies(aio.out, aio.err, movies)
			return nil
		},
	}
	cmd.Flags().StringVar(&zone, "zone", "censored", "censored|uncensored|western|fc2")
	cmd.Flags().StringArrayVar(&tagRefs, "tag", nil, "Tag id/EN/中文 (repeatable)")
	cmd.Flags().StringArrayVar(&mainFlags, "main", nil, "Main flag p|m|c|s|i|v (repeatable)")
	cmd.Flags().StringVar(&year, "year", "", "Four-digit year")
	cmd.Flags().StringVar(&month, "month", "", "Month 1..12")
	cmd.Flags().StringVar(&sort, "sort", "hit", "hit|release|score|update|want_watch_count|watched_count")
	cmd.Flags().StringVar(&order, "order", "desc", "asc|desc")
	cmd.Flags().IntVar(&page, "page", 1, "Page")
	cmd.Flags().IntVar(&limit, "limit", 20, "Page size")
	cmd.Flags().BoolVar(&hasMagnets, "has-magnets", false, "Drop magnets_count==0")
	cmd.Flags().BoolVar(&asJSON, "json", false, "JSON output")
	return cmd
}

// ensure tags package referenced for tests helpers if needed
var _ = tags.AliasMap
var _ = strings.Join
