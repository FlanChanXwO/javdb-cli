// Package list 提供合集（用户播放列表）作品列表命令。
package list

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/client"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/entity"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/result"
	"github.com/FlanChanXwO/javdb-cli/internal/common/jsonx"
)

// New builds the list (playlist) filmography command.
func New(options *invocation.RootOptions, streams *invocation.Streams) *cobra.Command {
	var (
		zone, sort, order string
		page, limit       int
		tagRefs, main     []string
		allPages          bool
		hasMagnets        bool
		asJSON            bool
	)
	cmd := &cobra.Command{
		Use:   "list REF",
		Short: "List movies inside a 合集 (user playlist)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.New(options, "")
			if err != nil {
				return err
			}
			result, err := entity.Execute(context.Background(), c, "list", args[0], entity.Options{
				Zone: zone, Sort: sort, Order: order, Page: page, Limit: limit,
				TagRefs: tagRefs, Main: main, AllPages: allPages, HasMagnets: hasMagnets,
			})
			if err != nil {
				return err
			}
			if asJSON {
				return writeJSON(streams.Out, map[string]any{
					"entity": result.Entity, "entity_id": result.EntityID, "movies": result.Movies,
				})
			}
			return writeMovieRows(streams.Out, streams.Err, result.Movies)
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
