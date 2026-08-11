// Package search 提供影片/实体搜索命令。
package search

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/app"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/entity"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/movie"
	"github.com/FlanChanXwO/javdb-cli/internal/common/jsonx"
	"github.com/FlanChanXwO/javdb-cli/sdk"
)

// New builds the movie and dimension search command.
func New(flags *app.Flags, aio *app.IO) *cobra.Command {
	var (
		page, limit   int
		zone, sort    string
		filterBy, typ string
		hasMagnets    bool
		asJSON        bool
	)
	cmd := &cobra.Command{
		Use:   "search KEYWORD",
		Short: "Search movies (or other dimensions with --type)",
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
			opt := javdb.SearchOptions{
				Page:     page,
				Limit:    limit,
				Zone:     zone,
				Sort:     sort,
				FilterBy: filterBy,
				Type:     typ,
			}
			res, err := c.Search(context.Background(), args[0], opt)
			if err != nil {
				return fmt.Errorf("search failed: %w", err)
			}
			if typ == "" || typ == "movie" {
				movies := res.Movies()
				if hasMagnets {
					movies = movie.FilterHasMagnets(movies)
				}
				if asJSON {
					return writeJSON(aio.Out, map[string]any{"movies": movies})
				}
				return writeMovieRows(aio.Out, aio.Err, movies)
			}
			key := searchTypeKey(typ)
			items := res.Named(key)
			if asJSON {
				return writeJSON(aio.Out, map[string]any{key: items})
			}
			return writeNamedRows(aio.Out, aio.Err, items)
		},
	}
	cmd.Flags().IntVar(&page, "page", 1, "Page number")
	cmd.Flags().IntVar(&limit, "limit", 0, "Page size (0 = server default)")
	cmd.Flags().StringVar(&zone, "zone", "censored", "censored|uncensored|western|fc2|all")
	cmd.Flags().StringVar(&sort, "sort", "", "relevance|release|score|update|hit")
	cmd.Flags().StringVar(&filterBy, "filter-by", "", "can_play|magnets|subtitle|single")
	cmd.Flags().StringVar(&typ, "type", "", "movie|code|series|actor|maker|director|list")
	cmd.Flags().BoolVar(&hasMagnets, "has-magnets", false, "Drop movie rows with magnets_count == 0")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Machine-readable JSON")
	return cmd
}

// searchTypeKey 将 --type 映射为搜索响应 list key（movie/空 → movies）。
func searchTypeKey(type_ string) string {
	switch type_ {
	case "movie", "":
		return "movies"
	case "code":
		return "codes"
	case "series":
		return "series"
	case "actor":
		return "actors"
	case "maker":
		return "makers"
	case "director":
		return "directors"
	case "list":
		return "lists"
	default:
		return "movies"
	}
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

// writeNamedRows 用 entity 投影写出命名实体列表文本；空列表输出 (空列表)。
func writeNamedRows(w, errW io.Writer, items []map[string]any) error {
	if len(items) == 0 {
		_, err := errW.Write([]byte("(空列表)\n"))
		return err
	}
	for _, row := range entity.ProjectNamedAll(items) {
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
