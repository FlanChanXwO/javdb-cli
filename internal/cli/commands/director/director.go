// Package director 提供导演作品列表命令。
package director

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/app"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/entity"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/movie"
	"github.com/FlanChanXwO/javdb-cli/internal/common/jsonx"
)

// New builds the director filmography command.
func New(flags *app.Flags, aio *app.IO) *cobra.Command {
	var (
		zone, sort, order string
		page, limit       int
		tagRefs, main     []string
		allPages          bool
		hasMagnets        bool
		asJSON            bool
	)
	cmd := &cobra.Command{
		Use:   "director REF",
		Short: "List movies for a director (id or name)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := app.LoadRuntime(flags)
			if err != nil {
				return err
			}
			c, err := app.NewClient(rt, "")
			if err != nil {
				return err
			}
			result, err := entity.Execute(context.Background(), c, "director", args[0], entity.Options{
				Zone: zone, Sort: sort, Order: order, Page: page, Limit: limit,
				TagRefs: tagRefs, Main: main, AllPages: allPages, HasMagnets: hasMagnets,
			})
			if err != nil {
				return err
			}
			if asJSON {
				return writeJSON(aio.Out, map[string]any{
					"entity": result.Entity, "entity_id": result.EntityID, "movies": result.Movies,
				})
			}
			return writeMovieRows(aio.Out, aio.Err, result.Movies)
		},
	}
	cmd.Flags().StringVar(&zone, "zone", "censored", "censored|uncensored|western|fc2")
	cmd.Flags().StringArrayVar(&tagRefs, "tag", nil, "Content tag id/EN/中文 (repeatable)")
	cmd.Flags().StringArrayVar(&main, "main", nil, "Main flag p|m|c|s|i|v (repeatable)")
	cmd.Flags().StringVar(&sort, "sort", "release", "hit|release|score|update|want_watch_count|watched_count")
	cmd.Flags().StringVar(&order, "order", "desc", "asc|desc")
	cmd.Flags().IntVar(&page, "page", 1, "Page")
	cmd.Flags().IntVar(&limit, "limit", 20, "Page size")
	cmd.Flags().BoolVar(&allPages, "all", false, "Fetch all pages (capped)")
	cmd.Flags().BoolVar(&hasMagnets, "has-magnets", false, "Drop magnets_count==0")
	cmd.Flags().BoolVar(&asJSON, "json", false, "JSON with entity meta + movies")
	return cmd
}

// writeMovieRows 用 movie 投影写出影片列表文本；空列表输出 (空列表)。
func writeMovieRows(w, errW io.Writer, movies []map[string]any) error {
	if len(movies) == 0 {
		_, err := errW.Write([]byte("(空列表)\n"))
		return err
	}
	for _, row := range movie.ProjectAll(movies) {
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
