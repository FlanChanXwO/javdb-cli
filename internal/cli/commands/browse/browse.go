// Package browse 提供按内容标签/年份/月份浏览影片命令。
package browse

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/client"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/pipeline"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/result"
	javdb "github.com/FlanChanXwO/javdb-cli/sdk"
)

// New builds the tag/year/month movie browsing command.
func New(options *invocation.RootOptions, streams *invocation.Streams) *cobra.Command {
	var (
		zone, year, month, sort, order string
		page, limit                    int
		tagRefs, mainFlags             []string
		hasMagnets                     bool
		asJSON, asJSONL, asText        bool
	)
	fetch := func(c *javdb.Client, ctx context.Context) ([]map[string]any, error) {
		var tagIDs []string
		var err error
		if len(tagRefs) > 0 {
			tagIDs, err = c.ResolveTags(ctx, tagRefs, zone)
			if err != nil {
				return nil, fmt.Errorf("browse failed: %w", err)
			}
		}
		res, err := c.Browse(ctx, javdb.BrowseOptions{
			Zone: zone, Main: mainFlags, TagIDs: tagIDs,
			Year: year, Month: month, Sort: sort, Order: order,
			Page: page, Limit: limit,
		})
		if err != nil {
			return nil, fmt.Errorf("browse failed: %w", err)
		}
		movies := res.Movies()
		if hasMagnets {
			movies = result.FilterMoviesWithMagnets(movies)
		}
		return movies, nil
	}
	producer := &pipeline.MovieListProducer{
		Name: "browse",
		ClientFactory: func() (*javdb.Client, error) {
			return client.New(options, "")
		},
		Fetch: func(ctx context.Context, c *javdb.Client) ([]map[string]any, error) {
			return fetch(c, ctx)
		},
		JSON: func(movies []map[string]any) (map[string]any, error) {
			return map[string]any{"movies": movies}, nil
		},
	}
	cmd := &cobra.Command{
		Use:   "browse",
		Short: "Browse movies by content tags / year / month",
		RunE: func(cmd *cobra.Command, args []string) error {
			return producer.Execute(streams, asJSONL, asText, asJSON)
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
	cmd.Flags().BoolVar(&asJSONL, "jsonl", false, "Pipeline JSONL envelopes")
	cmd.Flags().BoolVar(&asText, "text", false, "Plain text lines (default)")
	return cmd
}
