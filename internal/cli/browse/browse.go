// Package browse 提供按内容标签/年份/月份浏览影片命令。
package browse

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/display"
	"github.com/FlanChanXwO/javdb-cli/internal/environment"
	"github.com/FlanChanXwO/javdb-cli/internal/i18n"
	"github.com/FlanChanXwO/javdb-cli/internal/record"
	"github.com/FlanChanXwO/javdb-cli/internal/session"
	javdb "github.com/FlanChanXwO/javdb-cli/sdk"
)

// New builds the tag/year/month movie browsing command.
func New(options *session.Options, streams *environment.Streams) *cobra.Command {
	var (
		zone, year, month, sort, order string
		page, limit                    int
		tagRefs, mainFlags             []string
		hasMagnets                     bool
		asJSON, asNDJSON               bool
	)
	fetch := func(c *session.Client, ctx context.Context) ([]map[string]any, error) {
		var tagIDs []string
		var err error
		if len(tagRefs) > 0 {
			tagIDs, err = c.ResolveTags(ctx, tagRefs, zone)
			if err != nil {
				return nil, err
			}
		}
		res, err := c.Browse(ctx, javdb.BrowseOptions{
			Zone: zone, Main: mainFlags, TagIDs: tagIDs,
			Year: year, Month: month, Sort: sort, Order: order,
			Page: page, Limit: limit,
		})
		if err != nil {
			return nil, err
		}
		movies := res.Movies()
		if hasMagnets {
			movies = display.FilterMoviesWithMagnets(movies)
		}
		return movies, nil
	}
	producer := &record.MovieListProducer{
		Name: "browse",
		ClientFactory: func() (*session.Client, error) {
			return session.NewWithDefaultToken(options)
		},
		Fetch: func(ctx context.Context, c *session.Client) ([]map[string]any, error) {
			return fetch(c, ctx)
		},
		JSON: func(movies []map[string]any) (map[string]any, error) {
			return map[string]any{"movies": movies}, nil
		},
	}
	cmd := &cobra.Command{
		Use:   "browse",
		Short: i18n.T("command.browse"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return producer.Execute(streams, asNDJSON, asJSON)
		},
	}
	cmd.Flags().StringVar(&zone, "zone", "censored", "censored|uncensored|western|fc2")
	cmd.Flags().StringArrayVar(&tagRefs, "tag", nil, i18n.T("flag.tag"))
	cmd.Flags().StringArrayVar(&mainFlags, "main", nil, i18n.T("flag.main"))
	cmd.Flags().StringVar(&year, "year", "", i18n.T("flag.year"))
	cmd.Flags().StringVar(&month, "month", "", i18n.T("flag.month"))
	cmd.Flags().StringVar(&sort, "sort", "hit", "hit|release|score|update|want_watch_count|watched_count")
	cmd.Flags().StringVar(&order, "order", "desc", "asc|desc")
	cmd.Flags().IntVar(&page, "page", 1, i18n.T("flag.page"))
	cmd.Flags().IntVar(&limit, "limit", 20, i18n.T("flag.limit"))
	cmd.Flags().BoolVar(&hasMagnets, "has-magnets", false, i18n.T("flag.has_magnets"))
	cmd.Flags().BoolVar(&asJSON, "json", false, i18n.T("flag.json"))
	cmd.Flags().BoolVar(&asNDJSON, "ndjson", false, i18n.T("flag.ndjson"))
	return cmd
}
