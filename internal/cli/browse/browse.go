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
		Short: i18n.T("cli.browse_movies_by_content_tags_year_month"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return producer.Execute(streams, asNDJSON, asJSON)
		},
	}
	cmd.Flags().StringVar(&zone, "zone", "censored", i18n.T("cli.censored_uncensored_western_fc"))
	cmd.Flags().StringArrayVar(&tagRefs, "tag", nil, i18n.T("cli.tag_id_en_repeatable"))
	cmd.Flags().StringArrayVar(&mainFlags, "main", nil, i18n.T("cli.main_flag_p_m_c_s_i_v_repeatable"))
	cmd.Flags().StringVar(&year, "year", "", i18n.T("cli.four_digit_year"))
	cmd.Flags().StringVar(&month, "month", "", i18n.T("cli.month"))
	cmd.Flags().StringVar(&sort, "sort", "hit", i18n.T("cli.hit_release_score_update_want_watch_count_watched_count"))
	cmd.Flags().StringVar(&order, "order", "desc", i18n.T("cli.asc_desc"))
	cmd.Flags().IntVar(&page, "page", 1, i18n.T("cli.page"))
	cmd.Flags().IntVar(&limit, "limit", 20, i18n.T("cli.page_size"))
	cmd.Flags().BoolVar(&hasMagnets, "has-magnets", false, i18n.T("cli.drop_magnets_count"))
	cmd.Flags().BoolVar(&asJSON, "json", false, i18n.T("cli.json_output"))
	cmd.Flags().BoolVar(&asNDJSON, "ndjson", false, i18n.T("cli.pipeline_ndjson_envelopes"))
	return cmd
}
