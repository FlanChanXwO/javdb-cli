// Package browse 提供按内容标签/年份/月份浏览影片命令。
package browse

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/client"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/result"
	"github.com/FlanChanXwO/javdb-cli/internal/common/jsonx"
	"github.com/FlanChanXwO/javdb-cli/sdk"
)

// New builds the tag/year/month movie browsing command.
func New(options *invocation.RootOptions, streams *invocation.Streams) *cobra.Command {
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
			c, err := client.New(options, "")
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
			res, err := c.Browse(ctx, javdb.BrowseOptions{
				Zone: zone, Main: mainFlags, TagIDs: tagIDs,
				Year: year, Month: month, Sort: sort, Order: order,
				Page: page, Limit: limit,
			})
			if err != nil {
				return fmt.Errorf("browse failed: %w", err)
			}
			movies := res.Movies()
			if hasMagnets {
				movies = result.FilterMoviesWithMagnets(movies)
			}
			if asJSON {
				return writeJSON(streams.Out, map[string]any{"movies": movies})
			}
			return writeMovieRows(streams.Out, streams.Err, movies)
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

// writeMovieRows 用 movie 投影写出影片列表文本；空列表输出 (空列表)。
func writeMovieRows(w, errW io.Writer, movies []map[string]any) error {
	if len(movies) == 0 {
		_, err := errW.Write([]byte("(空列表)\n"))
		return err
	}
	for _, row := range result.ProjectMovies(movies) {
		if _, err := fmt.Fprintln(w, row.Line()); err != nil {
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
